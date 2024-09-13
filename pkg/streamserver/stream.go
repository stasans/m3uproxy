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
	"log"
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
	masterPlaylist   *m3u8.MasterPlaylist
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

	path, _ := url.Parse(opts.Path)
	streamURL := &url.URL{}
	if path.Scheme == "" {

		streamURL, _ = url.Parse(stream.m3u.URI)

		if opts.Path != m3uPlaylist {
			if stream.prefix != "" {
				streamURL.Path = stream.prefix + "/" + opts.Path
			} else {
				streamURL.Path = opts.Path
			}
			if opts.Query != "" {
				streamURL.RawQuery = opts.Query
			}
		}
	} else {
		streamURL = path
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

	stream.active = false
	stream.masterPlaylist = nil
	resp, err := stream.Get(m3uPlaylist, timeout)
	if err != nil {
		return
	}
	stream.active = stream.validateM3U8Stream(resp)
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

	opts := StreamRequestOptions{
		Path:   m3uPlaylist,
		Method: "GET",
	}

	if path == m3uPlaylist {
		if stream.masterPlaylist != nil {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(stream.masterPlaylist.String()))
			stream.mux.Unlock()
			return nil
		}
	} else {
		opts.Path = path
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
		return err
	}

	defer resp.Body.Close()

	code := resp.StatusCode / 100
	if code != 2 {
		return errors.New("invalid server status code")
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

func (stream *Stream) Get(path string, timeout int) (*http.Response, error) {

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

	req, err := stream.HttpRequest(StreamRequestOptions{
		Path:   path,
		Method: "GET",
	})

	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid server status code")
	}

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
	contentType = strings.ToLower(contentType)
	switch contentType {
	case "application/vnd.apple.mpegurl":
		fallthrough
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
	case "audio/mp4":
		fallthrough
	case "audio/x-aac":
		fallthrough
	case "video/mp2t":
		fallthrough
	case "binary/octet-stream":
		return resp, nil
	default:
		return nil, errors.New("invalid content type")
	}
}

func (stream *Stream) validateM3U8Stream(resp *http.Response) bool {

	p, listType, err := m3u8.DecodeWith(bufio.NewReader(resp.Body), true, []m3u8.CustomDecoder{})
	if err != nil {
		log.Printf("Failed to decode m3u8 playlist: %v\n", err)
		return false
	}
	resp.Body.Close()

	switch listType {
	case m3u8.MASTER:

		playlist := p.(*m3u8.MasterPlaylist)
		if playlist == nil {
			return false
		}

		if len(playlist.Variants) == 0 {
			return false
		}

		variant := playlist.Variants[0]
		if variant == nil {
			return false
		}

		if _, err := stream.Get(variant.URI, serverConfig.DefaultTimeout); err != nil {
			return false
		}
		if len(playlist.Variants) == 0 {
			return true
		}

		resp, err := stream.Get(playlist.Variants[0].URI, serverConfig.DefaultTimeout)
		if err != nil {
			return false
		}

		stream.masterPlaylist = playlist

		return stream.validateM3U8Stream(resp)

	case m3u8.MEDIA:
		mediaPlaylist := p.(*m3u8.MediaPlaylist)
		segment := mediaPlaylist.Segments[0]
		if segment == nil {
			return false
		}
		_, err := stream.Get(segment.URI, serverConfig.DefaultTimeout)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return err == nil
	default:
		log.Printf("Unknown m3u8 playlist type: %v\n", listType)
		return false
	}

}
