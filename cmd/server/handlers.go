package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/streamstore"
	"github.com/a13labs/m3uproxy/pkg/userstore"

	"github.com/gorilla/mux"
)

const m3uInternalPath = "m3uproxy/streams"
const m3uProxyPath = "m3uproxy/proxy"

func setupHandlers() *mux.Router {
	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Streams and EPG endpoints
	r.HandleFunc("/streams.m3u", streamsHandler).Methods("GET")
	r.HandleFunc("/epg.xml", epgHandler).Methods("GET")

	// HLS streams (internal and external)
	r.HandleFunc("/"+m3uInternalPath+"/{path:.*}", internalStreamHandler).Methods("GET")
	r.HandleFunc("/"+m3uProxyPath+"/{token}/{streamId}/{path:.*}", proxyHandler).Methods("GET")
	return r
}

func streamsHandler(w http.ResponseWriter, r *http.Request) {

	username, password, ok := verifyAuth(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream: invalid credentials\n")
		return
	}

	token := userstore.GetActiveToken(username)
	if token == "" {
		var err error
		token, err = userstore.GenerateToken(username, password)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			log.Printf("Unauthorized access to stream: invalid credentials\n")
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	playlist := streamstore.ExportPlaylist(r.Host, m3uProxyPath, token)
	w.Write([]byte(playlist.String()))
	log.Printf("Generated M3U playlist for user %s\n", username)
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

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]

	ok := userstore.ValidateSingleToken(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: missing token\n", r.URL.Path)
		return
	}

	if err := streamstore.StreamStreamHandler(w, r); err != nil {
		log.Printf("Error serving stream stream: %v\n", err)
		http.Redirect(w, r, "/"+m3uInternalPath+"/no_service/index.m3u8", http.StatusMovedPermanently)
	}
}

func internalStreamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	if strings.HasPrefix(path, "no_service") {
		noServiceHandler(w, r)
		return
	}

	http.NotFound(w, r)
}
