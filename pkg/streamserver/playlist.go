package streamserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/a13labs/m3uproxy/pkg/streamserver/streamSources"
	"github.com/gorilla/mux"
)

var (
	licenseManger *streamLicenseManager
)

type streamEntry struct {
	index   int
	tvgId   string
	sources streamSources.Sources
}

type PlaylistHandler struct {
	config         *ServerConfig
	m3uCache       *m3uparser.M3UPlaylist
	playlistConfig *m3uprovider.PlaylistConfig
	channelsMux    sync.Mutex
	channels       map[string]*streamEntry
}

func NewPlaylistHandler(config *ServerConfig) *PlaylistHandler {
	return &PlaylistHandler{
		config:      config,
		channels:    make(map[string]*streamEntry),
		channelsMux: sync.Mutex{},
	}
}

func (p *PlaylistHandler) RegisterRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/streams.m3u", basicAuth(p.playlistRequest))
	r.HandleFunc("/drm/licensing", basicAuth(licenseKeysRequest))
	r.HandleFunc("/{token}/{channelId}/{path:.*}", p.streamRequest)
	return r
}

func (p *PlaylistHandler) loadFromSource() error {
	var err error
	p.playlistConfig, err = m3uprovider.LoadPlaylistConfig(p.config.data.Playlist)
	if err != nil {
		return err
	}

	p.m3uCache, err = m3uprovider.Load(p.playlistConfig)
	if err != nil {
		return err
	}

	// Load licenses
	// For now we just support processing clearkey licenses and KODIPROP tags
	for _, entry := range p.m3uCache.Entries {
		keyType, keyId, keyValue := "", "", ""
		for _, tag := range entry.Tags {
			if tag.Tag == "KODIPROP" {
				if strings.HasPrefix(tag.Value, "inputstream.adaptive.license_type=") {
					parts := strings.Split(tag.Value, "=")
					if len(parts) == 2 {
						keyType = parts[1]
					}
					continue
				}
				if strings.HasPrefix(tag.Value, "inputstream.adaptive.license_key=") {

					if keyType == "org.w3.clearkey" {
						parts := strings.Split(tag.Value, "=")
						if len(parts) == 2 {
							licenseKey := parts[1]
							keyId = strings.Split(licenseKey, ":")[0]
							keyValue = strings.Split(licenseKey, ":")[1]

							if licenseManger == nil {
								licenseManger = newStreamLicenseManager()
							}
							log.Printf("Found license, adding license key with id %s\n", keyId)
							licenseManger.addLicense("clearkey", keyId, keyValue)
							keyType, keyId, keyValue = "", "", ""
							break
						}
					}
				}
			}
		}
	}

	log.Printf("Loaded %d streams from %s\n", p.m3uCache.StreamCount(), p.config.data.Playlist)
	return nil
}

func (p *PlaylistHandler) playlistRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	authParts := strings.SplitN(authHeader, " ", 2)
	token := authParts[1]

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = r.URL.Scheme
	}
	if scheme == "" {
		scheme = "http"
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("#EXTM3U\n"))

	active_channels := p.getActiveChannels()
	if len(active_channels) == 0 {
		w.Write([]byte("#EXT-X-ENDLIST\n"))
		return
	}

	// Write the playlist
	for _, channel := range active_channels {
		if !channel.sources.Active() {
			continue
		}

		uri := fmt.Sprintf("%s://%s/%s/%s/%s", scheme, r.Host, token, channel.tvgId, channel.sources.MasterPlaylist())

		entry := m3uparser.M3UEntry{
			URI:   uri,
			Title: channel.sources.MediaName(),
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, channel.sources.M3UTags()...)
		if !channel.sources.IsRadio() {
			entry.AddTag("KODIPROP", "inputstream=inputstream.adaptive")
			entry.AddTag("KODIPROP", "inputstream.adaptive.manifest_type=hls")
		}
		w.Write([]byte(entry.String() + "\n"))
	}
}

func (p *PlaylistHandler) getActiveChannels() []*streamEntry {
	// get a list of all active streams
	p.channelsMux.Lock()
	active_channels := make([]*streamEntry, 0)
	for _, channel := range p.channels {
		if channel.sources.Active() {
			active_channels = append(active_channels, channel)
		}
	}
	p.channelsMux.Unlock()
	// Sort channels by index
	for i := 0; i < len(active_channels); i++ {
		for j := i + 1; j < len(active_channels); j++ {
			if active_channels[i].index > active_channels[j].index {
				active_channels[i], active_channels[j] = active_channels[j], active_channels[i]
			}
		}
	}
	return active_channels
}

func (p *PlaylistHandler) Load(ctx context.Context) error {

	if err := p.loadFromSource(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	streamsChan := make(chan *streamEntry)
	stopWorkers := make(chan bool)

	for i := 0; i < p.config.data.NumWorkers; i++ {
		wg.Add(1)
		go p.monitorWorker(streamsChan, stopWorkers, &wg)
	}

	go func() {
		for i, entry := range p.m3uCache.Entries {
			select {
			case <-ctx.Done():
				stopWorkers <- true
				wg.Wait()
				return
			default:
				if entry.URI == "" {
					continue
				}

				tvgId := entry.ExtInfTags.GetValue("tvg-id")
				if tvgId == "" {
					tvgId = entry.Title
				}

				radio := entry.ExtInfTags.GetValue("radio")
				if tvgId == "" && radio == "" {
					log.Printf("No tvg-id or radio tag found for %s, skipping\n", entry.URI)
					continue
				}

				channel, ok := p.channels[tvgId]
				if !ok {
					p.channelsMux.Lock()
					p.channels[tvgId] = &streamEntry{
						index:   i,
						tvgId:   tvgId,
						sources: streamSources.CreateSources(),
					}
					channel = p.channels[tvgId]
					p.channelsMux.Unlock()
				}

				if channel.sources.SourceExists(entry) {
					log.Printf("Stream source already exists: %s\n", entry.URI)
					continue
				}

				log.Printf("Adding stream source for %s, for channel %s\n", entry.URI, tvgId)
				channel.sources.AddSource(entry, p.config.data.Timeout)
				streamsChan <- channel
			}
		}
		close(streamsChan)
	}()

	wg.Wait()

	return nil
}

func (p *PlaylistHandler) streamRequest(w http.ResponseWriter, r *http.Request) {

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

	p.channelsMux.Lock()
	defer p.channelsMux.Unlock()

	channel, ok := p.channels[channelId]
	if !ok {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	if !channel.sources.Active() {
		http.Error(w, "Stream not active", http.StatusNotFound)
		return
	}

	channel.sources.Serve(w, r, p.config.data.Timeout)
}

func (p *PlaylistHandler) monitorWorker(streams <-chan *streamEntry, stop <-chan bool, wg *sync.WaitGroup) {

	defer wg.Done()
	for stream := range streams {
		select {
		case <-stop:
			return
		default:
			stream.sources.HealthCheck(p.config.data.Timeout)
			if stream.sources.GetActiveSource() == nil {
				continue
			}
			if !stream.sources.Active() {
				log.Printf("Stream %s is not active\n", stream.sources.GetActiveSource().MediaName())
			}
		}
	}
}
