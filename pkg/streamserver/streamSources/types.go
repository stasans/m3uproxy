package streamSources

import (
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
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
	headers          http.Header
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
	client           *http.Client
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
