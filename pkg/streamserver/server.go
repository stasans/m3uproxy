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

	"github.com/gorilla/mux"
)

var (
	streams           = make([]*streamStruct, 0)
	streamsMutex      sync.Mutex
	stopStreamLoading = make(chan bool)
	restartServer     = make(chan bool)
	running           = false

	updateTimer *time.Timer
)

// const m3uPlaylist = "master.m3u8"

func LoadStreams() error {

	if err := LoadPlaylist(); err != nil {
		return err
	}

	streamList := make([]*streamStruct, 0)

	var wg sync.WaitGroup
	var streamsChan = make(chan *streamStruct)

	stopWorkers := make(chan bool)
	for i := 0; i < Config.NumWorkers; i++ {
		wg.Add(1)
		go monitorWorker(streamsChan, stopWorkers, &wg)
	}

	go func() {
		for i, entry := range m3uCache.Entries {
			select {
			case <-stopStreamLoading:
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

func Run(configPath string) {

	if err := LoadServerConfig(configPath); err != nil {
		log.Fatalf("Failed to load server configuration: %v\n", err)
		return
	}

	setupLogging(Config.LogFile)

	for {
		log.Printf("Starting M3U Proxy Server\n")

		log.Printf("Starting stream server\n")
		log.Printf("Playlist: %s\n", Config.Playlist)
		log.Printf("EPG: %s\n", Config.Epg)

		err := auth.InitializeAuth(Config.Auth)
		if err != nil {
			log.Printf("Failed to initialize authentication: %s\n", err)
			return
		}

		updateTimer = time.NewTimer(time.Duration(Config.ScanTime) * time.Second)
		running = true
		go func() {
			LoadStreams()
			for {
				select {
				case <-stopStreamLoading:
					log.Println("Stopping stream server")
					return
				case <-updateTimer.C:
					LoadStreams()
					if running {
						updateTimer.Reset(time.Duration(Config.ScanTime) * time.Second)
					}
				}
				if !running {
					break
				}
			}
		}()

		r := mux.NewRouter()
		registerHealthCheckRoutes(r)
		registerAPIRoutes(r)
		registerPlaylistRoutes(r)
		registerPlayerRoutes(r)
		registerEpgRoutes(r)
		registerLicenseRoutes(r)
		registerStreamsRoutes(r)

		if configureSecurity() != nil {
			log.Println("GeoIP database not found, geo-location will not be available.")
		}

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", Config.Port),
			Handler: secure(r),
		}

		// Channel to listen for termination signal (SIGINT, SIGTERM)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			log.Printf("Server listening on %s.\n", server.Addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Println("Server failed:", err)
			}
		}()

		quitServer := false
		select {
		case <-sigChan:
			log.Println("Signal received, shutting down server...")
			quitServer = true
		case <-restartServer:
			quitServer = false
		}

		updateTimer.Stop()
		running = false
		stopStreamLoading <- true
		log.Printf("Stream server stopped\n")

		cleanGeoIp()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Println("Server forced to shutdown:", err)
		}

		if quitServer {
			log.Println("Server shutdown.")
			break
		}
	}
}

func Restart() {
	go func() {
		log.Println("Server restart in 3 seconds")
		time.Sleep(3 * time.Second)
		restartServer <- true
	}()
}

func healthCheckRequest(w http.ResponseWriter, r *http.Request) {
	if streams == nil {
		http.Error(w, "Streams not loaded", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
