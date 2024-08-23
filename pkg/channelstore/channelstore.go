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
package channelstore

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"

	"github.com/gorilla/mux"
)

type Channel struct {
	Entry m3uparser.M3UEntry `json:"entry"`
	Name  string             `json:"name"`
}

type ChannelsCacheData struct {
	baseURL string
	active  bool
}

var (
	channels         = make([]Channel, 0)
	channelsCache    = make([]ChannelsCacheData, 0)
	defaultTimeout   = 3
	channelsStoreMux sync.Mutex
)

func validateURL(urlPath string) bool {
	_, err := url.Parse(urlPath)
	return err == nil
}

func validateChannel(channel Channel) bool {
	if channel.Entry.URI == "" || channel.Name == "" {
		return false
	}
	if !validateURL(channel.Entry.URI) {
		return false
	}
	return true
}

func checkChannelOnline(channel Channel, timeout int) bool {
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Get(channel.Entry.URI)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func LoadPlaylist(playlist *m3uparser.M3UPlaylist) error {

	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()

	for _, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		channel := Channel{
			Entry: entry,
			Name:  entry.Title,
		}

		if !validateChannel(channel) {
			continue
		}

		parsedURL, _ := url.Parse(channel.Entry.URI)
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			baseURL += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		log.Printf("Loaded channel: %s\n", channel.Name)

		channels = append(channels, channel)
		channelsCache = append(channelsCache, ChannelsCacheData{
			baseURL: baseURL,
			active:  true,
		})
	}

	return nil
}

func ExportPlaylist(host, path, token string) m3uparser.M3UPlaylist {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	playlist := m3uparser.M3UPlaylist{
		Entries: make([]m3uparser.M3UEntry, 0),
	}
	for i, channel := range channels {
		entry := m3uparser.M3UEntry{
			URI:   fmt.Sprintf("http://%s/%s/%s/%d/playlist.m3u8", host, path, token, i),
			Title: channel.Name,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, channel.Entry.Tags...)
		playlist.Entries = append(playlist.Entries, entry)
	}
	return playlist
}

func SetDefaultTimeout(timeout int) {
	defaultTimeout = timeout
}

func GetDefaultTimeout() int {
	return defaultTimeout
}

func GetChannel(index int) (Channel, error) {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	if index < 0 || index >= len(channels) {
		return Channel{}, fmt.Errorf("Channel not found")
	}
	return channels[index], nil
}

func GetChannelCount() int {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	return len(channels)
}

func ClearChannels() {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	channels = make([]Channel, 0)
	channelsCache = make([]ChannelsCacheData, 0)
}

func SetChannelActive(index int, active bool) error {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	if index < 0 || index >= len(channelsCache) {
		return fmt.Errorf("Channel cache data not found")
	}
	channelsCache[index].active = active
	return nil
}

func IsChannelActive(index int) bool {
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	if index < 0 || index >= len(channelsCache) {
		return false
	}
	return channelsCache[index].active
}

func CheckChannels() {
	channelsActive := make([]bool, len(channels))
	for i := 0; i < len(channels); i++ {
		channelsActive[i] = checkChannelOnline(channels[i], defaultTimeout)
		if !channelsActive[i] {
			log.Printf("Channel %d is offline\n", i)
		}
	}
	channelsStoreMux.Lock()
	defer channelsStoreMux.Unlock()
	for i := 0; i < len(channels); i++ {
		channelsCache[i].active = channelsActive[i]
	}
}

func ChannelHandleStream(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	channelID, err := strconv.Atoi(vars["channelId"])
	if err != nil {
		return errors.New("invalid channel ID")
	}

	if !IsChannelActive(channelID) {
		log.Printf("Channel %d not available\n", channelID)
		return errors.New("Channel not available")
	}

	path := vars["path"]
	query := r.URL.RawQuery
	serviceURL := ""

	channelsStoreMux.Lock()
	channel := channels[channelID]
	cacheData := channelsCache[channelID]
	channelsStoreMux.Unlock()

	// If the path is the playlist, return the original playlist URL
	if path == "playlist.m3u8" {

		serviceURL = channel.Entry.URI

	} else {

		serviceURL = cacheData.baseURL + "/" + path
		if query != "" {
			serviceURL += "?" + query
		}
	}

	req, _ := http.NewRequest(r.Method, serviceURL, r.Body)

	for key := range r.Header {
		if key == "Host" {
			continue
		} else {
			req.Header.Add(key, r.Header.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return errors.New("failed to fetch channel")
	}

	defer resp.Body.Close()

	for key := range resp.Header {
		w.Header().Add(key, resp.Header.Get(key))
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}
