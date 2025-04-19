package types

import (
	"fmt"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/upstream"
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

func NewSource(entry m3uparser.M3UEntry, timeout int) (StreamSource, error) {
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

	conn := upstream.NewUpstreamConnection(headers, proxy, timeout)

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
				mux:              &sync.RWMutex{},
			},
		}, nil
	default:
		return &M3U8StreamSource{
			BaseStreamSource: BaseStreamSource{
				m3u:              entry,
				headers:          headers,
				httpProxy:        proxy,
				forceKodiHeaders: forceKodiHeaders,
				radio:            radio != "",
				conn:             conn,
				disableRemap:     disableRemap,
				mux:              &sync.RWMutex{},
			},
		}, nil
	}
}
