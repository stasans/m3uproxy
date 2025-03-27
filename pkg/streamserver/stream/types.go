package stream

import (
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
)

type Stream interface {
	Serve(w http.ResponseWriter, r *http.Request, timeout int)
	HealthCheck(timeout int) error
	Active() bool
	MediaType() contenttype.MediaType
	MediaName() string
	MasterPlaylist() string
	M3UTags() m3uparser.M3UTags
	IsRadio() bool
}

type BaseStream struct {
	Stream
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

type M3UStream struct {
	BaseStream
}

type MPDStream struct {
	BaseStream
}
