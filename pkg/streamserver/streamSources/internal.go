package streamSources

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
	"github.com/valyala/fasthttp"
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

func contentTypeAllowed(resp *fasthttp.Response) (contenttype.MediaType, bool) {
	ct := contenttype.NewMediaType(string(resp.Header.ContentType()))
	return ct, ct.MatchesAny(supportedMediaTypes...)
}

func executeRequestWithRedirectTracking(method, URI string, client *fasthttp.Client, headers map[string]string) (*fasthttp.Response, string, error) {
	const maxRedirects = 10
	currentURL := URI
	var resp *fasthttp.Response

	for i := 0; i < maxRedirects; i++ {
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetRequestURI(currentURL)
		req.Header.SetMethod(method)

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp = fasthttp.AcquireResponse()
		err := client.Do(req, resp)
		if err != nil {
			fasthttp.ReleaseResponse(resp)
			return nil, "", err
		}

		statusCode := resp.StatusCode()
		if statusCode/100 == 3 { // Handle redirects (3xx status codes)
			location := resp.Header.Peek("Location")
			if location == nil {
				fasthttp.ReleaseResponse(resp)
				return nil, "", fmt.Errorf("redirect response missing Location header")
			}

			// Resolve the new URL relative to the current URL
			newURL := string(location)
			if !strings.HasPrefix(newURL, "http") {
				baseURL, err := url.Parse(currentURL)
				if err != nil {
					fasthttp.ReleaseResponse(resp)
					return nil, "", fmt.Errorf("failed to parse base URL: %w", err)
				}
				relativeURL, err := url.Parse(newURL)
				if err != nil {
					fasthttp.ReleaseResponse(resp)
					return nil, "", fmt.Errorf("failed to parse relative URL: %w", err)
				}
				currentURL = baseURL.ResolveReference(relativeURL).String()
			} else {
				currentURL = newURL
			}

			// Release the response and continue to the next redirect
			fasthttp.ReleaseResponse(resp)
			continue
		}

		// If not a redirect, return the response
		return resp, currentURL, nil
	}

	// Exceeded maximum redirects
	if resp != nil {
		fasthttp.ReleaseResponse(resp)
	}
	return nil, "", fmt.Errorf("too many redirects")
}

func executeRequest(method, URI string, client *fasthttp.Client, headers map[string]string) (*fasthttp.Response, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(URI)
	req.Header.SetMethod(method)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp := fasthttp.AcquireResponse()
	err := client.Do(req, resp)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, err
	}

	statusCode := resp.StatusCode()
	if statusCode/100 != 2 {
		fasthttp.ReleaseResponse(resp)
		return nil, fmt.Errorf("http response code (%d)", statusCode)
	}

	if statusCode == fasthttp.StatusNoContent {
		fasthttp.ReleaseResponse(resp)
		return nil, errors.New("no content")
	}

	return resp, nil
}

func verifyStream(mediaURI string, client *fasthttp.Client, headers map[string]string) (contenttype.MediaType, error) {
	resp, err := executeRequest("GET", mediaURI, client, headers)
	if err != nil {
		return contenttype.MediaType{}, err
	}
	defer fasthttp.ReleaseResponse(resp)

	ct, valid := contentTypeAllowed(resp)
	if !valid {
		return contenttype.MediaType{}, errors.New("invalid content type")
	}

	if ct.Subtype == "vnd.apple.mpegurl" || ct.Subtype == "x-mpegurl" {
		m3uPlaylist, err := m3uparser.DecodeFromReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return contenttype.MediaType{}, err
		}

		if len(m3uPlaylist.Entries) == 0 {
			return contenttype.MediaType{}, errors.New("empty playlist")
		}

		uri, _ := url.Parse(m3uPlaylist.Entries[0].URI)

		if uri.Scheme == "" {
			uri.Scheme = "http" // Default to HTTP
			uri.Host = string(resp.Header.Peek("Host"))
			basePath := path.Dir(mediaURI)
			uri.Path = path.Join(basePath, uri.Path)
		}

		return verifyStream(uri.String(), client, headers)
	}

	return ct, nil
}
