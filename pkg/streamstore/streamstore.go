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
	"bufio"
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
	"github.com/grafov/m3u8"

	"github.com/gorilla/mux"
)

type StreamCache struct {
	index    int
	m3u      m3uparser.M3UEntry
	prefix   string
	active   bool
	playlist *m3u8.MasterPlaylist
	mux      *sync.Mutex
	client   http.Client
	headers  map[string]string
}

var (
	streamsCaches   = make([]StreamCache, 0)
	defaultTimeout  = 3
	streamsStoreMux sync.Mutex
)

func (stream *StreamCache) HttpHeaders() map[string]string {

	if stream.headers != nil {
		stream.headers = make(map[string]string)
		m3uproxyTags := stream.m3u.SearchTags("M3UPROXYHEADER")
		for _, tag := range m3uproxyTags {
			parts := strings.Split(tag.Value, "=")
			if len(parts) == 2 {
				stream.headers[parts[0]] = parts[1]
			}
		}
		vlcTags := stream.m3u.SearchTags("EXTVLCOPT")
		for _, tag := range vlcTags {
			parts := strings.Split(tag.Value, "=")
			if len(parts) == 2 {
				switch parts[0] {
				case "http-user-agent":
					stream.headers["User-Agent"] = parts[1]
				case "http-referrer":
					stream.headers["Referer"] = parts[1]
				default:
				}
			}
		}
	}
	return stream.headers
}

func (stream *StreamCache) HealthCheck(timeout int) {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	stream.active = false
	stream.playlist = nil
	stream.client.Timeout = time.Duration(timeout) * time.Second

	req, err := http.NewRequest("GET", stream.m3u.URI, nil)
	if err != nil {
		return
	}

	headers := stream.HttpHeaders()
	for key := range headers {
		req.Header.Add(key, headers[key])
	}

	resp, err := stream.client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	status := resp.StatusCode / 100
	if status != 2 {
		return
	}

	streamUrl, _ := url.Parse(stream.m3u.URI)

	// Check if the URL has changed (redirects) and update the stream URL
	if *resp.Request.URL != *streamUrl {

		prefix := ""

		if strings.LastIndex(resp.Request.URL.Path, "/") != -1 {
			prefix += resp.Request.URL.Path[:strings.LastIndex(resp.Request.URL.Path, "/")]
		}

		stream.m3u.URI = resp.Request.URL.String()
		stream.prefix = prefix
	}

	// Check if the stream is a valid m3u8 playlist
	p, listType, err := m3u8.DecodeWith(bufio.NewReader(resp.Body), true, []m3u8.CustomDecoder{})
	if err != nil {
		return
	}

	stream.active = true
	if listType == m3u8.MASTER {
		stream.playlist = p.(*m3u8.MasterPlaylist)
	}
}

func (stream *StreamCache) Serve(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	path := vars["path"]

	stream.mux.Lock()
	defer stream.mux.Unlock()

	if !stream.active {
		return errors.New("stream not active")
	}

	streamURL, _ := url.Parse(stream.m3u.URI)

	if path == "playlist.m3u8" {
		if stream.playlist != nil {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(stream.playlist.String()))
			return nil
		}
	} else {
		if stream.prefix != "" {
			streamURL.Path = stream.prefix + "/" + path
		} else {
			streamURL.Path = path
		}
		streamURL.RawQuery = r.URL.RawQuery
	}

	req, _ := http.NewRequest(r.Method, streamURL.String(), nil)

	headers := stream.HttpHeaders()
	for key := range headers {
		req.Header.Add(key, headers[key])
	}

	resp, err := stream.client.Do(req)

	if err != nil {
		return errors.New("failed to fetch stream")
	}

	defer resp.Body.Close()

	code := resp.StatusCode / 100
	if code != 2 {
		return errors.New("failed to fetch stream")
	}

	if resp.StatusCode == http.StatusNoContent {
		return errors.New("no content")
	}

	for key := range resp.Header {
		w.Header().Add(key, resp.Header.Get(key))
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil

}

func AddStreams(playlist *m3uparser.M3UPlaylist) error {

	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()

	streamsCaches = make([]StreamCache, 0)

	for i, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		parsedURL, err := url.Parse(entry.URI)
		if err != nil {
			log.Printf("Failed to parse URL: %s\n", entry.URI)
			continue
		}

		prefix := ""

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			prefix += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		stream := StreamCache{
			index:    i,
			m3u:      entry,
			prefix:   prefix,
			active:   false,
			playlist: nil,
			mux:      &sync.Mutex{},
		}
		streamsCaches = append(streamsCaches, stream)
	}

	return nil
}

func BuildM3UPlaylist(host, path, token string) m3uparser.M3UPlaylist {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	playlist := m3uparser.M3UPlaylist{
		Entries: make([]m3uparser.M3UEntry, 0),
	}
	for i, stream := range streamsCaches {
		entry := m3uparser.M3UEntry{
			URI:   fmt.Sprintf("http://%s/%s/%s/%d/playlist.m3u8", host, path, token, i),
			Title: stream.m3u.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3u.Tags...)
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

func StreamCount() int {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	return len(streamsCaches)
}

func Clear() {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	streamsCaches = make([]StreamCache, 0)
}

func ActivateStream(index int, active bool) error {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	if index < 0 || index >= len(streamsCaches) {
		return fmt.Errorf("stream cache data not found")
	}
	streamsCaches[index].mux.Lock()
	streamsCaches[index].active = active
	streamsCaches[index].mux.Unlock()
	return nil
}

func StreamIsActive(index int) bool {
	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()
	if index < 0 || index >= len(streamsCaches) {
		return false
	}
	return streamsCaches[index].active
}

func MonitorStreams() {
	for i := 0; i < len(streamsCaches); i++ {
		streamsCaches[i].HealthCheck(defaultTimeout)
		if !streamsCaches[i].active {
			log.Printf("Stream %d is offline\n", i)
		}
	}
}

func Serve(w http.ResponseWriter, r *http.Request) error {

	vars := mux.Vars(r)

	streamID, err := strconv.Atoi(vars["streamId"])
	if err != nil {
		return errors.New("invalid stream ID")
	}

	return streamsCaches[streamID].Serve(w, r)
}
