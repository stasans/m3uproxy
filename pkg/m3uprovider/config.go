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
	"os"
)

type OverrideEntry struct {
	ChannelName      string            `json:"name,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Disabled         bool              `json:"disabled,omitempty"`
	HttpProxy        string            `json:"http_proxy,omitempty"`
	ForceKodiHeaders bool              `json:"kodi,omitempty"`
	DisableRemap     bool              `json:"disable_remap,omitempty"`
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

func (c *PlaylistConfig) Merge(other PlaylistConfig) {
	if other.Providers != nil {
		c.Providers = other.Providers
	}
	if other.ProvidersPriority != nil {
		c.ProvidersPriority = other.ProvidersPriority
	}
	if other.ChannelOrder != nil {
		c.ChannelOrder = other.ChannelOrder
	}
	if other.Overrides != nil {
		c.Overrides = other.Overrides
	}
}

func (c *PlaylistConfig) SaveToFile(file string) error {

	if s, err := os.Stat(file); err == nil && !s.IsDir() {
		os.Remove(file)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	f, err := os.OpenFile(file, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	if err = e.Encode(c); err != nil {
		return err
	}

	return nil
}

func (c *PlaylistConfig) Validate() bool {
	_, err := Load(c)
	return err == nil
}

func LoadPlaylistConfig(path string) (*PlaylistConfig, error) {

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

	return &config, nil
}
