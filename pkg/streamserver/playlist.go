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

	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("#EXTM3U\n"))
	for i, stream := range streams {
		if !stream.stream.Active() {
			continue
		}

		uri := fmt.Sprintf("%s://%s/%s/%d/%s", scheme, r.Host, token, i, stream.stream.MasterPlaylist())

		entry := m3uparser.M3UEntry{
			URI:   uri,
			Title: stream.stream.MediaName(),
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.stream.M3UTags()...)
		if !stream.stream.IsRadio() {
			entry.AddTag("KODIPROP", "inputstream=inputstream.adaptive")
			entry.AddTag("KODIPROP", "inputstream.adaptive.manifest_type=hls")
		}
		w.Write([]byte(entry.String() + "\n"))
	}
}
