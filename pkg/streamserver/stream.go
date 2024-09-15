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
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"

	"github.com/gorilla/mux"
)

type Stream struct {
	index            int
	m3u              m3uparser.M3UEntry
	prefix           string
	active           bool
	mux              *sync.Mutex
	headers          map[string]string
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
	transport        *http.Transport
	disableRemap     bool
}

type StreamRequestOptions struct {
	Path   string
	Query  string
	Method string
}

var supportedMediaTypes = []contenttype.MediaType{
	contenttype.NewMediaType("application/vnd.apple.mpegurl"),
	contenttype.NewMediaType("application/x-mpegurl"),
	contenttype.NewMediaType("audio/x-mpegurl"),
	contenttype.NewMediaType("audio/mpeg"),
	contenttype.NewMediaType("audio/aacp"),
	contenttype.NewMediaType("audio/aac"),
	contenttype.NewMediaType("audio/mp4"),
	contenttype.NewMediaType("audio/x-aac"),
	contenttype.NewMediaType("video/mp2t"),
	contenttype.NewMediaType("binary/octet-stream"),
}

func (stream *Stream) HealthCheck() {
	resp, err := stream.Get(stream.m3u.URI)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	stream.mux.Lock()
	if resp.Request.URL.String() != stream.m3u.URI {
		stream.m3u.URI = resp.Request.URL.String()
		if strings.LastIndex(resp.Request.URL.Path, "/") != -1 {
			stream.prefix = resp.Request.URL.Path[:strings.LastIndex(resp.Request.URL.Path, "/")]
		} else {
			stream.prefix = ""
		}
	}
	stream.mux.Unlock()

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
	streamActive := err == nil

	if streamActive && (mediaType.Subtype == "vnd.apple.mpegurl" || mediaType.Subtype == "x-mpegurl") {
		streamActive = validateM3UStream(stream, resp)
	}

	stream.mux.Lock()
	stream.active = streamActive
	stream.mux.Unlock()
}

func (stream *Stream) Serve(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	streamURL, _ := url.Parse(stream.m3u.URI)
	uri := url.URL{
		Scheme: streamURL.Scheme,
		Host:   streamURL.Host,
	}

	switch vars["path"] {
	case "master.m3u8":

		cache := r.URL.Query().Get("cache")
		if cache != "" {
			originalUrlBytes, err := base64.URLEncoding.DecodeString(cache)

			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			org, err := url.Parse(string(originalUrlBytes))
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			uri.RawQuery = org.RawQuery

			if org.Scheme != "" {
				uri.Scheme = org.Scheme
				uri.Host = org.Host
				uri.Path = org.Path
			} else {
				if stream.prefix != "" && !strings.HasPrefix(org.Path, "/") {
					uri.Path = stream.prefix + "/" + org.Path
				} else {
					uri.Path = org.Path
				}
			}

		} else {
			uri.RawQuery = streamURL.RawQuery
			uri.Path = streamURL.Path
		}
	case "media.ts":
		cache := r.URL.Query().Get("cache")
		if cache == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		originalUrlBytes, err := base64.URLEncoding.DecodeString(cache)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		org, err := url.Parse(string(originalUrlBytes))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		uri.RawQuery = org.RawQuery

		if org.Scheme != "" {
			uri.Scheme = org.Scheme
			uri.Host = org.Host
			uri.Path = org.Path
		} else {
			if stream.prefix != "" && !strings.HasPrefix(org.Path, "/") {
				uri.Path = stream.prefix + "/" + org.Path
			} else {
				uri.Path = org.Path
			}
		}

	default:

		uri.RawQuery = r.URL.RawQuery

		if stream.prefix != "" {
			uri.Path = stream.prefix + "/" + vars["path"]
		} else {
			uri.Path = vars["path"]
		}
	}

	path := uri.String()
	resp, err := stream.Get(path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
	if err != nil {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	if mediaType.Subtype == "vnd.apple.mpegurl" || mediaType.Subtype == "x-mpegurl" {
		remapAndServe(resp, w)
		return
	}

	w.Header().Set("Content-Type", ct)
	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}
	io.Copy(w, resp.Body)
}

func (stream *Stream) Do(method, URI string) (*http.Response, error) {

	client := &http.Client{
		Timeout:   time.Duration(serverConfig.Timeout) * time.Second,
		Transport: stream.transport,
	}

	req, err := http.NewRequest(method, URI, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range stream.headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	code := resp.StatusCode / 100
	if code != 2 {
		return nil, errors.New("invalid server status code")
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, errors.New("no content")
	}

	return resp, nil
}

func (stream *Stream) Get(path string) (*http.Response, error) {
	resp, err := stream.Do("GET", path)
	return resp, err
}

func validateM3UStream(stream *Stream, resp *http.Response) bool {

	m3uPlaylist, err := m3uparser.DecodeFromReader(resp.Body)
	if err != nil {
		return false
	}

	if len(m3uPlaylist.Entries) == 0 {
		return false
	}

	streamURL, _ := url.Parse(stream.m3u.URI)
	uri := url.URL{
		Scheme: streamURL.Scheme,
		Host:   streamURL.Host,
	}

	r, _ := url.Parse(m3uPlaylist.Entries[0].URI)

	if r.Scheme != "" {
		uri.Scheme = r.Scheme
		uri.Host = r.Host
		uri.Path = r.Path
		uri.RawQuery = r.RawQuery
	} else {
		uri.RawQuery = r.RawQuery

		if stream.prefix != "" && !strings.HasPrefix(r.Path, "/") {
			uri.Path = stream.prefix + "/" + r.Path
		} else {
			uri.Path = r.Path
		}
	}

	streamResponse, err := stream.Get(uri.String())
	if err != nil {
		return false
	}
	defer streamResponse.Body.Close()
	ct := streamResponse.Header.Get("Content-Type")
	mediaType, _, err := contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
	if err != nil {
		return false
	}

	if mediaType.Subtype == "vnd.apple.mpegurl" || mediaType.Subtype == "x-mpegurl" {
		return validateM3UStream(stream, streamResponse)
	}

	return true
}

func remapAndServe(resp *http.Response, w http.ResponseWriter) {

	m3uPlaylist, err := m3uparser.DecodeFromReader(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp.Body.Close()

	var filePrefix string

	switch m3uPlaylist.Type {
	case "master":
		filePrefix = "master.m3u8"
	case "media":
		filePrefix = "media.ts"
	default:
		log.Printf("Unknown m3u8 playlist type: %v\n", m3uPlaylist.Type)
		return
	}

	for i, _ := range m3uPlaylist.Entries {
		remap := fmt.Sprintf("%s?cache=%s", filePrefix, base64.URLEncoding.EncodeToString([]byte(m3uPlaylist.Entries[i].URI)))
		m3uPlaylist.Entries[i].URI = remap
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	m3uPlaylist.WriteTo(w)
}
