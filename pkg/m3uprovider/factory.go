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

type entryOverride struct {
	Channel  string            `json:"channel"`
	URL      string            `json:"url,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
}

type playlistConfig struct {
	Providers         map[string]json.RawMessage `json:"providers"`
	ProvidersPriority []string                   `json:"providers_priority"`
	ChannelOrder      []string                   `json:"channel_order"`
	Overrides         []entryOverride            `json:"overrides"`
}

func NewProvider(name string, config json.RawMessage) types.M3UProvider {
	switch name {
	case "iptv.org":
		return iptvorg.NewIPTVOrgProvider(config)
	case "file":
		return file.NewM3UFileProvider(config)
	default:
		return nil
	}
}

func LoadPlaylist(path string) (*m3uparser.M3UPlaylist, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := playlistConfig{}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	providersPriority := make([]string, 0)
	if config.ProvidersPriority != nil {
		if len(config.ProvidersPriority) != len(config.Providers) {
			return nil, err
		}
		providersPriority = append(providersPriority, config.ProvidersPriority...)
	} else {
		for providerName := range config.Providers {
			providersPriority = append(providersPriority, providerName)
		}
	}

	playlists := make(map[string]*m3uparser.M3UPlaylist)
	for _, providerName := range providersPriority {

		provider := NewProvider(providerName, config.Providers[providerName])
		if provider == nil {
			return nil, errors.New("provider not available '" + providerName + "'")
		}

		log.Printf("Provider: %s\n", providerName)
		playlists[providerName] = provider.GetPlaylist()
	}

	log.Printf("%d playlists loaded", len(playlists))
	log.Println("Merging playlists according to the priority defined, duplicates will be skipped.")

	masterPlaylist := m3uparser.M3UPlaylist{
		Version: 3,
		Entries: make(m3uparser.M3UEntries, 0),
		Tags:    make(m3uparser.M3UTags, 0),
	}

	for _, playlist := range playlists {
		for _, entry := range playlist.Entries {
			tvgId := entry.TVGTags.GetValue("tvg-id")
			if tvgId == "" {
				masterPlaylist.Entries = append(masterPlaylist.Entries, entry)
				continue
			}
			if masterPlaylist.GetEntryByTvgTag("tvg-id", tvgId) != nil {
				log.Printf("Duplicate entry: '%s', skipping.", entry.Title)
				continue
			}
			masterPlaylist.Entries = append(masterPlaylist.Entries, entry)
		}
	}

	if len(config.Overrides) > 0 {
		log.Println("Applying overrides")
		for _, override := range config.Overrides {
			entry := masterPlaylist.GetEntryByTvgTag("tvg-id", override.Channel)
			if entry == nil {
				continue
			}
			log.Printf("Applying override for channel '%s'.", entry.Title)
			if override.Disabled {
				masterPlaylist.RemoveEntryByTvgTag("tvg-id", override.Channel)
				continue
			}
			if override.URL != "" {
				entry.URI = override.URL
			}
			if len(override.Headers) > 0 {
				for k, v := range override.Headers {
					entry.Tags = append(entry.Tags, m3uparser.M3UTag{
						Tag:   "M3UPROXYHEADER",
						Value: k + "=" + v,
					})
				}
			}
			masterPlaylist.RemoveEntryByTvgTag("tvg-id", override.Channel)
			masterPlaylist.Entries = append(masterPlaylist.Entries, *entry)
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
