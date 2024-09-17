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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"

	"github.com/gorilla/mux"
)

type ServerConfig struct {
	Port           int             `json:"port"`
	Playlist       string          `json:"playlist"`
	Epg            string          `json:"epg"`
	Timeout        int             `json:"default_timeout,omitempty"`
	NumWorkers     int             `json:"num_workers,omitempty"`
	NoServiceImage string          `json:"no_service_image,omitempty"`
	ScanTime       int             `json:"scan_time,omitempty"`
	HideInactive   bool            `json:"hide_inactive,omitempty"`
	KodiSupport    bool            `json:"kodi,omitempty"`
	UseHttps       bool            `json:"force_https_url,omitempty"`
	Security       SecurityConfig  `json:"security,omitempty"`
	Auth           json.RawMessage `json:"auth"`
}

var (
	streams      = make([]*streamStruct, 0)
	streamsMutex sync.Mutex
	stopServer   = make(chan bool)
	running      = false

	noServiceStream *ffmpegStream
	serverConfig    ServerConfig
	updateTimer     *time.Timer
)

const m3uPlaylist = "master.m3u8"

func LoadStreams() error {

	if _, err := os.Stat(serverConfig.Playlist); os.IsNotExist(err) {
		log.Printf("File %s not found\n", serverConfig.Playlist)
		return nil
	}

	playlist, err := m3uprovider.LoadFromFile(serverConfig.Playlist)
	if err != nil {
		return err
	}

	log.Printf("Loaded %d streams from %s\n", playlist.StreamCount(), serverConfig.Playlist)

	streamList := make([]*streamStruct, 0)

	var wg sync.WaitGroup
	var streamsChan = make(chan *streamStruct)

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

				if headers["User-Agent"] == "" {
					headers["User-Agent"] = "VLC/3.0.11 LibVLC/3.0.11"
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

				stream := streamStruct{
					index:            i,
					m3u:              entry,
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

func Start(config ServerConfig) {

	serverConfig = config

	log.Printf("Starting stream server\n")
	log.Printf("Playlist: %s\n", serverConfig.Playlist)
	log.Printf("No Service: %s\n", serverConfig.NoServiceImage)
	log.Printf("EPG: %s\n", serverConfig.Epg)

	if serverConfig.Timeout < 1 {
		serverConfig.Timeout = 3
	}
	if serverConfig.NumWorkers < 1 {
		serverConfig.NumWorkers = 1
	}

	if serverConfig.ScanTime == 0 {
		serverConfig.ScanTime = 24 * 60 * 60
	}

	if serverConfig.Port == 0 {
		serverConfig.Port = 8080
	}

	log.Printf("Starting M3U Proxy Server\n")

	err := auth.InitializeAuth(serverConfig.Auth)
	if err != nil {
		log.Printf("Failed to initialize authentication: %s\n", err)
		return
	}

	// Initialize FFmpeg
	if err := ffmpeg.Initialize(); err != nil {
		log.Fatalf("Failed to initialize FFmpeg: %v\n", err)
	}

	// Start the no service stream
	log.Printf("Generating HLS for no service image\n")
	noServiceStream = newImageStream(serverConfig.NoServiceImage, "no_service")

	if err := noServiceStream.start(); err != nil {
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

	handler := mux.NewRouter()
	registerRoutes(handler)
	handler.HandleFunc("/auth", basicAuth(handleAuthRequest))
	// Health check endpoint
	handler.HandleFunc("/health", handleHealthCheck)
	// Player, Playlist, EPG and Streams endpoints
	handler.HandleFunc("/player", handlerGetPlayer)
	handler.HandleFunc("/streams.m3u", basicAuth(handleGetPlaylist))
	handler.HandleFunc("/epg.xml", handleGetEpg)
	handler.HandleFunc("/{token}/{streamId}/{path:.*}", handleGetStream)

	if configureGeoIp() != nil {
		log.Println("GeoIP database not found, geo-location will not be available.")
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverConfig.Port),
		Handler: geoip(handler),
	}

	// Channel to listen for termination signal (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		log.Printf("Server listening on %s.\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("Server failed:", err)
		}
	}()

	<-quit // Wait for SIGINT or SIGTERM

	log.Println("Shutting down server...")

	// Stop the no service stream
	if noServiceStream != nil {
		if err := noServiceStream.stop(); err != nil {
			log.Fatalf("Failed to stop no service stream: %v\n", err)
		}
		noServiceStream.free()
	}
	updateTimer.Stop()
	running = false
	stopServer <- true
	log.Printf("Stream server stopped\n")

	cleanGeoIp()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	log.Println("Server shutdown.")
}
