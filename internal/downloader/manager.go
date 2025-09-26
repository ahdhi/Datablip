package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusCompleted   DownloadStatus = "completed"
	StatusError       DownloadStatus = "error"
)

type Download struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	Filename       string         `json:"filename"`
	OutputPath     string         `json:"outputPath"`
	Status         DownloadStatus `json:"status"`
	Progress       float64        `json:"progress"`
	TotalSize      int64          `json:"totalSize"`
	Downloaded     int64          `json:"downloaded"`
	Speed          float64        `json:"speed"`
	Chunks         int            `json:"chunks"`
	ChunkProgress  []float64      `json:"chunkProgress"`
	TimeRemaining  int            `json:"timeRemaining"`
	StartTime      time.Time      `json:"startTime"`
	Error          string         `json:"error,omitempty"`
	ConnectTimeout string         `json:"connectTimeout"`
	ReadTimeout    string         `json:"readTimeout"`

	mu             sync.RWMutex
	pauseChan      chan bool
	lastDownloaded int64
	lastUpdateTime time.Time
}

type Manager struct {
	downloads map[string]*Download
	mu        sync.RWMutex
	listeners []chan DownloadUpdate
}

type DownloadUpdate struct {
	DownloadID string      `json:"downloadId"`
	Type       string      `json:"type"`
	Data       interface{} `json:"data"`
}

func NewManager() *Manager {
	return &Manager{
		downloads: make(map[string]*Download),
		listeners: make([]chan DownloadUpdate, 0),
	}
}

func (m *Manager) AddDownload(url, filename string, chunks int, connectTimeout, readTimeout string) (*Download, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set output path in downloads directory
	outputPath := fmt.Sprintf("downloads/%s", filename)
	if filename == "" {
		outputPath = fmt.Sprintf("downloads/download_%s", generateID())
	}

	download := &Download{
		ID:             generateID(),
		URL:            url,
		Filename:       filename,
		OutputPath:     outputPath,
		Status:         StatusPending,
		Chunks:         chunks,
		ChunkProgress:  make([]float64, chunks),
		ConnectTimeout: connectTimeout,
		ReadTimeout:    readTimeout,
		StartTime:      time.Now(),
		pauseChan:      make(chan bool),
		lastDownloaded: 0,
		lastUpdateTime: time.Now(),
	}

	m.downloads[download.ID] = download

	// Start download in goroutine
	go m.startDownload(download)

	return download, nil
}

func (m *Manager) startDownload(d *Download) {
	d.Status = StatusDownloading
	m.broadcastUpdate(DownloadUpdate{
		DownloadID: d.ID,
		Type:       "status",
		Data:       d,
	})

	// Get file size and check if server supports range requests
	resp, err := http.Head(d.URL)
	if err != nil {
		d.Status = StatusError
		d.Error = err.Error()
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "error",
			Data:       d,
		})
		return
	}
	d.TotalSize = resp.ContentLength

	// Check if server supports range requests
	supportsRanges := resp.Header.Get("Accept-Ranges") == "bytes"
	fmt.Printf("Server supports range requests: %v\n", supportsRanges)
	fmt.Printf("Total file size: %d bytes\n", d.TotalSize)

	if !supportsRanges || d.Chunks == 1 {
		// Download as single file
		fmt.Printf("Downloading as single file (no chunking)\n")
		m.downloadSingleFile(d)
		return
	}

	// Create chunks and download
	chunkSize := d.TotalSize / int64(d.Chunks)
	var wg sync.WaitGroup
	errorChan := make(chan error, d.Chunks)

	fmt.Printf("Starting chunked download with %d chunks of %d bytes each\n", d.Chunks, chunkSize)

	// Start progress updater goroutine
	go m.updateProgress(d)

	for i := 0; i < d.Chunks; i++ {
		wg.Add(1)
		go func(chunkIndex int) {
			defer wg.Done()
			err := m.downloadChunk(d, chunkIndex, chunkSize)
			if err != nil {
				errorChan <- fmt.Errorf("chunk %d failed: %v", chunkIndex, err)
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for chunk errors
	var chunkErrors []string
	for err := range errorChan {
		chunkErrors = append(chunkErrors, err.Error())
	}

	if len(chunkErrors) > 0 {
		d.Status = StatusError
		d.Error = fmt.Sprintf("Some chunks failed: %v", chunkErrors)
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "error",
			Data:       d,
		})
		return
	}

	// Merge chunks
	if d.Status == StatusDownloading {
		fmt.Printf("All chunks downloaded successfully, merging files...\n")
		err := m.mergeChunks(d)
		if err != nil {
			d.Status = StatusError
			d.Error = err.Error()
			m.broadcastUpdate(DownloadUpdate{
				DownloadID: d.ID,
				Type:       "error",
				Data:       d,
			})
			return
		}

		d.Status = StatusCompleted
		d.Progress = 100
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "completed",
			Data:       d,
		})
	}
}

func (m *Manager) downloadChunk(d *Download, chunkIndex int, chunkSize int64) error {
	startByte := int64(chunkIndex) * chunkSize
	endByte := startByte + chunkSize - 1

	if chunkIndex == d.Chunks-1 {
		endByte = d.TotalSize - 1
	}

	actualChunkSize := endByte - startByte + 1

	fmt.Printf("Downloading chunk %d: bytes %d-%d (%d bytes)\n", chunkIndex, startByte, endByte, actualChunkSize)

	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil {
		return fmt.Errorf("error creating request for chunk %d: %v", chunkIndex, err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading chunk %d: %v", chunkIndex, err)
	}
	defer resp.Body.Close()

	// Check if server supports range requests
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server doesn't support range requests for chunk %d, status: %d", chunkIndex, resp.StatusCode)
	}

	// Create temp file for chunk with specific naming
	tempFileName := fmt.Sprintf("chunk_%s_%d.tmp", d.ID, chunkIndex)
	tempFile, err := os.Create(tempFileName)
	if err != nil {
		return fmt.Errorf("error creating temp file for chunk %d: %v", chunkIndex, err)
	}
	defer tempFile.Close()

	// Copy with progress tracking
	buffer := make([]byte, 32*1024)
	var downloaded int64

downloadLoop:
	for {
		select {
		case <-d.pauseChan:
			// Handle pause
			<-d.pauseChan // Wait for resume
		default:
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				return fmt.Errorf("error reading chunk %d: %v", chunkIndex, err)
			}
			if n == 0 {
				break downloadLoop
			}

			_, writeErr := tempFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("error writing chunk %d: %v", chunkIndex, writeErr)
			}
			downloaded += int64(n)

			// Update chunk progress
			d.mu.Lock()
			d.ChunkProgress[chunkIndex] = float64(downloaded) / float64(actualChunkSize) * 100
			d.mu.Unlock()

			// Send immediate progress update for chunk progress
			if downloaded%1048576 == 0 || err == io.EOF { // Update every 1MB or at end
				m.broadcastUpdate(DownloadUpdate{
					DownloadID: d.ID,
					Type:       "progress",
					Data:       d,
				})
			}

			if err == io.EOF {
				break downloadLoop
			}
		}
	}

	// Verify we downloaded the expected amount
	if downloaded != actualChunkSize {
		return fmt.Errorf("chunk %d incomplete: expected %d bytes, got %d bytes", chunkIndex, actualChunkSize, downloaded)
	}

	fmt.Printf("Chunk %d completed successfully: %d bytes downloaded\n", chunkIndex, downloaded)

	// Send immediate progress update when chunk completes
	m.broadcastUpdate(DownloadUpdate{
		DownloadID: d.ID,
		Type:       "progress",
		Data:       d,
	})

	return nil
}

func (m *Manager) downloadSingleFile(d *Download) {
	// Create downloads directory if it doesn't exist
	os.MkdirAll("downloads", 0755)

	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil {
		d.Status = StatusError
		d.Error = err.Error()
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "error",
			Data:       d,
		})
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.Status = StatusError
		d.Error = err.Error()
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "error",
			Data:       d,
		})
		return
	}
	defer resp.Body.Close()

	// Create the output file
	outputFile, err := os.Create(d.OutputPath)
	if err != nil {
		d.Status = StatusError
		d.Error = err.Error()
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "error",
			Data:       d,
		})
		return
	}
	defer outputFile.Close()

	fmt.Printf("Downloading single file: %s\n", d.Filename)

	// Copy with progress tracking
	buffer := make([]byte, 32*1024)
	var downloaded int64

	// Start progress updater for single file download
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if d.Status != StatusDownloading {
				return
			}

			d.mu.Lock()
			if d.TotalSize > 0 {
				d.Progress = float64(downloaded) / float64(d.TotalSize) * 100
				d.Downloaded = downloaded
			}
			d.mu.Unlock()

			m.broadcastUpdate(DownloadUpdate{
				DownloadID: d.ID,
				Type:       "progress",
				Data:       d,
			})
		}
	}()

downloadLoop:
	for {
		select {
		case <-d.pauseChan:
			// Handle pause
			<-d.pauseChan // Wait for resume
		default:
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				d.Status = StatusError
				d.Error = err.Error()
				m.broadcastUpdate(DownloadUpdate{
					DownloadID: d.ID,
					Type:       "error",
					Data:       d,
				})
				return
			}
			if n == 0 {
				break downloadLoop
			}

			_, writeErr := outputFile.Write(buffer[:n])
			if writeErr != nil {
				d.Status = StatusError
				d.Error = writeErr.Error()
				m.broadcastUpdate(DownloadUpdate{
					DownloadID: d.ID,
					Type:       "error",
					Data:       d,
				})
				return
			}
			downloaded += int64(n)

			if err == io.EOF {
				break downloadLoop
			}
		}
	}

	d.Status = StatusCompleted
	d.Progress = 100
	d.Downloaded = downloaded
	fmt.Printf("Single file download completed: %d bytes\n", downloaded)

	m.broadcastUpdate(DownloadUpdate{
		DownloadID: d.ID,
		Type:       "completed",
		Data:       d,
	})
}

func (m *Manager) mergeChunks(d *Download) error {
	// Create downloads directory if it doesn't exist
	os.MkdirAll("downloads", 0755)

	// Create the final output file
	outputFile, err := os.Create(d.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	fmt.Printf("Merging %d chunks for download %s\n", d.Chunks, d.ID)

	var totalMerged int64

	// Merge all chunk files in order
	for i := 0; i < d.Chunks; i++ {
		chunkFileName := fmt.Sprintf("chunk_%s_%d.tmp", d.ID, i)

		chunkFile, err := os.Open(chunkFileName)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %d: %v", i, err)
		}

		// Copy chunk content to output file
		copied, err := io.Copy(outputFile, chunkFile)
		chunkFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %v", i, err)
		}

		totalMerged += copied

		// Remove temporary chunk file
		os.Remove(chunkFileName)

		fmt.Printf("Merged chunk %d/%d (%d bytes)\n", i+1, d.Chunks, copied)
	}

	// Verify total size
	if totalMerged != d.TotalSize {
		return fmt.Errorf("merged file size mismatch: expected %d bytes, got %d bytes", d.TotalSize, totalMerged)
	}

	fmt.Printf("Successfully merged all chunks for download %s (%d bytes total)\n", d.ID, totalMerged)
	return nil
}

func (m *Manager) PauseDownload(id string) error {
	m.mu.RLock()
	download, exists := m.downloads[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("download not found")
	}

	if download.Status == StatusDownloading {
		download.Status = StatusPaused
		download.pauseChan <- true
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: id,
			Type:       "paused",
			Data:       download,
		})
	}

	return nil
}

func (m *Manager) ResumeDownload(id string) error {
	m.mu.RLock()
	download, exists := m.downloads[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("download not found")
	}

	if download.Status == StatusPaused {
		download.Status = StatusDownloading
		download.pauseChan <- false
		m.broadcastUpdate(DownloadUpdate{
			DownloadID: id,
			Type:       "resumed",
			Data:       download,
		})
	}

	return nil
}

func (m *Manager) Subscribe() chan DownloadUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan DownloadUpdate, 100)
	m.listeners = append(m.listeners, ch)
	return ch
}

func (m *Manager) broadcastUpdate(update DownloadUpdate) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, listener := range m.listeners {
		select {
		case listener <- update:
		default:
			// Skip if channel is full
		}
	}
}

func (m *Manager) GetAllDownloads() []*Download {
	m.mu.RLock()
	defer m.mu.RUnlock()

	downloads := make([]*Download, 0, len(m.downloads))
	for _, download := range m.downloads {
		downloads = append(downloads, download)
	}
	return downloads
}

func (m *Manager) GetDownload(id string) (*Download, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	download, exists := m.downloads[id]
	if !exists {
		return nil, fmt.Errorf("download not found")
	}
	return download, nil
}

func (m *Manager) updateProgress(d *Download) {
	ticker := time.NewTicker(250 * time.Millisecond) // Update 4 times per second
	defer ticker.Stop()

	for tick := range ticker.C {
		_ = tick // Use the tick variable to avoid unused variable warning
		if d.Status != StatusDownloading {
			return
		}

		d.mu.Lock()
		totalProgress := 0.0
		for _, chunkProgress := range d.ChunkProgress {
			totalProgress += chunkProgress
		}
		d.Progress = totalProgress / float64(d.Chunks)
		d.Downloaded = int64(float64(d.TotalSize) * d.Progress / 100)

		// Calculate speed
		now := time.Now()
		timeDiff := now.Sub(d.lastUpdateTime).Seconds()
		if timeDiff > 0 {
			bytesDiff := d.Downloaded - d.lastDownloaded
			d.Speed = float64(bytesDiff) / timeDiff
			d.lastDownloaded = d.Downloaded
			d.lastUpdateTime = now

			// Calculate time remaining
			if d.Speed > 0 {
				remainingBytes := d.TotalSize - d.Downloaded
				d.TimeRemaining = int(float64(remainingBytes) / d.Speed)
			}
		}

		d.mu.Unlock()

		m.broadcastUpdate(DownloadUpdate{
			DownloadID: d.ID,
			Type:       "progress",
			Data:       d,
		})
	}
}

func (m *Manager) DeleteDownload(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	download, exists := m.downloads[id]
	if !exists {
		return fmt.Errorf("download not found")
	}

	// Cancel the download if it's in progress
	if download.Status == StatusDownloading {
		download.Status = StatusError
		download.Error = "Download cancelled"

		// Clean up any temporary chunk files
		for i := 0; i < download.Chunks; i++ {
			chunkFileName := fmt.Sprintf("chunk_%s_%d.tmp", download.ID, i)
			os.Remove(chunkFileName)
		}
	}

	delete(m.downloads, id)
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
