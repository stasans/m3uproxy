/*
Copyright © 2024 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package streamserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/userstore"

	"github.com/gorilla/mux"
)

type StreamServerConfig struct {
	Playlist       string `json:"playlist"`
	DefaultTimeout int    `json:"default_timeout,omitempty"`
	NumWorkers     int    `json:"num_workers,omitempty"`
	NoServiceImage string `json:"no_service_image,omitempty"`
	ScanTime       int    `json:"scan_time,omitempty"`
	HideInactive   bool   `json:"hide_inactive,omitempty"`
}

var (
	streams      = make([]Stream, 0)
	streamsMutex sync.Mutex
	stopServer   = make(chan bool)

	defaultTimeout     int
	numWorkers         int
	noServiceStream    *ffmpeg.HLSStream
	noServiceAvailable = false
	hideInactive       = false

	updateTimer *time.Timer
)

const m3uInternalPath = "m3uproxy/streams"
const m3uProxyPath = "m3uproxy/proxy"

func AddStreams(playlist *m3uparser.M3UPlaylist) error {

	streamsMutex.Lock()
	defer streamsMutex.Unlock()

	streams = make([]Stream, 0)

	for i, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		tvgId := entry.TVGTags.GetValue("tvg-id")
		radio := entry.TVGTags.GetValue("radio")
		if tvgId == "" && radio == "" {
			log.Printf("Missing tvg-id or radio tag for stream %s\n", entry.URI)
			continue
		}

		parsedURL, err := url.Parse(entry.URI)
		if err != nil {
			log.Printf("Failed to parse URL: %s\n", entry.URI)
			continue
		}

		prefix := ""

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			prefix += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		proxy := ""
		m3uproxyTags := entry.SearchTags("M3UPROXYTRANSPORT")
		if len(m3uproxyTags) > 0 {
			parts := strings.Split(m3uproxyTags[0].Value, "=")
			if len(parts) == 2 {
				switch parts[0] {
				case "proxy":
					proxy = parts[1]
				default:
					log.Printf("Unknown M3UPROXYTRANSPORT tag: %s\n", parts[0])
				}
			}
		}

		stream := Stream{
			index:     i,
			m3u:       entry,
			prefix:    prefix,
			active:    false,
			playlist:  nil,
			headers:   nil,
			httpProxy: proxy,
			mux:       &sync.Mutex{},
		}
		streams = append(streams, stream)
	}

	return nil
}

func StreamCount() int {
	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	return len(streams)
}

func Clear() {
	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	streams = make([]Stream, 0)
}

func MonitorStreams(cancel <-chan bool, signal chan<- bool) {

	var wg sync.WaitGroup
	var streamsChan = make(chan int)

	stopWorkers := make(chan bool)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go monitorWorker(streamsChan, stopWorkers, &wg)
	}

	go func() {
		for i := 0; i < len(streams); i++ {
			select {
			case <-cancel:
				stopWorkers <- true
				signal <- true
				wg.Wait()
				return
			default:
				streamsChan <- i
			}
		}
		close(streamsChan)
	}()

	wg.Wait()
}

func Start(data json.RawMessage) {

	var config StreamServerConfig
	err := json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Failed to parse stream server configuration: %s\n", err)
		return
	}

	log.Printf("Starting stream server\n")
	log.Printf("Playlist: %s\n", config.Playlist)
	log.Printf("No Service: %s\n", config.NoServiceImage)

	if config.DefaultTimeout < 1 {
		config.DefaultTimeout = 3
	}
	if config.NumWorkers < 1 {
		config.NumWorkers = 10
	}

	if config.ScanTime == 0 {
		config.ScanTime = 24 * 60 * 60
	}

	defaultTimeout = config.DefaultTimeout
	numWorkers = config.NumWorkers
	hideInactive = config.HideInactive

	// Initialize FFmpeg
	if err := ffmpeg.Initialize(); err != nil {
		log.Fatalf("Failed to initialize FFmpeg: %v\n", err)
	}

	// Start the no service stream
	log.Printf("Generating HLS for no service image\n")
	noServiceAvailable = true
	noServiceStream = ffmpeg.GenerateImageHLS(config.NoServiceImage, "no_service")

	if err := noServiceStream.Start(); err != nil {
		log.Fatalf("Failed to start no service stream: %v\n", err)
		noServiceAvailable = false
	}

	quit := make(chan bool)
	updateTimer = time.NewTimer(time.Duration(config.ScanTime) * time.Second)
	go func() {
		updatePlaylistAndMonitor(config, stopServer, quit)
		for {
			select {
			case <-quit:
				log.Println("Stopping stream server")
				return
			case <-stopServer:
				log.Println("Stopping stream server")
				return
			case <-updateTimer.C:
				updatePlaylistAndMonitor(config, stopServer, quit)
			}
		}
	}()
}

func Shutdown() {
	if !noServiceAvailable {
		if err := noServiceStream.Stop(); err != nil {
			log.Fatalf("Failed to stop no service stream: %v\n", err)
		}
		noServiceStream.Cleanup()
		noServiceAvailable = false
	}
	if !updateTimer.Stop() {
		<-updateTimer.C
	}
	stopServer <- true
	log.Printf("Stream server stopped\n")
}

func SetupHandlers(r *mux.Router) {
	r.HandleFunc("/"+m3uInternalPath+"/{path:.*}", HandleInternalStream).Methods("GET")
	r.HandleFunc("/"+m3uProxyPath+"/{token}/{streamId}/{path:.*}", HandleStreamRequest).Methods("GET")
	r.HandleFunc("/streams.m3u", HandleStreamPlaylist).Methods("GET")
}

func HandleStreamRequest(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]

	ok := userstore.ValidateSingleToken(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: missing token\n", r.URL.Path)
		return
	}

	streamID, err := strconv.Atoi(vars["streamId"])
	if err != nil {
		http.Redirect(w, r, "/"+m3uInternalPath+"/no_service/index.m3u8", http.StatusMovedPermanently)
		return
	}

	if err := streams[streamID].Serve(w, r); err != nil {
		log.Printf("Error serving stream stream: %v\n", err)
		http.Redirect(w, r, "/"+m3uInternalPath+"/no_service/index.m3u8", http.StatusMovedPermanently)
		return
	}
}

func HandleInternalStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	if strings.HasPrefix(path, "no_service") {
		if !noServiceAvailable {
			http.NotFound(w, r)
			return
		}
		noServiceStream.Serve(w, r)
		return
	}

	http.NotFound(w, r)
}

func HandleStreamPlaylist(w http.ResponseWriter, r *http.Request) {

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

	log.Printf("Generated M3U playlist for user %s\n", username)

	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("#EXTM3U\n"))
	for i, stream := range streams {
		if hideInactive && !stream.active {
			continue
		}
		entry := m3uparser.M3UEntry{
			URI:   fmt.Sprintf("http://%s/%s/%s/%d/playlist.m3u8", r.Host, m3uProxyPath, token, i),
			Title: stream.m3u.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3u.Tags...)
		w.Write([]byte(entry.String() + "\n"))
	}
}
