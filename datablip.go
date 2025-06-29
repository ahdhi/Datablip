package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultConnectTimeout = 30 * time.Second // Connection timeout
	DefaultReadTimeout    = 5 * time.Minute  // Per-chunk read timeout
	DefaultConnections    = 4
	ProgressBarWidth      = 30
)

type ChunkInfo struct {
	ID        int
	StartByte int64
	EndByte   int64
	Size      int64
}

// ChunkProgress tracks individual chunk download progress
type ChunkProgress struct {
	ID              int
	downloadedBytes int64
	totalBytes      int64
	startTime       time.Time
	status          string // "waiting", "downloading", "completed", "failed"
	speed           float64
	mu              sync.RWMutex
}

func NewChunkProgress(id int, totalBytes int64) *ChunkProgress {
	return &ChunkProgress{
		ID:         id,
		totalBytes: totalBytes,
		startTime:  time.Now(),
		status:     "waiting",
	}
}

func (cp *ChunkProgress) SetStatus(status string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.status = status
	if status == "downloading" && cp.startTime.IsZero() {
		cp.startTime = time.Now()
	}
}

func (cp *ChunkProgress) AddBytes(bytes int64) {
	atomic.AddInt64(&cp.downloadedBytes, bytes)

	// Update speed calculation
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.status == "downloading" {
		elapsed := time.Since(cp.startTime).Seconds()
		if elapsed > 0 {
			cp.speed = float64(atomic.LoadInt64(&cp.downloadedBytes)) / elapsed
		}
	}
}

func (cp *ChunkProgress) GetProgress() (downloaded, total int64, percentage float64, speed float64, status string) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	downloaded = atomic.LoadInt64(&cp.downloadedBytes)
	total = cp.totalBytes
	if total > 0 {
		percentage = float64(downloaded) / float64(total) * 100
	}
	speed = cp.speed
	status = cp.status
	return
}

// ProgressManager manages all chunk progress tracking
type ProgressManager struct {
	chunkProgresses []*ChunkProgress
	totalSize       int64
	startTime       time.Time
	mu              sync.RWMutex
}

func NewProgressManager(chunks []ChunkInfo) *ProgressManager {
	pm := &ProgressManager{
		chunkProgresses: make([]*ChunkProgress, len(chunks)),
		startTime:       time.Now(),
	}

	for i, chunk := range chunks {
		pm.chunkProgresses[i] = NewChunkProgress(i, chunk.Size)
		pm.totalSize += chunk.Size
	}

	return pm
}

func (pm *ProgressManager) GetChunkProgress(chunkID int) *ChunkProgress {
	if chunkID >= 0 && chunkID < len(pm.chunkProgresses) {
		return pm.chunkProgresses[chunkID]
	}
	return nil
}

func (pm *ProgressManager) GetOverallProgress() (downloaded, total int64, percentage float64, speed float64) {
	var totalDownloaded int64

	for _, cp := range pm.chunkProgresses {
		downloaded, _, _, _, _ := cp.GetProgress()
		totalDownloaded += downloaded
	}

	total = pm.totalSize
	if total > 0 {
		percentage = float64(totalDownloaded) / float64(total) * 100
	}

	elapsed := time.Since(pm.startTime).Seconds()
	if elapsed > 0 {
		speed = float64(totalDownloaded) / elapsed
	}

	return totalDownloaded, total, percentage, speed
}

func (pm *ProgressManager) FormatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
}

func (pm *ProgressManager) FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func (pm *ProgressManager) DisplayProgress() {
	// Move cursor to the top-left of the display area (no screen clear)
	fmt.Print("\033[H")

	// Display overall progress
	downloaded, total, percentage, speed := pm.GetOverallProgress()
	overallCompleted := int(float64(ProgressBarWidth) * percentage / 100)
	overallRemaining := ProgressBarWidth - overallCompleted
	progressBar := "[" + strings.Repeat("=", overallCompleted) + strings.Repeat("-", overallRemaining) + "]"

	fmt.Printf("Overall Progress:\n")
	fmt.Printf("%s %.1f%% (%s/%s) %s\n\n",
		progressBar,
		percentage,
		pm.FormatSize(downloaded),
		pm.FormatSize(total),
		pm.FormatSpeed(speed))

	// Display individual chunk progress
	fmt.Printf("Individual Chunks:\n")
	fmt.Printf("%-8s %-12s %-32s %-12s %-10s %s\n",
		"Chunk", "Status", "Progress", "Downloaded", "Speed", "ETA")
	fmt.Printf("%s\n", strings.Repeat("-", 85))

	for _, cp := range pm.chunkProgresses {
		downloaded, total, percentage, speed, status := cp.GetProgress()

		// Create mini progress bar for chunk
		chunkCompleted := int(float64(20) * percentage / 100)
		chunkRemaining := 20 - chunkCompleted
		chunkBar := "[" + strings.Repeat("=", chunkCompleted) + strings.Repeat("-", chunkRemaining) + "]"

		// Calculate ETA
		eta := "∞"
		if speed > 0 && status == "downloading" {
			remainingBytes := total - downloaded
			etaSeconds := float64(remainingBytes) / speed
			if etaSeconds > 0 && etaSeconds < 3600 { // Only show if less than 1 hour
				eta = fmt.Sprintf("%.0fs", etaSeconds)
			}
		}

		// Status color coding (using ANSI colors)
		statusColor := ""
		statusReset := "\033[0m"
		switch status {
		case "waiting":
			statusColor = "\033[33m" // Yellow
		case "downloading":
			statusColor = "\033[36m" // Cyan
		case "completed":
			statusColor = "\033[32m" // Green
		case "failed":
			statusColor = "\033[31m" // Red
		}

		fmt.Printf("%-8d %s%-12s%s %-32s %-12s %-10s %s\n",
			cp.ID,
			statusColor, status, statusReset,
			fmt.Sprintf("%s %.1f%%", chunkBar, percentage),
			pm.FormatSize(downloaded),
			pm.FormatSpeed(speed),
			eta)
	}

	// Show active/completed/failed counts
	var waiting, downloading, completed, failed int
	for _, cp := range pm.chunkProgresses {
		_, _, _, _, status := cp.GetProgress()
		switch status {
		case "waiting":
			waiting++
		case "downloading":
			downloading++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	fmt.Printf("\nStatus Summary: ")
	fmt.Printf("\033[33mWaiting: %d\033[0m, ", waiting)
	fmt.Printf("\033[36mDownloading: %d\033[0m, ", downloading)
	fmt.Printf("\033[32mCompleted: %d\033[0m, ", completed)
	fmt.Printf("\033[31mFailed: %d\033[0m\n", failed)
}

type MergeProgress struct {
	totalSize   int64
	mergedBytes int64
	startTime   time.Time
	mu          sync.RWMutex
}

func (mp *MergeProgress) AddBytes(bytes int64) {
	atomic.AddInt64(&mp.mergedBytes, bytes)
}

func (mp *MergeProgress) GetProgress() (merged, total int64, percentage float64, speed float64) {
	merged = atomic.LoadInt64(&mp.mergedBytes)
	total = mp.totalSize
	percentage = float64(merged) / float64(total) * 100

	elapsed := time.Since(mp.startTime).Seconds()
	if elapsed > 0 {
		speed = float64(merged) / elapsed
	}

	return
}

type MergeProgressReader struct {
	reader   io.Reader
	progress *MergeProgress
}

func (mpr *MergeProgressReader) Read(p []byte) (n int, err error) {
	n, err = mpr.reader.Read(p)
	if n > 0 {
		mpr.progress.AddBytes(int64(n))
	}
	return
}

type ChunkProgressReader struct {
	reader        io.Reader
	chunkProgress *ChunkProgress
}

func (cpr *ChunkProgressReader) Read(p []byte) (n int, err error) {
	n, err = cpr.reader.Read(p)
	if n > 0 {
		cpr.chunkProgress.AddBytes(int64(n))
	}
	return
}

type Downloader struct {
	URL             string
	OutputPath      string
	Chunks          int
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	client          *http.Client
	progressManager *ProgressManager
}

func NewDownloader(url, outputPath string, chunks int) *Downloader {
	return &Downloader{
		URL:            url,
		OutputPath:     outputPath,
		Chunks:         chunks,
		ConnectTimeout: DefaultConnectTimeout,
		ReadTimeout:    DefaultReadTimeout,
		client: &http.Client{
			Timeout: DefaultConnectTimeout,
		},
	}
}

func (d *Downloader) SetTimeouts(connectTimeout, readTimeout time.Duration) {
	d.ConnectTimeout = connectTimeout
	d.ReadTimeout = readTimeout
	d.client.Timeout = connectTimeout
}

func (d *Downloader) getFileSize() (int64, error) {
	fmt.Printf("Getting file information from: %s\n", d.URL)

	resp, err := d.client.Head(d.URL)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	size := resp.ContentLength
	if size <= 0 {
		return 0, fmt.Errorf("could not determine file size or server doesn't support range requests")
	}

	return size, nil
}

func (d *Downloader) createChunks(fileSize int64) []ChunkInfo {
	var chunks []ChunkInfo
	chunkSize := fileSize / int64(d.Chunks)

	for i := 0; i < d.Chunks; i++ {
		startByte := int64(i) * chunkSize
		endByte := startByte + chunkSize - 1

		if i == d.Chunks-1 {
			endByte = fileSize - 1
		}

		chunkInfo := ChunkInfo{
			ID:        i,
			StartByte: startByte,
			EndByte:   endByte,
			Size:      endByte - startByte + 1,
		}

		chunks = append(chunks, chunkInfo)
	}

	return chunks
}

func (d *Downloader) downloadChunk(chunk ChunkInfo, outputFile string) error {
	chunkProgress := d.progressManager.GetChunkProgress(chunk.ID)
	chunkProgress.SetStatus("downloading")

	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("failed to create request: %w", err)
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.StartByte, chunk.EndByte)
	req.Header.Set("Range", rangeHeader)
	req.Header.Set("User-Agent", "MultiPartDownloader/1.0")

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: d.ConnectTimeout,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("failed to make request for chunk %d: %w", chunk.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("chunk %d: server returned status code %d", chunk.ID, resp.StatusCode)
	}

	output, err := os.Create(outputFile)
	if err != nil {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("failed to create output file for chunk %d: %w", chunk.ID, err)
	}
	defer output.Close()

	progressReader := &ChunkProgressReader{
		reader:        resp.Body,
		chunkProgress: chunkProgress,
	}

	written, err := d.copyWithActivityTimeout(output, progressReader, d.ReadTimeout)
	if err != nil {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("failed to write data for chunk %d: %w", chunk.ID, err)
	}

	if resp.StatusCode == http.StatusPartialContent && abs(written-chunk.Size) > 1024 {
		chunkProgress.SetStatus("failed")
		return fmt.Errorf("chunk %d: expected %d bytes, got %d bytes (difference: %d)",
			chunk.ID, chunk.Size, written, abs(written-chunk.Size))
	}

	chunkProgress.SetStatus("completed")
	return nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (d *Downloader) copyWithActivityTimeout(dst io.Writer, src io.Reader, timeout time.Duration) (int64, error) {
	buf := make([]byte, 64*1024)
	var written int64
	lastActivity := time.Now()

	for {
		if timeout > 0 {
			deadline := time.Now().Add(timeout)
			if conn, ok := src.(interface{ SetReadDeadline(time.Time) error }); ok {
				conn.SetReadDeadline(deadline)
			}
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			lastActivity = time.Now()
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}

		if er != nil {
			if er == io.EOF {
				break
			}
			if timeout > 0 && time.Since(lastActivity) > timeout {
				return written, fmt.Errorf("read timeout after %v of inactivity", timeout)
			}
			return written, er
		}

		if timeout > 0 && time.Since(lastActivity) > timeout {
			return written, fmt.Errorf("read timeout after %v of inactivity", timeout)
		}
	}
	return written, nil
}

func (d *Downloader) verifyChunks(chunkFiles []string, expectedChunks []ChunkInfo) error {
	fmt.Println("\nVerifying downloaded chunks...")
	var totalDownloadedSize int64

	for i, chunkFile := range chunkFiles {
		info, err := os.Stat(chunkFile)
		if err != nil {
			return fmt.Errorf("chunk %d verification failed - file not found (%s): %w", i, chunkFile, err)
		}

		actualSize := info.Size()
		expectedSize := expectedChunks[i].Size
		totalDownloadedSize += actualSize

		if actualSize == 0 {
			return fmt.Errorf("chunk %d verification failed - file is empty (%s)", i, chunkFile)
		}

		if actualSize < expectedSize-1024 || actualSize > expectedSize+1024 {
			fmt.Printf("WARNING: Chunk %d size mismatch - expected %d bytes, got %d bytes (%s)\n",
				i, expectedSize, actualSize, chunkFile)
		}

		fmt.Printf("  ✓ Chunk %d: %s (%s)\n", i, chunkFile, d.progressManager.FormatSize(actualSize))
	}

	fmt.Printf("✓ All %d chunks verified (total: %s)\n",
		len(chunkFiles), d.progressManager.FormatSize(totalDownloadedSize))
	return nil
}

func (d *Downloader) mergeChunks(chunkFiles []string) error {
	var totalMergeSize int64
	chunkSizes := make([]int64, len(chunkFiles))

	for i, chunkFile := range chunkFiles {
		info, err := os.Stat(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to stat chunk %d (%s): %w", i, chunkFile, err)
		}
		chunkSizes[i] = info.Size()
		totalMergeSize += info.Size()
	}

	fmt.Printf("\nMerging %d chunks (total: %s)...\n", len(chunkFiles), d.progressManager.FormatSize(totalMergeSize))

	output, err := os.Create(d.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	mergeProgress := &MergeProgress{
		totalSize: totalMergeSize,
		startTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.displayMergeProgress(ctx, mergeProgress)

	for i, chunkFile := range chunkFiles {
		fmt.Printf("Merging chunk %d/%d (%s)...", i+1, len(chunkFiles), d.progressManager.FormatSize(chunkSizes[i]))

		input, err := os.Open(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d (%s): %w", i, chunkFile, err)
		}

		progressReader := &MergeProgressReader{
			reader:   input,
			progress: mergeProgress,
		}

		written, err := io.Copy(output, progressReader)
		input.Close()

		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}

		if written != chunkSizes[i] {
			return fmt.Errorf("chunk %d: expected to copy %d bytes, but copied %d bytes",
				i, chunkSizes[i], written)
		}

		fmt.Printf(" ✓\n")
	}

	cancel()

	if err := output.Sync(); err != nil {
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	output.Close()
	return d.verifyFinalFile(totalMergeSize)
}

func (d *Downloader) verifyFinalFile(expectedSize int64) error {
	fmt.Println("Performing final file verification...")

	finalInfo, err := os.Stat(d.OutputPath)
	if err != nil {
		return fmt.Errorf("final file verification failed - file not found (%s): %w", d.OutputPath, err)
	}

	actualSize := finalInfo.Size()

	if actualSize != expectedSize {
		return fmt.Errorf("final file verification failed - expected %d bytes, got %d bytes (%s)",
			expectedSize, actualSize, d.OutputPath)
	}

	file, err := os.Open(d.OutputPath)
	if err != nil {
		return fmt.Errorf("final file verification failed - cannot open file (%s): %w", d.OutputPath, err)
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("final file verification failed - cannot read file (%s): %w", d.OutputPath, err)
	}
	if n == 0 && actualSize > 0 {
		return fmt.Errorf("final file verification failed - file appears to be empty or corrupted (%s)", d.OutputPath)
	}

	fmt.Printf("✓ Final file verification successful: %s\n", d.OutputPath)
	fmt.Printf("  File size: %s (%d bytes)\n", d.progressManager.FormatSize(actualSize), actualSize)
	fmt.Printf("  File permissions: %v\n", finalInfo.Mode())
	fmt.Printf("  Modified: %v\n", finalInfo.ModTime())

	return nil
}

func (d *Downloader) ensureMergeCompletion(chunkFiles []string, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("\nMerge attempt %d of %d...\n", attempt, maxRetries)

		if attempt > 1 {
			if err := os.Remove(d.OutputPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: failed to remove partial file: %v\n", err)
			}
		}

		err := d.mergeChunks(chunkFiles)
		if err == nil {
			fmt.Printf("✓ Merge completed successfully on attempt %d\n", attempt)
			return nil
		}

		lastErr = err
		fmt.Printf("✗ Merge attempt %d failed: %v\n", attempt, err)

		if attempt < maxRetries {
			fmt.Printf("Retrying in 2 seconds...\n")
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("merge failed after %d attempts, last error: %w", maxRetries, lastErr)
}

func (d *Downloader) displayMergeProgress(ctx context.Context, mergeProgress *MergeProgress) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			merged, total, percentage, speed := mergeProgress.GetProgress()

			completed := int(float64(ProgressBarWidth) * percentage / 100)
			remaining := ProgressBarWidth - completed

			progressBar := "[" + strings.Repeat("=", completed) + strings.Repeat("-", remaining) + "]"

			fmt.Printf("\rMerge: %s %.1f%% (%s/%s) %s",
				progressBar,
				percentage,
				d.progressManager.FormatSize(merged),
				d.progressManager.FormatSize(total),
				d.progressManager.FormatSpeed(speed))
		}
	}
}

func (d *Downloader) startProgressDisplay(ctx context.Context) {
	// Clear screen once at the start
	fmt.Print("\033[2J\033[H")

	ticker := time.NewTicker(200 * time.Millisecond) // Update every 200ms for smoother display
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.progressManager.DisplayProgress()
		}
	}
}

func (d *Downloader) Download() error {
	fileSize, err := d.getFileSize()
	if err != nil {
		return err
	}

	fmt.Printf("File size: %d bytes (%.2f MB)\n", fileSize, float64(fileSize)/(1024*1024))

	chunks := d.createChunks(fileSize)
	d.progressManager = NewProgressManager(chunks)

	fmt.Printf("Created %d chunks for concurrent download\n", len(chunks))

	tempDir, err := os.MkdirTemp("", "download-chunks-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(filepath.Dir(d.OutputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.startProgressDisplay(ctx)

	fmt.Printf("\nStarting concurrent download of %d chunks...\n\n", len(chunks))

	var wg sync.WaitGroup
	chunkFiles := make([]string, len(chunks))
	errorChan := make(chan error, len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		chunkFiles[i] = filepath.Join(tempDir, fmt.Sprintf("chunk-%d", i))

		go func(c ChunkInfo, outputFile string) {
			defer wg.Done()

			if err := d.downloadChunk(c, outputFile); err != nil {
				errorChan <- fmt.Errorf("chunk %d failed: %w", c.ID, err)
				return
			}
		}(chunk, chunkFiles[i])
	}

	wg.Wait()
	close(errorChan)

	cancel() // Stop progress display

	// Final progress display
	d.progressManager.DisplayProgress()
	fmt.Println()

	var downloadErrors []error
	for err := range errorChan {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		fmt.Printf("Download failed with %d errors:\n", len(downloadErrors))
		for _, err := range downloadErrors {
			fmt.Printf("  - %v\n", err)
		}
		return fmt.Errorf("download failed with %d chunk errors", len(downloadErrors))
	}

	fmt.Printf("✓ All %d chunks downloaded successfully\n", len(chunks))

	if err := d.verifyChunks(chunkFiles, chunks); err != nil {
		return fmt.Errorf("chunk verification failed: %w", err)
	}

	if err := d.ensureMergeCompletion(chunkFiles, 3); err != nil {
		return fmt.Errorf("merge completion failed: %w", err)
	}

	elapsed := time.Since(d.progressManager.startTime)
	avgSpeed := float64(fileSize) / elapsed.Seconds()

	fmt.Printf("\n🎉 Download completed successfully: %s\n", d.OutputPath)
	fmt.Printf("Total time: %v, Average speed: %s\n", elapsed.Round(time.Second), d.progressManager.FormatSpeed(avgSpeed))

	return nil
}

func main() {

	url := flag.String("url", "https://myUrlofTheFile.iso", "URL of the file to download.")
	outputPath := flag.String("output", "filename.extension", "Path to save the downloaded file.")
	chunks := flag.Int("chunks", 4, "Number of concurrent download chunks.")
	connectTimeout := flag.Duration("connect-timeout", 30*time.Second, "Connection timeout (e.g., '30s', '1m').")
	readTimeout := flag.Duration("read-timeout", 10*time.Minute, "Read timeout per chunk (e.g., '10m', '1h').")

	flag.Parse()

	downloader := NewDownloader(*url, *outputPath, *chunks)
	downloader.SetTimeouts(*connectTimeout, *readTimeout)

	fmt.Printf("Downloading: %s\n", *url)
	fmt.Printf("Output: %s\n", *outputPath)
	fmt.Printf("Chunks: %d\n", *chunks)
	fmt.Printf("Timeouts - Connect: %v, Read per chunk: %v\n",
		downloader.ConnectTimeout, downloader.ReadTimeout)
	fmt.Println()

	if err := downloader.Download(); err != nil {
		fmt.Printf("\nDownload failed: %v\n", err)
		os.Exit(1)
	}
}
