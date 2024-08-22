package channelstore

import (
	"fmt"
	"m3u-proxy/pkg/m3uparser"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	channels       = make([]Channel, 0)
	channelsCache  = make([]ChannelsCacheData, 0)
	defaultTimeout = 3
)

func validateURL(urlPath string) bool {
	_, err := url.Parse(urlPath)
	return err == nil
}

func validateChannel(channel Channel, checkOnline bool, timeout int) bool {
	if channel.Entry.URI == "" || channel.Name == "" {
		return false
	}
	if !validateURL(channel.Entry.URI) {
		return false
	}
	if checkOnline {
		client := http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}
		req, err := client.Get(channel.Entry.URI)
		if err != nil {
			return false
		}
		if req.StatusCode != http.StatusOK {
			return false
		}
	}

	return true
}

func LoadPlaylist(playlist *m3uparser.M3UPlaylist, checkOnline bool) error {

	for _, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		channel := Channel{
			Entry: entry,
			Name:  entry.Title,
		}

		if !validateChannel(channel, checkOnline, defaultTimeout) {
			continue
		}

		parsedURL, _ := url.Parse(channel.Entry.URI)
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			baseURL += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		channels = append(channels, channel)
		channelsCache = append(channelsCache, ChannelsCacheData{baseURL: baseURL, active: true})
	}

	return nil
}

func ExportPlaylist(host, path, token string) m3uparser.M3UPlaylist {
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

func GetChannels() []Channel {
	return channels
}

func GetChannel(index int) (Channel, error) {
	if index < 0 || index >= len(channels) {
		return Channel{}, fmt.Errorf("Channel not found")
	}
	return channels[index], nil
}

func GetChannelCacheData(index int) (ChannelsCacheData, error) {
	if index < 0 || index >= len(channelsCache) {
		return ChannelsCacheData{}, fmt.Errorf("Channel cache data not found")
	}
	return channelsCache[index], nil
}

func GetChannelCount() int {
	return len(channels)
}

func ClearChannels() {
	channels = make([]Channel, 0)
	channelsCache = make([]ChannelsCacheData, 0)
}

func SetChannelActive(index int, active bool) error {
	if index < 0 || index >= len(channelsCache) {
		return fmt.Errorf("Channel cache data not found")
	}
	channelsCache[index].active = active
	return nil
}

func GetChannelActive(index int) (bool, error) {
	if index < 0 || index >= len(channelsCache) {
		return false, fmt.Errorf("Channel cache data not found")
	}
	return channelsCache[index].active, nil
}

func GetChannelStreamURL(index int, path string, query string) (string, error) {
	if index < 0 || index >= len(channels) {
		return "", fmt.Errorf("Channel not found")
	}
	channel := channels[index]
	if path == "playlist.m3u8" {
		return channel.Entry.URI, nil
	}
	cacheData := channelsCache[index]
	serviceURL := cacheData.baseURL + "/" + path
	if query != "" {
		serviceURL += "?" + query
	}
	return serviceURL, nil
}

func GetChannelStream(index int, path string, query string) (*http.Response, error) {
	serviceURL, err := GetChannelStreamURL(index, path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func GetChannelStreamWithHeaders(index int, path string, query string, headers http.Header) (*http.Response, error) {
	serviceURL, err := GetChannelStreamURL(index, path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return http.DefaultClient.Do(req)
}

func GetChannelStreamWithTimeout(index int, path string, query string, timeout int) (*http.Response, error) {
	serviceURL, err := GetChannelStreamURL(index, path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	return client.Do(req)
}

func GetChannelStreamWithHeadersAndTimeout(index int, path string, query string, headers http.Header, timeout int) (*http.Response, error) {
	serviceURL, err := GetChannelStreamURL(index, path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	return client.Do(req)
}

func ValidateChannels() {
	for i := 0; i < len(channels); i++ {
		channelsCache[i].active = validateChannel(channels[i], true, defaultTimeout)
	}
}
