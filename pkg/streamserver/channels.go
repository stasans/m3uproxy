package streamserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/logger"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/a13labs/m3uproxy/pkg/sources"
	"github.com/gorilla/mux"
)

var (
	licenseManger *streamLicenseManager
)

type streamEntry struct {
	index   int
	tvgId   string
	sources sources.Sources
}

type ChannelsHandler struct {
	config         *ServerConfig
	m3uCache       *m3uparser.M3UPlaylist
	playlistConfig *m3uprovider.PlaylistConfig
	channelsMux    sync.Mutex
	channels       map[string]*streamEntry
}

func NewChannelsHandler(config *ServerConfig) *ChannelsHandler {
	return &ChannelsHandler{
		config:      config,
		channels:    make(map[string]*streamEntry),
		channelsMux: sync.Mutex{},
	}
}

func (p *ChannelsHandler) RegisterRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/channels.m3u", basicAuth(p.playlistRequest))
	r.HandleFunc("/drm/licensing", basicAuth(licenseKeysRequest))
	r.HandleFunc("/{token}/{channelId}/media/{path:.*}", p.mediaRequest)
	r.HandleFunc("/{token}/{channelId}/{path:.*}", p.manifestRequest)
	return r
}

func (p *ChannelsHandler) loadConfig() error {
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
							logger.Infof("Found license, adding license key with id %s", keyId)
							licenseManger.addLicense("clearkey", keyId, keyValue)
							keyType, keyId, keyValue = "", "", ""
							break
						}
					}
				}
			}
		}
	}

	logger.Infof("Loaded %d streams from %s", p.m3uCache.StreamCount(), p.config.data.Playlist)
	return nil
}

func (p *ChannelsHandler) getActiveChannels() []*streamEntry {
	// get a list of all active streams
	p.channelsMux.Lock()
	activeChannels := make([]*streamEntry, 0)
	for _, channel := range p.channels {
		if channel.sources.Active() {
			activeChannels = append(activeChannels, channel)
		}
	}
	p.channelsMux.Unlock()
	// Sort channels by index
	for i := 0; i < len(activeChannels); i++ {
		for j := i + 1; j < len(activeChannels); j++ {
			if activeChannels[i].index > activeChannels[j].index {
				activeChannels[i], activeChannels[j] = activeChannels[j], activeChannels[i]
			}
		}
	}
	return activeChannels
}

func (p *ChannelsHandler) Load(ctx context.Context) error {

	if err := p.loadConfig(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	streamsChan := make(chan *streamEntry)
	stopWorkers := make(chan bool)

	for i := 0; i < p.config.data.NumWorkers; i++ {
		wg.Add(1)
		go monitorWorker(streamsChan, stopWorkers, &wg)
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
					logger.Warnf("No tvg-id or radio tag found for %s, skipping", entry.URI)
					continue
				}

				channel, ok := p.channels[tvgId]
				if !ok {
					p.channelsMux.Lock()
					p.channels[tvgId] = &streamEntry{
						index:   i,
						tvgId:   tvgId,
						sources: sources.NewSources(),
					}
					channel = p.channels[tvgId]
					p.channelsMux.Unlock()
				}

				if channel.sources.SourceExists(entry) {
					logger.Warnf("Stream source already exists: %s", entry.URI)
					continue
				}

				logger.Infof("Adding stream source for %s, for channel %s", entry.URI, tvgId)
				added, err := channel.sources.AddSource(entry, p.config.data.Timeout)
				if err != nil {
					logger.Errorf("Error adding stream source for %s: %v", entry.URI, err)
					continue
				}
				if !added {
					logger.Warnf("Stream source already exists: %s", entry.URI)
					continue
				}
				streamsChan <- channel
			}
		}
		close(streamsChan)
	}()

	wg.Wait()

	return nil
}

func (p *ChannelsHandler) playlistRequest(w http.ResponseWriter, r *http.Request) {

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

	activeChannels := p.getActiveChannels()
	if len(activeChannels) == 0 {
		return
	}

	// Write the playlist
	for _, channel := range activeChannels {
		if !channel.sources.Active() {
			continue
		}

		tvgId := strings.ReplaceAll(channel.tvgId, " ", "%20")
		uri := fmt.Sprintf("%s://%s/%s/%s/%s", scheme, r.Host, token, tvgId, channel.sources.MasterPlaylist())

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

func (p *ChannelsHandler) manifestRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		logger.Errorf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.", r.URL.Path)
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

	channel.sources.ServeManifest(w, r, p.config.data.Timeout)
}

func (p *ChannelsHandler) mediaRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		logger.Errorf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.", r.URL.Path)
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

	channel.sources.ServeMedia(w, r, p.config.data.Timeout)
}

func monitorWorker(streams <-chan *streamEntry, stop <-chan bool, wg *sync.WaitGroup) {

	defer wg.Done()
	for stream := range streams {
		select {
		case <-stop:
			return
		default:
			stream.sources.HealthCheck()
			if stream.sources.GetActiveSource() == nil {
				continue
			}
			if !stream.sources.Active() {
				logger.Warnf("Stream %s is not active", stream.sources.GetActiveSource().MediaName())
			}
		}
	}
}
