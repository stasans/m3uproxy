package streamSources

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
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
				// For now, we only handle the proxy option
				// and ignore others.
			}
		}
	}

	transport := http.DefaultTransport.(*http.Transport)
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	headers := http.Header{}
	m3uproxyTags = entry.SearchTags("M3UPROXYHEADER")
	for _, tag := range m3uproxyTags {
		parts := strings.Split(tag.Value, "=")
		if len(parts) == 2 {
			headers.Set(parts[0], parts[1])
		}
	}

	vlcTags := entry.SearchTags("EXTVLCOPT")
	for _, tag := range vlcTags {
		parts := strings.Split(tag.Value, "=")
		if len(parts) == 2 {
			switch parts[0] {
			case "http-user-agent":
				headers.Set("User-Agent", parts[1])
			case "http-referrer":
				headers.Set("Referer", parts[1])
			default:
			}
		}
	}

	if headers.Get("User-Agent") == "" {
		// Default User-Agent for VLC
		// This is a workaround for some servers that require a User-Agent header
		// to be set. VLC uses "VLC/3.0.11 LibVLC/3.0.11" as the default User-Agent.
		// This can be changed to any other User-Agent string if needed.
		headers.Set("User-Agent", "VLC/3.0.11 LibVLC/3.0.11")
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
	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}

	resp, err := executeRequest("GET", entry.URI, client, headers)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	if resp.Request.URL.String() != entry.URI {
		entry.URI = resp.Request.URL.String()
	}

	ct, valid := contentTypeAllowed(resp)
	if !valid {
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
				client:           client,
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
				client:           client,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}, nil
	}
}

func (stream *BaseStreamSource) HealthCheck() error {
	resp, err := executeRequest("GET", stream.m3u.URI, stream.client, stream.headers)
	if err != nil {
		return err
	}
	resp.Body.Close()

	stream.mux.Lock()
	if resp.Request.URL.String() != stream.m3u.URI {
		stream.m3u.URI = resp.Request.URL.String()
	}
	stream.mux.Unlock()

	_, err = verifyStream(stream.m3u.URI, stream.client, stream.headers)

	stream.mux.Lock()
	stream.active = err == nil
	stream.mux.Unlock()

	return err
}

func (stream *BaseStreamSource) Active() bool {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.active
}

func (stream *BaseStreamSource) MediaType() contenttype.MediaType {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.mediaType
}

func (stream *BaseStreamSource) MediaName() string {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.m3u.Title
}

func (stream *BaseStreamSource) M3UTags() m3uparser.M3UTags {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.m3u.Tags
}

func (stream *BaseStreamSource) IsRadio() bool {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.radio
}

func (stream *BaseStreamSource) Url() string {
	stream.mux.Lock()
	defer stream.mux.Unlock()
	return stream.m3u.URI
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

	// Check if the source is already in the list
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
