package upstream

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elnormous/contenttype"
	"github.com/valyala/fasthttp"
)

func NewUpstreamConnection(headers map[string]string, timeout int) *UpstreamConnection {
	return &UpstreamConnection{
		client: &fasthttp.Client{
			ReadTimeout: time.Duration(timeout) * time.Second,
		},
		headers: headers,
	}
}

func (u *UpstreamConnection) Check(method string, uri string) (string, contenttype.MediaType, error) {
	const maxRedirects = 10
	currentURL := uri
	var resp *fasthttp.Response

	for i := 0; i < maxRedirects; i++ {
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetRequestURI(currentURL)
		req.Header.SetMethod(method)

		for key, value := range u.headers {
			req.Header.Set(key, value)
		}

		resp = fasthttp.AcquireResponse()
		err := u.client.Do(req, resp)
		if err != nil {
			fasthttp.ReleaseResponse(resp)
			return "", contenttype.MediaType{}, err
		}

		statusCode := resp.StatusCode()
		ct := contenttype.NewMediaType(string(resp.Header.ContentType()))

		if statusCode/100 == 3 { // Handle redirects (3xx status codes)
			location := resp.Header.Peek("Location")
			if location == nil {
				fasthttp.ReleaseResponse(resp)
				return "", ct, fmt.Errorf("redirect response missing Location header")
			}

			// Resolve the new URL relative to the current URL
			newURL := string(location)
			if !strings.HasPrefix(newURL, "http") {
				baseURL, err := url.Parse(currentURL)
				if err != nil {
					fasthttp.ReleaseResponse(resp)
					return "", ct, fmt.Errorf("failed to parse base URL: %w", err)
				}
				relativeURL, err := url.Parse(newURL)
				if err != nil {
					fasthttp.ReleaseResponse(resp)
					return "", ct, fmt.Errorf("failed to parse relative URL: %w", err)
				}
				currentURL = baseURL.ResolveReference(relativeURL).String()
			} else {
				currentURL = newURL
			}

			// Release the response and continue to the next redirect
			fasthttp.ReleaseResponse(resp)
			continue
		}

		return currentURL, ct, nil
	}

	// Exceeded maximum redirects
	if resp != nil {
		fasthttp.ReleaseResponse(resp)
	}
	return "", contenttype.MediaType{}, fmt.Errorf("too many redirects")
}

func (u *UpstreamConnection) Get(method, URI string) ([]byte, int, contenttype.MediaType, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(URI)
	req.Header.SetMethod(method)

	for key, value := range u.headers {
		req.Header.Set(key, value)
	}

	resp := fasthttp.AcquireResponse()
	err := u.client.Do(req, resp)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, -1, contenttype.MediaType{}, err
	}

	statusCode := resp.StatusCode()
	ct := contenttype.NewMediaType(string(resp.Header.ContentType()))
	if statusCode/100 != 2 {
		fasthttp.ReleaseResponse(resp)
		return nil, statusCode, ct, fmt.Errorf("http response code (%d)", statusCode)
	}

	if statusCode == fasthttp.StatusNoContent {
		fasthttp.ReleaseResponse(resp)
		return nil, statusCode, ct, errors.New("no content")
	}

	dst := make([]byte, len(resp.Body()))
	copy(dst, resp.Body())
	return dst, statusCode, ct, nil
}
