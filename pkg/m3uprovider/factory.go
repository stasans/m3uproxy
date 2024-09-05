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
	Providers         map[string]interface{} `json:"providers"`
	ProvidersPriority []string               `json:"providers_priority"`
	ChannelOrder      []string               `json:"channel_order"`
	Overrides         []entryOverride        `json:"overrides"`
}

func NewProvider(name, config string) types.M3UProvider {
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

	configData, err := os.ReadFile(path)
	if err != nil {

		return nil, err
	}

	config := playlistConfig{}
	err = json.Unmarshal([]byte(configData), &config)
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
		log.Printf("Provider: %s\n", providerName)

		providerConfig := config.Providers[providerName]
		providerConfigData, err := json.Marshal(providerConfig)
		if err != nil {
			return nil, err
		}

		if providerName == "iptv.org" {
			provider := iptvorg.NewIPTVOrgProvider(string(providerConfigData))
			if provider == nil {
				return nil, err
			}
			playlists[providerName] = provider.GetPlaylist()
			continue
		}

		if providerName == "file" {
			provider := file.NewM3UFileProvider(string(providerConfigData))
			if provider == nil {
				return nil, err
			}
			playlists[providerName] = provider.GetPlaylist()
			continue
		}

		return nil, errors.New("provider not available")
	}

	log.Printf("%d playlists loaded", len(playlists))
	log.Println("Merging playlists according to the priority defined, duplicates will be skipped")
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
				log.Printf("Duplicate entry: %s, skipping.", entry.Title)
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
				log.Printf("Channel %s not found, skipping override", override.Channel)
				continue
			}
			log.Printf("Applying override for channel %s", entry.Title)
			if override.Disabled {
				log.Printf("Disabling channel %s", entry.Title)
				masterPlaylist.RemoveEntryByTvgTag("tvg-id", override.Channel)
				continue
			}
			if override.URL != "" {
				log.Printf("Overriding URL for channel %s", entry.Title)
				entry.URI = override.URL
			}
			if len(override.Headers) > 0 {
				log.Printf("Adding headers for channel %s", entry.Title)
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
		log.Println("Ordering playlist by channel order")
		orderedPlaylist := make(m3uparser.M3UEntries, 0)
		for _, channel := range config.ChannelOrder {
			entry := masterPlaylist.GetEntryByTvgTag("tvg-id", channel)
			if entry != nil {
				orderedPlaylist = append(orderedPlaylist, *entry)
				masterPlaylist.RemoveEntryByTvgTag("tvg-id", channel)
			}
		}
		orderedPlaylist = append(orderedPlaylist, masterPlaylist.Entries...)
		masterPlaylist.Entries = orderedPlaylist
	}

	return &masterPlaylist, nil
}
