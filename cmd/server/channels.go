package server

import (
	"fmt"
	"log"
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

type channelsCacheData struct {
	baseURL string
	active  bool
}

var (
	channels      = make([]Channel, 0)
	channelsCache = make([]channelsCacheData, 0)
)

func validateURL(urlPath string) bool {
	_, err := url.Parse(urlPath)
	return err == nil
}

func validateChannel(channel Channel, checkOnline bool) bool {
	if channel.Entry.URI == "" || channel.Name == "" {
		return false
	}
	if !validateURL(channel.Entry.URI) {
		return false
	}
	if checkOnline {
		client := http.Client{
			Timeout: 3 * time.Second,
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

func loadM3U(filePath string) error {

	playlist, err := m3uparser.ParseM3UFile(filePath)
	if err != nil {
		return err
	}

	for _, entry := range playlist.Entries {
		if entry.URI == "" {
			continue
		}
		channel := Channel{
			Entry: entry,
			Name:  entry.Title,
		}
		if !validateChannel(channel, false) {
			log.Printf("Invalid channel: %s\n", channel.Name)
			continue
		}
		parsedURL, _ := url.Parse(channel.Entry.URI)
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host
		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			baseURL += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		log.Printf("Loaded channel: %s\n", channel.Name)
		channels = append(channels, channel)
		channelsCache = append(channelsCache, channelsCacheData{baseURL: baseURL, active: true})

	}

	log.Printf("Loaded %d channels\n", len(channels))
	return nil
}

func generateM3U(host, token string) string {
	m3uContent := "#EXTM3U\n"
	for i, channel := range channels {

		for _, tag := range channel.Entry.Tags {
			m3uContent += fmt.Sprintf("#%s:%s\n", tag.Tag, tag.Value)
		}
		m3uContent += fmt.Sprintf("http://%s/channel/%s/%d/stream\n", host, token, i)
	}
	return m3uContent
}
