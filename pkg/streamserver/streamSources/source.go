package streamSources

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

func sourceFactory(entry m3uparser.M3UEntry, timeout int) StreamSource {

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
				log.Printf("Unknown M3UPROXYTRANSPORT tag: %s\n", parts[0])
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

	if headers["User-Agent"] == "" {
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

	resp, err := executeRequest("GET", entry.URI, transport, headers, timeout)
	if err != nil {
		return nil
	}

	resp.Body.Close()

	if resp.Request.URL.String() != entry.URI {
		entry.URI = resp.Request.URL.String()
	}

	ct, valid := contentTypeAllowed(resp)
	if !valid {
		log.Printf("Invalid content type: %s\n", ct)
		return nil
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
				transport:        transport,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}
	default:
		return &M3UStreamSource{
			BaseStreamSource: BaseStreamSource{
				m3u:              entry,
				headers:          headers,
				httpProxy:        proxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio != "",
				transport:        transport,
				disableRemap:     disableRemap,
				mux:              &sync.Mutex{},
			},
		}
	}
}

func (stream *BaseStreamSource) HealthCheck(timeout int) error {
	resp, err := executeRequest("GET", stream.m3u.URI, stream.transport, stream.headers, timeout)
	if err != nil {
		return err
	}
	resp.Body.Close()

	stream.mux.Lock()
	if resp.Request.URL.String() != stream.m3u.URI {
		stream.m3u.URI = resp.Request.URL.String()
	}
	stream.mux.Unlock()

	_, err = verifyStream(stream.m3u.URI, stream.transport, stream.headers, timeout)

	stream.mux.Lock()
	stream.active = err == nil
	stream.mux.Unlock()

	if err != nil {
		log.Printf("Stream %s is not healthy: %s\n", stream.m3u.Title, err)
	}
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

func (s *Sources) AddSource(entry m3uparser.M3UEntry, timeout int) {

	// Check if the source is already in the list
	if s.SourceExists(entry) {
		log.Printf("Stream source already exists: %s\n", entry.URI)
		return
	}

	source := sourceFactory(entry, timeout)
	if source == nil {
		log.Printf("Error registering stream source: %s\n", entry.URI)
		return
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.sources = append(s.sources, source)
}

func (s *Sources) GetActiveSource() StreamSource {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.activeSource
}

func (s *Sources) HealthCheck(timeout int) error {

	var activeSource StreamSource = nil
	for _, source := range s.sources {
		err := source.HealthCheck(timeout)
		if err != nil {
			log.Printf("Stream %s is not healthy: %s\n", source.MediaName(), err)
		}
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

func (s *Sources) Serve(w http.ResponseWriter, r *http.Request, timeout int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.activeSource == nil {
		http.Error(w, "No active stream source", http.StatusServiceUnavailable)
		return
	}

	s.activeSource.Serve(w, r, timeout)
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
