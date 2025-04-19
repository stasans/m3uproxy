package types

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

func (s *BaseStreamSource) HealthCheck() error {
	uri, _, err := s.conn.Check("GET", s.m3u.URI)
	if err != nil {
		return err
	}

	s.mux.Lock()
	if uri != s.m3u.URI {
		s.m3u.URI = uri
	}
	s.mux.Unlock()

	_, err = s.verify(s.m3u.URI)

	s.mux.Lock()
	s.active = err == nil
	s.mux.Unlock()

	return err
}

func (s *BaseStreamSource) Diagnostic() StreamSourceDiag {
	uri, _, err := s.conn.Check("GET", s.m3u.URI)
	diag := StreamSourceDiag{
		Entry:       s.m3u,
		Headers:     s.headers,
		HttpProxy:   s.httpProxy,
		Active:      false,
		Diagnostics: make([]HttpDiags, 0),
	}
	if err != nil {
		diag.Diagnostics = append(diag.Diagnostics, HttpDiags{
			Url:    uri,
			Status: http.StatusBadRequest,
			Error:  err.Error(),
		})
		return diag
	}

	s.mux.Lock()
	if uri != s.m3u.URI {
		s.m3u.URI = uri
	}
	s.mux.Unlock()

	s.verifyWithDiags(s.m3u.URI, &diag)
	return diag
}

func (s *BaseStreamSource) Active() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.active
}

func (s *BaseStreamSource) MediaType() contenttype.MediaType {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.mediaType
}

func (s *BaseStreamSource) MediaName() string {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.m3u.Title
}

func (s *BaseStreamSource) M3UTags() m3uparser.M3UTags {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.m3u.Tags
}

func (s *BaseStreamSource) IsRadio() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.radio
}

func (s *BaseStreamSource) Url() string {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.m3u.URI
}

func (s *BaseStreamSource) verify(mediaURI string) (contenttype.MediaType, error) {
	s.mux.RLock()
	body, _, ct, err := s.conn.Get("GET", mediaURI)
	s.mux.RUnlock()
	if err != nil {
		return contenttype.MediaType{}, err
	}

	if !ct.MatchesAny(supportedMediaTypes...) {
		return contenttype.MediaType{}, errors.New("invalid content type")
	}

	if ct.Subtype == "vnd.apple.mpegurl" || ct.Subtype == "x-mpegurl" {
		m3uPlaylist, err := m3uparser.DecodeFromReader(bytes.NewReader(body))
		if err != nil {
			return contenttype.MediaType{}, err
		}

		if len(m3uPlaylist.Entries) == 0 {
			return contenttype.MediaType{}, errors.New("empty playlist")
		}

		uri, _ := url.Parse(m3uPlaylist.Entries[0].URI)

		if uri.Scheme == "" {
			originalURI, err := url.Parse(mediaURI)
			if err != nil {
				return contenttype.MediaType{}, err
			}
			uri.Scheme = originalURI.Scheme
			uri.Host = originalURI.Host
			if !strings.HasPrefix(uri.Path, "/") {
				uri.Path = path.Join(path.Dir(originalURI.Path), uri.Path)
			}
		}

		return s.verify(uri.String())
	}

	return ct, nil
}

func (s *BaseStreamSource) verifyWithDiags(mediaURI string, diag *StreamSourceDiag) {
	s.mux.RLock()
	body, status, ct, err := s.conn.Get("GET", mediaURI)
	s.mux.RUnlock()
	if err != nil {
		diag.Diagnostics = append(diag.Diagnostics, HttpDiags{
			Url:    mediaURI,
			Status: status,
			Error:  err.Error(),
		})
		return
	}
	httpDiag := HttpDiags{
		Url:       mediaURI,
		Body:      string(body),
		MediaType: ct.String(),
		Status:    status,
	}

	if !ct.MatchesAny(supportedMediaTypes...) {
		httpDiag.Error = "invalid content type"
		diag.Diagnostics = append(diag.Diagnostics, httpDiag)
		return
	}

	if ct.Subtype == "vnd.apple.mpegurl" || ct.Subtype == "x-mpegurl" {
		m3uPlaylist, err := m3uparser.DecodeFromReader(bytes.NewReader(body))
		if err != nil {
			httpDiag.Error = err.Error()
			diag.Diagnostics = append(diag.Diagnostics, httpDiag)
			return
		}

		if len(m3uPlaylist.Entries) == 0 {
			httpDiag.Error = "empty playlist"
			diag.Diagnostics = append(diag.Diagnostics, httpDiag)
			return
		}

		uri, _ := url.Parse(m3uPlaylist.Entries[0].URI)

		if uri.Scheme == "" {
			originalURI, err := url.Parse(mediaURI)
			if err != nil {
				httpDiag.Error = err.Error()
				diag.Diagnostics = append(diag.Diagnostics, httpDiag)
				return
			}
			uri.Scheme = originalURI.Scheme
			uri.Host = originalURI.Host
			if !strings.HasPrefix(uri.Path, "/") {
				uri.Path = path.Join(path.Dir(originalURI.Path), uri.Path)
			}
		}

		s.verifyWithDiags(uri.String(), diag)
	}

	diag.Diagnostics = append(diag.Diagnostics, httpDiag)
	diag.Active = status == http.StatusOK
}
