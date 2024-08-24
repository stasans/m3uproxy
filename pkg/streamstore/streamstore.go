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
package streamstore

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

type Stream struct {
	m3uEntry m3uparser.M3UEntry
	baseURL  string
	active   bool
}

var (
	streams         = make([]Stream, 0)
	defaultTimeout  = 3
	streamsStoreMux sync.Mutex
)

func checkStreamOnline(stream Stream, timeout int) bool {
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Get(stream.m3uEntry.URI)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func LoadPlaylist(playlist *m3uparser.M3UPlaylist) error {

	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()

	for _, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		parsedURL, err := url.Parse(entry.URI)
		if err != nil {
			log.Printf("Failed to parse URL: %s\n", entry.URI)
			continue
		}
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			baseURL += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		stream := Stream{
			m3uEntry: entry,
			baseURL:  baseURL,
			active:   true,
		}
		streams = append(streams, stream)

		log.Printf("Loaded stream: %s\n", entry.Title)
	}

	return nil
}

func ExportPlaylist(host, path, token string) m3uparser.M3UPlaylist {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	playlist := m3uparser.M3UPlaylist{
		Entries: make([]m3uparser.M3UEntry, 0),
	}
	for i, stream := range streams {
		entry := m3uparser.M3UEntry{
			URI:   fmt.Sprintf("http://%s/%s/%s/%d/playlist.m3u8", host, path, token, i),
			Title: stream.m3uEntry.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3uEntry.Tags...)
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

func GetStreamCount() int {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	return len(streams)
}

func ClearStreams() {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	streams = make([]Stream, 0)
}

func SetStreamActive(index int, active bool) error {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	if index < 0 || index >= len(streams) {
		return fmt.Errorf("Stream cache data not found")
	}
	streams[index].active = active
	return nil
}

func IsStreamActive(index int) bool {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	if index < 0 || index >= len(streams) {
		return false
	}
	return streams[index].active
}

func CheckStreams() {
	streamsActive := make([]bool, len(streams))
	for i := 0; i < len(streams); i++ {
		streamsActive[i] = checkStreamOnline(streams[i], defaultTimeout)
		if !streamsActive[i] {
			log.Printf("Stream %d is offline\n", i)
		}
	}
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	for i := 0; i < len(streams); i++ {
		streams[i].active = streamsActive[i]
	}
}

func StreamStreamHandler(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	streamID, err := strconv.Atoi(vars["streamId"])
	if err != nil {
		return errors.New("invalid stream ID")
	}

	if !IsStreamActive(streamID) {
		log.Printf("Stream %d not available\n", streamID)
		return errors.New("Stream not available")
	}

	path := vars["path"]
	query := r.URL.RawQuery
	serviceURL := ""

	streamsStoreMux.Lock()
	stream := streams[streamID]
	streamsStoreMux.Unlock()

	// If the path is the playlist, return the original playlist URL
	if path == "playlist.m3u8" {

		serviceURL = stream.m3uEntry.URI

	} else {

		serviceURL = stream.baseURL + "/" + path
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
		return errors.New("failed to fetch stream")
	}

	code := resp.StatusCode / 100
	if code == 2 || code == 3 {

		if resp.StatusCode == http.StatusNoContent {
			return errors.New("no content")
		}

		defer resp.Body.Close()

		for key := range resp.Header {
			w.Header().Add(key, resp.Header.Get(key))
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)

		return nil
	}

	log.Printf("Failed to fetch stream: %d\n", resp.StatusCode)
	return errors.New("failed to fetch stream")
}
