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
package m3uprovider

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider/file"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider/iptvorg"
	types "github.com/a13labs/m3uproxy/pkg/m3uprovider/types"
)

type OverrideEntry struct {
	ChannelName      string            `json:"name,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Disabled         bool              `json:"disabled,omitempty"`
	HttpProxy        string            `json:"http_proxy,omitempty"`
	ForceKodiHeaders bool              `json:"kodi,omitempty"`
}

type ProviderConfig struct {
	Provider string          `json:"provider"`
	Config   json.RawMessage `json:"config"`
}

type PlaylistConfig struct {
	Providers         map[string]ProviderConfig `json:"providers"`
	ProvidersPriority []string                  `json:"providers_priority,omitempty"`
	ChannelOrder      []string                  `json:"channel_order,omitempty"`
	Overrides         map[string]OverrideEntry  `json:"overrides,omitempty"`
}

func NewProvider(config ProviderConfig) types.M3UProvider {

	switch config.Provider {
	case "iptv.org":
		return iptvorg.NewIPTVOrgProvider(config.Config)
	case "file":
		return file.NewM3UFileProvider(config.Config)
	default:
		return nil
	}
}

func Load(config PlaylistConfig) (*m3uparser.M3UPlaylist, error) {

	providersPriority := make([]string, 0)
	if config.ProvidersPriority != nil {
		if len(config.ProvidersPriority) != len(config.Providers) {
			return nil, errors.New("providers_priority and providers must have the same length")
		}
		providersPriority = append(providersPriority, config.ProvidersPriority...)
	} else {
		for providerName := range config.Providers {
			providersPriority = append(providersPriority, providerName)
		}
	}

	masterPlaylist := m3uparser.M3UPlaylist{
		Version: 3,
		Entries: make(m3uparser.M3UEntries, 0),
		Tags:    make(m3uparser.M3UTags, 0),
	}

	for _, providerName := range providersPriority {

		provider := NewProvider(config.Providers[providerName])
		if provider == nil {
			return nil, errors.New("provider not available '" + providerName + "'")
		}

		log.Printf("Provider: %s\n", providerName)
		playlist := provider.GetPlaylist()
		for _, entry := range playlist.Entries {
			tvgId := entry.TVGTags.GetValue("tvg-id")
			if tvgId == "" {
				masterPlaylist.Entries = append(masterPlaylist.Entries, entry)
				continue
			}
			if masterPlaylist.SearchEntryByTvgTag("tvg-id", tvgId) != nil {
				log.Printf("Duplicate entry: '%s', skipping.", entry.Title)
				continue
			}
			override, ok := config.Overrides[tvgId]
			if ok && override.Disabled {
				log.Printf("Channel '%s' is disabled, skipping.", entry.Title)
				continue
			}
			if ok && override.ChannelName != "" {
				entry.Title = override.ChannelName
			}
			if ok && override.URL != "" {
				entry.URI = override.URL
			}
			if ok && len(override.Headers) > 0 {
				for k, v := range override.Headers {
					entry.Tags = append(entry.Tags, m3uparser.M3UTag{
						Tag:   "M3UPROXYHEADER",
						Value: k + "=" + v,
					})
				}
			}
			if ok && override.HttpProxy != "" {
				entry.Tags = append(entry.Tags, m3uparser.M3UTag{
					Tag:   "M3UPROXYTRANSPORT",
					Value: "proxy=" + override.HttpProxy,
				})
			}
			if ok && override.ForceKodiHeaders {
				entry.Tags = append(entry.Tags, m3uparser.M3UTag{
					Tag:   "M3UPROXYOPT",
					Value: "forcekodiheaders",
				})
			}
			masterPlaylist.Entries = append(masterPlaylist.Entries, entry)
		}

	}

	if len(config.ChannelOrder) > 0 {
		log.Println("Ordering playlist by provided channel order.")

		for needle, channel := range config.ChannelOrder {
			for pos := needle; pos < len(masterPlaylist.Entries); pos++ {
				if masterPlaylist.Entries[pos].TVGTags.GetValue("tvg-id") == channel {
					if needle == pos {
						break
					}
					masterPlaylist.Entries[needle], masterPlaylist.Entries[pos] = masterPlaylist.Entries[pos], masterPlaylist.Entries[needle]
					break
				}
			}
		}
	}

	return &masterPlaylist, nil
}

func LoadFromFile(path string) (*m3uparser.M3UPlaylist, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := PlaylistConfig{}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return Load(config)
}
