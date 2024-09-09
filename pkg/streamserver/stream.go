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
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/grafov/m3u8"

	"github.com/gorilla/mux"
)

type Stream struct {
	index            int
	m3u              m3uparser.M3UEntry
	prefix           string
	active           bool
	hlsPlaylist      *m3u8.MasterPlaylist
	mux              *sync.Mutex
	headers          map[string]string
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
}

type StreamRequestOptions struct {
	Path   string
	Query  string
	Method string
}

func (stream *Stream) HttpRequest(opts StreamRequestOptions) (*http.Request, error) {

	streamURL, _ := url.Parse(stream.m3u.URI)

	if opts.Path != "playlist.m3u8" {
		if stream.prefix != "" {
			streamURL.Path = stream.prefix + "/" + opts.Path
		} else {
			streamURL.Path = opts.Path
		}
		if opts.Query != "" {
			streamURL.RawQuery = opts.Query
		}
	}

	req, err := http.NewRequest(opts.Method, streamURL.String(), nil)
	if err != nil {
		return nil, err
	}

	for key, value := range stream.headers {
		req.Header.Add(key, value)
	}

	return req, nil
}

func (stream *Stream) HealthCheck(timeout int) {

	stream.mux.Lock()
	defer stream.mux.Unlock()

	transport := http.DefaultTransport.(*http.Transport)
	if stream.httpProxy != "" {
		proxyURL, err := url.Parse(stream.httpProxy)
		if err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}

	stream.active = false
	stream.hlsPlaylist = nil

	req, err := stream.HttpRequest(StreamRequestOptions{
		Path:   "playlist.m3u8",
		Method: "GET",
	})

	if err != nil {
		return
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

	// Check if the URL has changed (redirects) and update the stream URL
	if resp.Request.URL.String() != stream.m3u.URI {

		stream.m3u.URI = resp.Request.URL.String()
		if strings.LastIndex(resp.Request.URL.Path, "/") != -1 {
			stream.prefix = resp.Request.URL.Path[:strings.LastIndex(resp.Request.URL.Path, "/")]
		} else {
			stream.prefix = ""
		}
	}

	contentType := resp.Header.Get("Content-Type")
	parts := strings.Split(contentType, ";")
	if len(parts) > 1 {
		contentType = strings.TrimRight(parts[0], " ")
	}
	switch strings.ToLower(contentType) {
	case "application/vnd.apple.mpegurl":
		// Check if the stream is a valid m3u8 playlist
		p, listType, err := m3u8.DecodeWith(bufio.NewReader(resp.Body), true, []m3u8.CustomDecoder{})
		if err != nil {
			return
		}
		if listType == m3u8.MASTER {
			stream.hlsPlaylist = p.(*m3u8.MasterPlaylist)
		}
	case "application/x-mpegurl":
		fallthrough
	case "audio/x-mpegurl":
		fallthrough
	case "audio/mpeg":
		fallthrough
	case "audio/aacp":
		fallthrough
	case "audio/aac":
		fallthrough
	case "binary/octet-stream":
	default:
		return
	}

	stream.active = true
}

func (stream *Stream) Serve(w http.ResponseWriter, r *http.Request, timeout int) error {

	vars := mux.Vars(r)
	path := vars["path"]

	stream.mux.Lock()

	if !stream.active {
		stream.mux.Unlock()
		return errors.New("stream not active")
	}

	transport := http.DefaultTransport.(*http.Transport)
	if stream.httpProxy != "" {
		proxyURL, err := url.Parse(stream.httpProxy)
		if err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}

	opts := StreamRequestOptions{}

	if path == "playlist.m3u8" {
		if stream.hlsPlaylist != nil {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(stream.hlsPlaylist.String()))
			stream.mux.Unlock()
			return nil
		}
		opts.Path = "playlist.m3u8"
		opts.Method = "GET"
	} else {
		opts.Path = path
		opts.Method = "GET"
		opts.Query = r.URL.RawQuery
	}

	req, err := stream.HttpRequest(opts)
	if err != nil {
		stream.mux.Unlock()
		return errors.New("failed to fetch stream")
	}

	stream.mux.Unlock()

	resp, err := client.Do(req)

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

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil

}
