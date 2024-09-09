package server

import (
	"log"
	"net/http"

	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/streamserver"

	"github.com/gorilla/mux"
)

var epgFilePath = "epg.xml"

func routes(config *rootCmd.Config) http.Handler {
	r := mux.NewRouter()

	// Streams and EPG endpoints
	epgFilePath = config.Epg
	r.HandleFunc("/epg.xml", getEpg).Methods("GET")
	r.HandleFunc("/player", getPlayer).Methods("GET")
	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	return streamserver.StreamServer(r)
}

func getEpg(w http.ResponseWriter, r *http.Request) {
	content, err := loadContent(epgFilePath)
	if err != nil {
		http.Error(w, "EPG file not found", http.StatusNotFound)
		log.Printf("EPG file not found at %s\n", epgFilePath)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
	log.Printf("EPG data served successfully\n")
}

func getPlayer(w http.ResponseWriter, r *http.Request) {
	content, err := loadContent("assets/player.html")
	if err != nil {
		http.Error(w, "Player file not found", http.StatusNotFound)
		log.Printf("Player file not found at %s\n", "player.html")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
	log.Printf("Player served successfully\n")
}
