package streamserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/streamserver/streamSources"

	"github.com/gorilla/mux"
)

type streamEntry struct {
	index   int
	tvgId   string
	sources streamSources.Sources
}

var (
	channels          = make(map[string]*streamEntry, 0)
	channelsMux       sync.Mutex
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

	var wg sync.WaitGroup
	var streamsChan = make(chan *streamEntry)

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
					log.Printf("No tvg-id or radio tag found for %s, skipping\n", entry.URI)
					continue
				}

				channel, ok := channels[tvgId]
				if !ok {
					channelsMux.Lock()
					channels[tvgId] = &streamEntry{
						index:   i,
						tvgId:   tvgId,
						sources: streamSources.CreateSources(),
					}
					channel = channels[tvgId]
					channelsMux.Unlock()
				}

				if channel.sources.SourceExists(entry) {
					log.Printf("Stream source already exists: %s\n", entry.URI)
					continue
				}

				log.Printf("Adding stream source for %s, for channel %s\n", entry.URI, tvgId)
				channel.sources.AddSource(entry, Config.Timeout)
				streamsChan <- channel
			}
		}
		close(streamsChan)
	}()

	wg.Wait()

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
		registerChannelsRoutes(r)

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
	if channels == nil {
		http.Error(w, "Streams not loaded", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func streamRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.\n", r.URL.Path)
		return
	}

	channelId, ok := vars["channelId"]
	if !ok {
		http.Error(w, "Invalid stream ID", http.StatusBadRequest)
		return
	}

	channelsMux.Lock()
	defer channelsMux.Unlock()

	channel, ok := channels[channelId]
	if !ok {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	if !channel.sources.Active() {
		http.Error(w, "Stream not active", http.StatusNotFound)
		return
	}

	channel.sources.Serve(w, r, Config.Timeout)
}

func registerChannelsRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/{token}/{channelId}/{path:.*}", streamRequest)
	return r
}
