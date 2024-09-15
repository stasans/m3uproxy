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
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"

	"github.com/gorilla/mux"
)

type StreamServerConfig struct {
	Playlist       string `json:"playlist"`
	Timeout        int    `json:"default_timeout,omitempty"`
	NumWorkers     int    `json:"num_workers,omitempty"`
	NoServiceImage string `json:"no_service_image,omitempty"`
	ScanTime       int    `json:"scan_time,omitempty"`
	HideInactive   bool   `json:"hide_inactive,omitempty"`
	KodiSupport    bool   `json:"kodi,omitempty"`
	UseHttps       bool   `json:"force_https_url,omitempty"`
}

var (
	streams      = make([]*Stream, 0)
	streamsMutex sync.Mutex
	stopServer   = make(chan bool)
	running      = false

	noServiceStream *LocalStream
	serverConfig    StreamServerConfig
	updateTimer     *time.Timer
)

const m3uPlaylist = "master.m3u8"

func monitorWorker(stream <-chan *Stream, stop <-chan bool, wg *sync.WaitGroup) {

	defer wg.Done()
	for s := range stream {
		select {
		case <-stop:
			return
		default:
			s.HealthCheck()
			if !s.active {
				log.Printf("Stream '%s' is offline.\n", s.m3u.Title)
			}
		}
	}
}

func LoadStreams() error {

	if _, err := os.Stat(serverConfig.Playlist); os.IsNotExist(err) {
		log.Printf("File %s not found\n", serverConfig.Playlist)
		return nil
	}

	extension := filepath.Ext(serverConfig.Playlist)
	var playlist *m3uparser.M3UPlaylist
	var err error
	if extension == ".m3u" {
		log.Printf("Loading M3U file %s\n", serverConfig.Playlist)
		playlist, err = m3uparser.ParseM3UFile(serverConfig.Playlist)
		if err != nil {
			return err
		}
	} else if extension == ".json" {
		log.Printf("Loading JSON file %s\n", serverConfig.Playlist)
		playlist, err = m3uprovider.LoadFromFile(serverConfig.Playlist)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid file extension")
	}

	if playlist == nil {
		return errors.New("failed to load playlist")
	}

	log.Printf("Loaded %d streams from %s\n", playlist.StreamCount(), serverConfig.Playlist)

	streamList := make([]*Stream, 0)

	var wg sync.WaitGroup
	var streamsChan = make(chan *Stream)

	stopWorkers := make(chan bool)
	for i := 0; i < serverConfig.NumWorkers; i++ {
		wg.Add(1)
		go monitorWorker(streamsChan, stopWorkers, &wg)
	}

	go func() {
		for i, entry := range playlist.Entries {
			select {
			case <-stopServer:
				stopWorkers <- true
				wg.Wait()
				return
			default:
				if entry.URI == "" {
					continue
				}

				tvgId := entry.TVGTags.GetValue("tvg-id")
				radio := entry.TVGTags.GetValue("radio")
				if tvgId == "" && radio == "" {
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
					prefix = strings.TrimLeft(prefix, "/")
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

				transport := http.DefaultTransport.(*http.Transport)
				if proxy != "" {
					proxyURL, err := url.Parse(proxy)
					if err == nil {
						transport = &http.Transport{
							Proxy: http.ProxyURL(proxyURL),
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
				disableRemap := false
				for _, tag := range m3uproxyTags {
					switch tag.Value {
					case "forcekodiheaders":
						forceKodiHeaders = true
					case "disableremap":
						disableRemap = true
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
					headers:          headers,
					httpProxy:        proxy,
					forceKodiHeaders: forceKodiHeaders,
					radio:            radio == "true",
					mux:              &sync.Mutex{},
					transport:        transport,
					disableRemap:     disableRemap,
				}

				streamList = append(streamList, &stream)
				streamsChan <- &stream
			}
		}
		close(streamsChan)
	}()

	wg.Wait()
	log.Printf("Loaded %d active streams\n", len(streamList))

	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	streams = streamList

	return nil
}

func StreamCount() int {
	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	return len(streams)
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

	if serverConfig.Timeout < 1 {
		serverConfig.Timeout = 3
	}
	if serverConfig.NumWorkers < 1 {
		serverConfig.NumWorkers = 1
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
	noServiceStream = NewImageStream(serverConfig.NoServiceImage, "no_service")

	if err := noServiceStream.Start(); err != nil {
		log.Fatalf("Failed to start no service stream: %v\n", err)
		noServiceStream = nil
	}

	updateTimer = time.NewTimer(time.Duration(serverConfig.ScanTime) * time.Second)
	running = true
	go func() {
		LoadStreams()
		for {
			select {
			case <-stopServer:
				log.Println("Stopping stream server")
				return
			case <-updateTimer.C:
				LoadStreams()
				if running {
					updateTimer.Reset(time.Duration(serverConfig.ScanTime) * time.Second)
				}
			}
			if !running {
				break
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
	running = false
	stopServer <- true
	log.Printf("Stream server stopped\n")
}

func Routes(next *mux.Router) http.Handler {
	next.HandleFunc("/{token}/{streamId}/{path:.*}", handleStreamRequest)
	next.HandleFunc("/streams.m3u", handleStreamPlaylist)
	return next
}

func handleStreamRequest(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.\n", r.URL.Path)
		return
	}

	streamID, err := strconv.Atoi(vars["streamId"])
	if err != nil {
		http.Error(w, "Invalid stream ID", http.StatusBadRequest)
		return
	}

	streamsMutex.Lock()
	defer streamsMutex.Unlock()

	if streamID < 0 || streamID >= len(streams) {
		http.Error(w, "Invalid stream ID", http.StatusBadRequest)
		return
	}

	stream := streams[streamID]
	if !stream.active {
		noServiceStream.Serve(w, r)
		return
	}
	stream.Serve(w, r)
}

func handleStreamPlaylist(w http.ResponseWriter, r *http.Request) {

	token, ok := getJWTToken(r)
	if !ok {
		var err error

		username, password, ok := getUserCredentials(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			log.Printf("Unauthorized access to stream: invalid credentials\n")
			return
		}

		token, err = auth.CreateToken(username, password)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			log.Printf("Unauthorized access to stream: invalid credentials\n")
			return
		}
	}

	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("#EXTM3U\n"))
	for i, stream := range streams {
		if serverConfig.HideInactive && !stream.active {
			continue
		}

		protocol := "http"
		if serverConfig.UseHttps && r.TLS == nil {
			protocol = "https"
		}

		uri := fmt.Sprintf("%s://%s/%s/%d/%s", protocol, r.Host, token, i, m3uPlaylist)
		if stream.disableRemap {
			uri = stream.m3u.URI
		}

		entry := m3uparser.M3UEntry{
			URI:   uri,
			Title: stream.m3u.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3u.Tags...)
		if !stream.radio {
			if stream.forceKodiHeaders || (serverConfig.KodiSupport && !stream.radio) {
				entry.AddTag("KODIPROP", "inputstream=inputstream.adaptive")
				entry.AddTag("KODIPROP", "inputstream.adaptive.manifest_type=hls")
			}
		}
		w.Write([]byte(entry.String() + "\n"))
	}
}
