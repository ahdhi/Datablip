package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/govind1331/Datablip/internal/api"
	"github.com/govind1331/Datablip/internal/downloader"
	"github.com/govind1331/Datablip/internal/websocket"
)

func main() {
	var (
		port = flag.String("port", "8080", "Server port")
	)
	flag.Parse()

	// Initialize download manager
	manager := downloader.NewManager()

	// Initialize API server
	apiServer := api.NewServer(manager)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(manager)
	go wsHub.Run()

	// Setup main router
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", wsHub.ServeWS)

	// API and static files
	router.PathPrefix("/").Handler(apiServer)

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Server starting on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
