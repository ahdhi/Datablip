package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/govind1331/Datablip/internal/downloader"
)

type Server struct {
	manager *downloader.Manager
	router  *mux.Router
}

func NewServer(manager *downloader.Manager) *Server {
	s := &Server{
		manager: manager,
		router:  mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/downloads", s.listDownloads).Methods("GET")
	api.HandleFunc("/downloads", s.createDownload).Methods("POST")
	api.HandleFunc("/downloads/{id}", s.getDownload).Methods("GET")
	api.HandleFunc("/downloads/{id}/pause", s.pauseDownload).Methods("POST")
	api.HandleFunc("/downloads/{id}/resume", s.resumeDownload).Methods("POST")
	api.HandleFunc("/downloads/{id}/file", s.downloadFile).Methods("GET")
	api.HandleFunc("/downloads/{id}", s.deleteDownload).Methods("DELETE")
	api.HandleFunc("/settings", s.getSettings).Methods("GET")
	api.HandleFunc("/settings", s.updateSettings).Methods("PUT")

	// Serve frontend
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/frontend/build/")))
}

type CreateDownloadRequest struct {
	URL            string `json:"url"`
	Filename       string `json:"filename"`
	Chunks         int    `json:"chunks"`
	ConnectTimeout string `json:"connectTimeout"`
	ReadTimeout    string `json:"readTimeout"`
}

func (s *Server) createDownload(w http.ResponseWriter, r *http.Request) {
	var req CreateDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Debug logging
	fmt.Printf("=== CREATE DOWNLOAD REQUEST ===\n")
	fmt.Printf("URL: %s\n", req.URL)
	fmt.Printf("Filename: %s\n", req.Filename)
	fmt.Printf("Chunks: %d\n", req.Chunks)
	fmt.Printf("ConnectTimeout: %s\n", req.ConnectTimeout)
	fmt.Printf("ReadTimeout: %s\n", req.ReadTimeout)
	fmt.Printf("===============================\n")

	download, err := s.manager.AddDownload(
		req.URL,
		req.Filename,
		req.Chunks,
		req.ConnectTimeout,
		req.ReadTimeout,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(download)
}

func (s *Server) listDownloads(w http.ResponseWriter, r *http.Request) {
	downloads := s.manager.GetAllDownloads()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloads)
}

func (s *Server) getDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	download, err := s.manager.GetDownload(vars["id"])

	if err != nil {
		http.Error(w, "Download not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(download)
}

func (s *Server) pauseDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := s.manager.PauseDownload(vars["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) resumeDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := s.manager.ResumeDownload(vars["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	download, err := s.manager.GetDownload(vars["id"])

	if err != nil {
		http.Error(w, "Download not found", http.StatusNotFound)
		return
	}

	if download.Status != "completed" {
		http.Error(w, "Download not completed yet", http.StatusBadRequest)
		return
	}

	// Check if file exists
	if _, err := os.Stat(download.OutputPath); os.IsNotExist(err) {
		http.Error(w, "Downloaded file not found", http.StatusNotFound)
		return
	}

	// Open the file
	file, err := os.Open(download.OutputPath)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set appropriate headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(download.Filename)))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Copy file to response
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error serving file", http.StatusInternalServerError)
		return
	}
}

func (s *Server) deleteDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := s.manager.DeleteDownload(vars["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	// Return global settings
	settings := map[string]interface{}{
		"defaultChunks":          4,
		"connectTimeout":         "30s",
		"readTimeout":            "10m",
		"maxConcurrentDownloads": 3,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	// Update global settings
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.router.ServeHTTP(w, r)
}
