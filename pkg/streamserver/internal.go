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
	"os"
	"path"
	"strings"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	mpd "github.com/a13labs/m3uproxy/pkg/mpdparser"
	"github.com/elnormous/contenttype"
)

func executeRequest(method, URI string, transport *http.Transport, headers map[string]string) (*http.Response, error) {

	client := &http.Client{
		Timeout:   time.Duration(Config.Timeout) * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest(method, URI, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
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

	return resp, nil
}

func verifyStream(mediaURI string, transport *http.Transport, headers map[string]string) (contenttype.MediaType, error) {

	resp, err := executeRequest("GET", mediaURI, transport, headers)
	if err != nil {
		return contenttype.MediaType{}, err
	}

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
	if err != nil {
		return contenttype.MediaType{}, err
	}

	if mediaType.Subtype == "vnd.apple.mpegurl" || mediaType.Subtype == "x-mpegurl" {
		m3uPlaylist, err := m3uparser.DecodeFromReader(resp.Body)
		if err != nil {
			return contenttype.MediaType{}, err
		}

		if len(m3uPlaylist.Entries) == 0 {
			return contenttype.MediaType{}, errors.New("empty playlist")
		}

		uri, _ := url.Parse(m3uPlaylist.Entries[0].URI)

		if uri.Scheme == "" {
			uri.Scheme = resp.Request.URL.Scheme
			uri.Host = resp.Request.URL.Host
			basePath := path.Dir(resp.Request.URL.Path)
			uri.Path = path.Join(basePath, uri.Path)
		}

		return verifyStream(uri.String(), transport, headers)
	}

	return mediaType, nil
}

// serveAndRemap serves the mediaURI and remaps the URLs in the playlist to point to the proxy cache url
// instead of the original URL using base64 encoding.
func serveAndRemap(mediaURI string, transport *http.Transport, headers map[string]string, w http.ResponseWriter) {

	resp, err := executeRequest("GET", mediaURI, transport, headers)
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

	w.Header().Set("Content-Type", ct)
	switch {
	case mediaType.Subtype == "vnd.apple.mpegurl" || mediaType.Subtype == "x-mpegurl":
		remapM38U(w, resp)
	case mediaType.Subtype == "dash+xml":
		remapMPD(w, resp)
	default:
		io.Copy(w, resp.Body)
	}
}

func remapM38U(w http.ResponseWriter, resp *http.Response) {
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

	for i := range m3uPlaylist.Entries {
		uri, _ := url.Parse(m3uPlaylist.Entries[i].URI)

		if uri.Scheme == "" {
			uri.Scheme = resp.Request.URL.Scheme
			uri.Host = resp.Request.URL.Host
			basePath := path.Dir(resp.Request.URL.Path)
			uri.Path = path.Join(basePath, uri.Path)
		}

		remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
		m3uPlaylist.Entries[i].URI = fmt.Sprintf("%s?cache=%s", filePrefix, remap)
	}

	m3uPlaylist.WriteTo(w)
}

func remapMPD(w http.ResponseWriter, resp *http.Response) {

	mpdPlaylist, err := mpd.DecodeFromReader(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for i := range mpdPlaylist.Period {
		for j := range mpdPlaylist.Period[i].AdaptationSets {
			for k := range mpdPlaylist.Period[i].AdaptationSets[j].Representations {

				if len(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL) == 0 {
					uri := new(url.URL)
					uri.Scheme = resp.Request.URL.Scheme
					uri.Host = resp.Request.URL.Host
					basePath := path.Dir(resp.Request.URL.Path)
					uri.Path = path.Join(basePath, uri.Path)
					remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
					mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL = append(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL, &mpd.BaseURL{Value: fmt.Sprintf("mpd-%s/", remap)})
					continue
				} else {
					for l := range mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL {
						currentBaseURL := mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value

						uri, _ := url.Parse(currentBaseURL)

						if uri.Scheme == "" {
							uri.Scheme = resp.Request.URL.Scheme
							uri.Host = resp.Request.URL.Host
							basePath := path.Dir(resp.Request.URL.Path)
							uri.Path = path.Join(basePath, uri.Path)
						}

						remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
						mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value = fmt.Sprintf("mpd-%s/", remap)
					}
				}
			}
		}
	}

	mpdPlaylist.WriteTo(w)
}

func loadContent(filePath string) (string, error) {
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		// Load content from URL
		resp, err := http.Get(filePath)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	} else {
		// Load content from local file
		file, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		body, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
}
