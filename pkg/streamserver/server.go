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
	KodiSupport    bool   `json:"kodi,omitempty"`
}

var (
	streams      = make([]Stream, 0)
	streamsMutex sync.Mutex
	stopServer   = make(chan bool)

	noServiceStream *ffmpeg.HLSStream
	serverConfig    StreamServerConfig
	updateTimer     *time.Timer
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

		headers := make(map[string]string)
		m3uproxyTags = entry.SearchTags("M3UPROXYHEADER")
		for _, tag := range m3uproxyTags {
			parts := strings.Split(tag.Value, "=")
			if len(parts) == 2 {
				headers[parts[0]] = parts[1]
			}
		}
		vlcTags := entry.SearchTags("EXTVLCOPT")
		for _, tag := range vlcTags {
			parts := strings.Split(tag.Value, "=")
			if len(parts) == 2 {
				switch parts[0] {
				case "http-user-agent":
					headers["User-Agent"] = parts[1]
				case "http-referrer":
					headers["Referer"] = parts[1]
				default:
				}
			}
		}

		m3uproxyTags = entry.SearchTags("M3UPROXYOPT")
		forceKodiHeaders := false
		for _, tag := range m3uproxyTags {
			switch tag.Value {
			case "forcekodiheaders":
				forceKodiHeaders = true
			default:
			}
		}

		// Clear non-standard tags
		entry.ClearTags()

		stream := Stream{
			index:            i,
			m3u:              entry,
			prefix:           prefix,
			active:           false,
			hlsPlaylist:      nil,
			headers:          headers,
			httpProxy:        proxy,
			forceKodiHeaders: forceKodiHeaders,
			mux:              &sync.Mutex{},
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
	for i := 0; i < serverConfig.NumWorkers; i++ {
		wg.Add(1)
		go monitorWorker(streamsChan, stopWorkers, &wg, serverConfig.DefaultTimeout)
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

	err := json.Unmarshal(data, &serverConfig)
	if err != nil {
		log.Printf("Failed to parse stream server configuration: %s\n", err)
		return
	}

	log.Printf("Starting stream server\n")
	log.Printf("Playlist: %s\n", serverConfig.Playlist)
	log.Printf("No Service: %s\n", serverConfig.NoServiceImage)

	if serverConfig.DefaultTimeout < 1 {
		serverConfig.DefaultTimeout = 3
	}
	if serverConfig.NumWorkers < 1 {
		serverConfig.NumWorkers = 10
	}

	if serverConfig.ScanTime == 0 {
		serverConfig.ScanTime = 24 * 60 * 60
	}

	// Initialize FFmpeg
	if err := ffmpeg.Initialize(); err != nil {
		log.Fatalf("Failed to initialize FFmpeg: %v\n", err)
	}

	// Start the no service stream
	log.Printf("Generating HLS for no service image\n")
	noServiceStream = ffmpeg.GenerateImageHLS(serverConfig.NoServiceImage, "no_service")

	if err := noServiceStream.Start(); err != nil {
		log.Fatalf("Failed to start no service stream: %v\n", err)
		noServiceStream = nil
	}

	quit := make(chan bool)
	updateTimer = time.NewTimer(time.Duration(serverConfig.ScanTime) * time.Second)
	go func() {
		updatePlaylistAndMonitor(serverConfig, stopServer, quit)
		for {
			select {
			case <-quit:
				log.Println("Stopping stream server")
				return
			case <-stopServer:
				log.Println("Stopping stream server")
				return
			case <-updateTimer.C:
				updatePlaylistAndMonitor(serverConfig, stopServer, quit)
			}
		}
	}()
}

func Shutdown() {
	if noServiceStream != nil {
		if err := noServiceStream.Stop(); err != nil {
			log.Fatalf("Failed to stop no service stream: %v\n", err)
		}
		noServiceStream.Cleanup()
	}
	updateTimer.Stop()
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

	if err := streams[streamID].Serve(w, r, serverConfig.DefaultTimeout); err != nil {
		log.Printf("Error serving stream stream: %v\n", err)
		http.Redirect(w, r, "/"+m3uInternalPath+"/no_service/index.m3u8", http.StatusMovedPermanently)
		return
	}
}

func HandleInternalStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	if strings.HasPrefix(path, "no_service") {
		if noServiceStream == nil {
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
		if serverConfig.HideInactive && !stream.active {
			continue
		}
		entry := m3uparser.M3UEntry{
			URI:   fmt.Sprintf("http://%s/%s/%s/%d/playlist.m3u8", r.Host, m3uProxyPath, token, i),
			Title: stream.m3u.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3u.Tags...)
		if stream.forceKodiHeaders || (serverConfig.KodiSupport && stream.hlsPlaylist != nil) {
			entry.AddTag("KODIPROP", "inputstream=inputstream.adaptive")
			entry.AddTag("KODIPROP", "inputstream.adaptive.manifest_type=hls")
		}
		w.Write([]byte(entry.String() + "\n"))
	}
}
