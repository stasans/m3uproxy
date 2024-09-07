package server

import (
	"log"
	"net/http"

	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/streamserver"

	"github.com/gorilla/mux"
)

var epgFilePath = "epg.xml"

func setupHandlers(config *rootCmd.Config) *mux.Router {
	r := mux.NewRouter()

	// Streams and EPG endpoints
	epgFilePath = config.Epg
	streamserver.SetupHandlers(r)
	r.HandleFunc("/epg.xml", epgHandler).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	return r
}

func epgHandler(w http.ResponseWriter, r *http.Request) {
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
