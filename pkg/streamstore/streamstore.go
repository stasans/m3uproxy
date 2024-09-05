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

type Stream struct {
	index    int
	m3uEntry m3uparser.M3UEntry
	basePath string
	active   bool
	playlist *m3u8.MasterPlaylist
	mux      *sync.Mutex
	headers  map[string]string
}

var (
	streams         = make([]Stream, 0)
	defaultTimeout  = 3
	streamsStoreMux sync.Mutex
)

func checkStreamOnline(stream *Stream, timeout int) {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	stream.active = false
	stream.playlist = nil
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequest("GET", stream.m3uEntry.URI, nil)
	if err != nil {
		return
	}

	for key := range stream.headers {
		req.Header.Add(key, stream.headers[key])
	}

	resp, err := client.Do(req)

	if err != nil {
		return
	}
	defer resp.Body.Close()

	status := resp.StatusCode / 100
	if status != 2 {
		return
	}

	realURL := resp.Request.URL.String()
	if realURL != stream.m3uEntry.URI {
		parsedURL, err := url.Parse(realURL)
		log.Printf("Redirected to: %s\n", realURL)

		if err != nil {
			log.Printf("Failed to parse URL: %s\n", realURL)
			return
		}

		basePath := ""

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			basePath += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		stream.m3uEntry.URI = realURL
		stream.basePath = basePath
	}

	p, listType, err := m3u8.DecodeWith(bufio.NewReader(resp.Body), true, []m3u8.CustomDecoder{})
	if err != nil {
		return
	}

	stream.active = true
	if listType == m3u8.MASTER {
		stream.playlist = p.(*m3u8.MasterPlaylist)
	}
}

func LoadPlaylist(playlist *m3uparser.M3UPlaylist) error {

	streamsStoreMux.Lock()
	defer streamsStoreMux.Unlock()

	for i, entry := range playlist.Entries {

		if entry.URI == "" {
			continue
		}

		parsedURL, err := url.Parse(entry.URI)
		if err != nil {
			log.Printf("Failed to parse URL: %s\n", entry.URI)
			continue
		}

		basePath := ""

		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			basePath += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		internalTags := entry.GetTag("M3UPROXYHEADER")
		headers := make(map[string]string)
		for _, tag := range internalTags {
			parts := strings.Split(tag.Value, "=")
			if len(parts) == 2 {
				headers[parts[0]] = parts[1]
			}
		}

		stream := Stream{
			index:    i,
			m3uEntry: entry,
			basePath: basePath,
			active:   false,
			playlist: nil,
			headers:  headers,
			mux:      &sync.Mutex{},
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
	streams[index].mux.Lock()
	streams[index].active = active
	streams[index].mux.Unlock()
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

	for i := 0; i < len(streams); i++ {
		checkStreamOnline(&streams[i], defaultTimeout)
		if !streams[i].active {
			log.Printf("Stream %d is offline\n", i)
		}
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

	stream := streams[streamID]
	stream.mux.Lock()
	defer stream.mux.Unlock()

	if path == "playlist.m3u8" && stream.playlist != nil {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stream.playlist.String()))
		return nil
	}

	serviceURL, _ := url.Parse(stream.m3uEntry.URI)

	if path != "playlist.m3u8" {
		if strings.HasPrefix(serviceURL.Path, stream.basePath) {
			serviceURL.Path = stream.basePath + "/" + path
			serviceURL.RawQuery = r.URL.RawQuery
		} else {
			serviceURL.Path = path
		}
	}

	req, _ := http.NewRequest(r.Method, serviceURL.String(), r.Body)

	for key := range stream.headers {
		r.Header.Add(key, stream.headers[key])
	}

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
	if code == 2 {

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
