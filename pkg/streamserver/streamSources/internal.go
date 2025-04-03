package streamSources

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

var supportedMediaTypes = []contenttype.MediaType{
	contenttype.NewMediaType("application/vnd.apple.mpegurl"),
	contenttype.NewMediaType("application/x-mpegurl"),
	contenttype.NewMediaType("audio/x-mpegurl"),
	contenttype.NewMediaType("audio/mpeg"),
	contenttype.NewMediaType("audio/aacp"),
	contenttype.NewMediaType("audio/aac"),
	contenttype.NewMediaType("audio/mp4"),
	contenttype.NewMediaType("audio/mp3"),
	contenttype.NewMediaType("audio/ac3"),
	contenttype.NewMediaType("audio/x-aac"),
	contenttype.NewMediaType("video/mp2t"),
	contenttype.NewMediaType("video/m2ts"),
	contenttype.NewMediaType("video/mp4"),
	contenttype.NewMediaType("binary/octet-stream"),
	contenttype.NewMediaType("application/octet-stream"),
	contenttype.NewMediaType("application/dash+xml"),
}

func contentTypeAllowed(resp *http.Response) (contenttype.MediaType, bool) {
	ct := contenttype.NewMediaType(resp.Header.Get("Content-Type"))
	return ct, ct.MatchesAny(supportedMediaTypes...)
}

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

	if resp.StatusCode == http.StatusNoContent {
		return nil, errors.New("no content")
	}

	return resp, nil
}

func verifyStream(mediaURI string, transport *http.Transport, headers map[string]string, timeout int) (contenttype.MediaType, error) {

	resp, err := executeRequest("GET", mediaURI, transport, headers, timeout)
	if err != nil {
		return contenttype.MediaType{}, err
	}

	ct, valid := contentTypeAllowed(resp)
	if !valid {
		return contenttype.MediaType{}, errors.New("invalid content type")
	}

	if ct.Subtype == "vnd.apple.mpegurl" || ct.Subtype == "x-mpegurl" {
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

	return ct, nil
}
