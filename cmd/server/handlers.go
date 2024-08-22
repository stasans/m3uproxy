package server

import (
	"io"
	"log"
	"m3u-proxy/pkg/channelstore"
	"m3u-proxy/pkg/ffmpeg"
	"m3u-proxy/pkg/userstore"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func setupHandlers() {
	r := mux.NewRouter()
	r.HandleFunc("/channels.m3u", channelsHandler).Methods("GET")
	r.HandleFunc("/epg.xml", epgHandler).Methods("GET")

	if ffmpeg.Initialize() == nil {
		log.Printf("FFmpeg initialized successfully. Registering /m3uproxy_internal for streams.\n")
		r.HandleFunc("/m3uproxy_internal/{path:.*}", ffmpeg.ServeHLS).Methods("GET")
		if _, err := os.Stat(noServiceImage); os.IsNotExist(err) {
			log.Fatalf("No service image not found at %s\n", noServiceImage)
		} else {
			log.Printf("Generating HLS for no service image\n")
			ffmpeg.GenerateHLS(noServiceImage, "no_service")
			noServiceAvailable = true
		}
	}
	r.HandleFunc("/m3uproxy/{token}/{channelId}/{path:.*}", proxyHandler).Methods("GET")
	r.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return r.URL.Path == "/"
	}).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/channels.m3u", http.StatusMovedPermanently)
	})
	http.Handle("/", r)
}

func handleStreamError(w http.ResponseWriter, r *http.Request, msg string) {
	if noServiceAvailable {
		http.Redirect(w, r, "/m3uproxy_internal/no_service.m3u8", http.StatusMovedPermanently)
	} else {
		http.Error(w, msg, http.StatusNotFound)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]

	ok := userstore.ValidateSingleToken(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Unauthorized access to channel stream %s: missing token\n", r.URL.Path)
		return
	}

	channelID, err := strconv.Atoi(vars["channelId"])
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		log.Printf("Invalid channel ID: %s\n", vars["channelID"])
		return
	}

	path := vars["path"]
	serviceURL, err := channelstore.GetChannelStreamURL(channelID, path, r.URL.RawQuery)
	if err != nil {
		return
	}

	// Create a new request to the service URL
	// and copy the headers from the original request
	req, _ := http.NewRequest("GET", serviceURL, r.Body)
	req.Header.Add("User-Agent", r.Header.Get("User-Agent"))
	req.Header.Add("Referer", r.Header.Get("Referer"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch channel: %v\n", channelID)
		handleStreamError(w, r, "Failed to fetch channel")
		return
	}

	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func channelsHandler(w http.ResponseWriter, r *http.Request) {

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
	playlist := channelstore.ExportPlaylist(r.Host, "m3uproxy", token)
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
