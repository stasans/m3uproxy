package streamserver

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/gorilla/mux"
)

var (
	m3uCache       *m3uparser.M3UPlaylist
	playlistConfig *m3uprovider.PlaylistConfig
	licenseManger  *streamLicenseManager
)

func LoadPlaylist() error {
	var err error
	playlistConfig, err = m3uprovider.LoadPlaylistConfig(Config.Playlist)
	if err != nil {
		return err
	}

	m3uCache, err = m3uprovider.Load(playlistConfig)
	if err != nil {
		return err
	}

	// Load licenses
	// For now we just support processing clearkey licenses and KODIPROP tags
	for _, entry := range m3uCache.Entries {
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

	log.Printf("Loaded %d streams from %s\n", m3uCache.StreamCount(), Config.Playlist)
	return nil
}

func SavePlaylist(p m3uprovider.PlaylistConfig) error {
	if !p.Validate() {
		return fmt.Errorf("invalid playlist config")
	}
	playlistConfig = &p
	return playlistConfig.SaveToFile(Config.Playlist)
}

func registerPlaylistRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/streams.m3u", basicAuth(playlistRequest))
	return r
}

func playlistRequest(w http.ResponseWriter, r *http.Request) {

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

	active_channels := getActiveChannels()
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

func getActiveChannels() []*streamEntry {
	// get a list of all active streams
	channelsMux.Lock()
	active_channels := make([]*streamEntry, 0)
	for _, channel := range channels {
		if channel.sources.Active() {
			active_channels = append(active_channels, channel)
		}
	}
	channelsMux.Unlock()
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
