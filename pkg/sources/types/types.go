package types

import (
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/upstream"
	"github.com/elnormous/contenttype"
)

type HttpDiags struct {
	Url       string `json:"url,omitempty"`
	Body      string `json:"body,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Error     string `json:"error,omitempty"`
	Status    int    `json:"status,omitempty"`
}
type StreamSourceDiag struct {
	Entry       m3uparser.M3UEntry `json:"entry,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	HttpProxy   string             `json:"http_proxy,omitempty"`
	Active      bool               `json:"active,omitempty"`
	Diagnostics []HttpDiags        `json:"diagnostics,omitempty"`
}

type StreamSource interface {
	ServeManifest(w http.ResponseWriter, r *http.Request, timeout int)
	ServeMedia(w http.ResponseWriter, r *http.Request, timeout int)
	HealthCheck() error
	Diagnostic() StreamSourceDiag
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
	mux              *sync.RWMutex
}
