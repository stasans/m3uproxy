package streamSources

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/upstream"
	"github.com/elnormous/contenttype"
)

func sourceFactory(entry m3uparser.M3UEntry, timeout int) (StreamSource, error) {
	radio := entry.ExtInfTags.GetValue("radio")

	proxy := ""
	m3uproxyTags := entry.SearchTags("M3UPROXYTRANSPORT")
	if len(m3uproxyTags) > 0 {
		parts := strings.Split(m3uproxyTags[0].Value, "=")
		if len(parts) == 2 {
			switch parts[0] {
			case "proxy":
				proxy = parts[1]
			default:
				// Handle other transport options if needed
			}
		}
	}

	headers := make(map[string]string)
	m3uproxyTags = entry.SearchTags("M3UPROXYHEADER")
	for _, tag := range m3uproxyTags {
		parts := strings.Split(tag.Value, "=")
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	vlcTags := entry.SearchTags("EXTVLCOPT")
	for _, tag := range vlcTags {
		parts := strings.Split(tag.Value, "=")
		if len(parts) == 2 {
			switch parts[0] {
			case "http-user-agent":
				headers["User-Agent"] = parts[1]
			case "http-referrer":
				headers["Referer"] = parts[1]
			default:
			}
		}
	}

	if _, ok := headers["User-Agent"]; !ok {
		headers["User-Agent"] = "VLC/3.0.11 LibVLC/3.0.11"
	}

	m3uproxyTags = entry.SearchTags("M3UPROXYOPT")
	forceKodiHeaders := false
	disableRemap := false
	for _, tag := range m3uproxyTags {
		switch tag.Value {
		case "forcekodiheaders":
			forceKodiHeaders = true
		case "disableremap":
			disableRemap = true
		default:
		}
	}

	// Clear non-standard tags
	entry.ClearTags()

	conn := upstream.NewUpstreamConnection(headers, timeout)

	uri, ct, err := conn.Check("GET", entry.URI)
	if err != nil {
		return nil, err
	}

	if uri != entry.URI {
		entry.URI = uri
	}

	if !ct.MatchesAny(supportedMediaTypes...) {
		return nil, fmt.Errorf("invalid content type: %s", ct)
	}

	switch {
	case ct.Subtype == "dash+xml":
		return &MPDStreamSource{
			BaseStreamSource: BaseStreamSource{
				m3u:              entry,
				headers:          headers,
				httpProxy:        proxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio != "",
				conn:             conn,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}, nil
	default:
		return &M3UStreamSource{
			BaseStreamSource: BaseStreamSource{
				m3u:              entry,
				headers:          headers,
				httpProxy:        proxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio != "",
				conn:             conn,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}, nil
	}
}

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

func (s *BaseStreamSource) Active() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.active
}

func (s *BaseStreamSource) MediaType() contenttype.MediaType {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.mediaType
}

func (s *BaseStreamSource) MediaName() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.m3u.Title
}

func (s *BaseStreamSource) M3UTags() m3uparser.M3UTags {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.m3u.Tags
}

func (s *BaseStreamSource) IsRadio() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.radio
}

func (s *BaseStreamSource) Url() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.m3u.URI
}

func (s *BaseStreamSource) verify(mediaURI string) (contenttype.MediaType, error) {
	body, _, ct, err := s.conn.Get("GET", mediaURI)
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
			uri.Scheme = "http" // Default to HTTP
			uri.Host = string(resp.Header.Peek("Host"))
			basePath := path.Dir(mediaURI)
			uri.Path = path.Join(basePath, uri.Path)
		}

		return s.verify(uri.String())
	}

	return ct, nil
}

func CreateSources() Sources {
	return Sources{
		sources:      make([]StreamSource, 0),
		mux:          &sync.Mutex{},
		activeSource: nil,
	}
}

func (s *Sources) Sources() []StreamSource {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.sources
}

func (s *Sources) SourceExists(entry m3uparser.M3UEntry) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, source := range s.sources {
		if source.Url() == entry.URI {
			return true
		}
	}
	return false
}

func (s *Sources) AddSource(entry m3uparser.M3UEntry, timeout int) (bool, error) {
	if s.SourceExists(entry) {
		return false, fmt.Errorf("source already exists")
	}

	source, err := sourceFactory(entry, timeout)
	if err != nil {
		return false, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.sources = append(s.sources, source)
	return true, nil
}

func (s *Sources) GetActiveSource() StreamSource {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.activeSource
}

func (s *Sources) HealthCheck() error {
	var activeSource StreamSource = nil
	for _, source := range s.sources {
		_ = source.HealthCheck()
		if source.Active() {
			activeSource = source
			break
		}
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.activeSource = activeSource
	if activeSource == nil {
		return fmt.Errorf("no active stream source found")
	}
	return nil
}

func (s *Sources) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.activeSource == nil {
		http.Error(w, "No active stream source", http.StatusServiceUnavailable)
		return
	}

	s.activeSource.ServeManifest(w, r, timeout)
}

func (s *Sources) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.activeSource == nil {
		http.Error(w, "No active stream source", http.StatusServiceUnavailable)
		return
	}

	s.activeSource.ServeMedia(w, r, timeout)
}

func (s *Sources) Active() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.activeSource != nil
}

func (s *Sources) MediaName() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.MediaName()
	}
	return ""
}

func (s *Sources) MasterPlaylist() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.MasterPlaylist()
	}
	return ""
}

func (s *Sources) M3UTags() m3uparser.M3UTags {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.M3UTags()
	}
	return nil
}

func (s *Sources) IsRadio() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.IsRadio()
	}
	return false
}
