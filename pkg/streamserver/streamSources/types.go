package streamSources

import (
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

type StreamSource interface {
	Serve(w http.ResponseWriter, r *http.Request, timeout int)
	HealthCheck(timeout int) error
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
	headers          map[string]string
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
	transport        *http.Transport
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
