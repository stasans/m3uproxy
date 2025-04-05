package streamSources

import (
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/upstream"
	"github.com/elnormous/contenttype"
)

type StreamSource interface {
	ServeManifest(w http.ResponseWriter, r *http.Request, timeout int)
	ServeMedia(w http.ResponseWriter, r *http.Request, timeout int)
	HealthCheck() error
	Active() bool
	MediaType() contenttype.MediaType
	MediaName() string
	MasterPlaylist() string
	M3UTags() m3uparser.M3UTags
	IsRadio() bool
	Url() string
}

type BaseStreamSource struct {
	StreamSource
	mediaType        contenttype.MediaType
	m3u              m3uparser.M3UEntry
	headers          map[string]string // Changed from http.Header to map[string]string for fasthttp compatibility
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
	conn             *upstream.UpstreamConnection
	disableRemap     bool
	active           bool
	mux              *sync.Mutex
}

type M3UStreamSource struct {
	BaseStreamSource
}

type MPDStreamSource struct {
	BaseStreamSource
}

type Sources struct {
	sources      []StreamSource
	mux          *sync.Mutex
	activeSource StreamSource
}

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
