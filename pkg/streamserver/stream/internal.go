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
package stream

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

func executeRequest(method, URI string, transport *http.Transport, headers map[string]string, timeout int) (*http.Response, error) {

	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
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

func verifyStream(mediaURI string, transport *http.Transport, headers map[string]string, timeout int) (contenttype.MediaType, error) {

	resp, err := executeRequest("GET", mediaURI, transport, headers, timeout)
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

		return verifyStream(uri.String(), transport, headers, timeout)
	}

	return mediaType, nil
}
