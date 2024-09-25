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
	"encoding/json"
	"os"
)

type GeoIPConfig struct {
	Database         string   `json:"database"`
	Whitelist        []string `json:"whitelist,omitempty"`
	InternalNetworks []string `json:"internal_networks,omitempty"`
}

type SecurityConfig struct {
	GeoIP              GeoIPConfig `json:"geoip,omitempty"`
	AllowedCORSDomains []string    `json:"allowed_cors_domains,omitempty"`
}

type ServerConfig struct {
	Port       int             `json:"port"`
	Playlist   string          `json:"playlist"`
	Epg        string          `json:"epg"`
	Timeout    int             `json:"default_timeout,omitempty"`
	NumWorkers int             `json:"num_workers,omitempty"`
	ScanTime   int             `json:"scan_time,omitempty"`
	Security   SecurityConfig  `json:"security,omitempty"`
	Auth       json.RawMessage `json:"auth"`
	LogFile    string          `json:"log_file,omitempty"`
}

var (
	Config     *ServerConfig
	ConfigPath string
)

func (c *ServerConfig) Merge(other ServerConfig) {
	if other.Timeout != 0 {
		c.Timeout = other.Timeout
	}
	if other.NumWorkers != 0 {
		c.NumWorkers = other.NumWorkers
	}
	if other.ScanTime != 0 {
		c.ScanTime = other.ScanTime
	}
	if len(other.Security.GeoIP.Whitelist) > 0 {
		c.Security.GeoIP.Whitelist = other.Security.GeoIP.Whitelist
	}
	if len(other.Security.GeoIP.InternalNetworks) > 0 {
		c.Security.GeoIP.InternalNetworks = other.Security.GeoIP.InternalNetworks
	}
}

func LoadServerConfig(path string) error {

	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	err = json.NewDecoder(file).Decode(&Config)
	if err != nil {
		return err
	}

	ConfigPath = path

	return nil
}

func SaveServerConfig(config ServerConfig) error {

	Config.Merge(config)

	file, err := os.Create(ConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(Config)
}
