package streamSources

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"
	"github.com/gorilla/mux"
)

func (stream *M3UStreamSource) Serve(w http.ResponseWriter, r *http.Request, timeout int) {

	vars := mux.Vars(r)

	var uri *url.URL
	switch vars["path"] {
	case "master.m3u8":
		cache := r.URL.Query().Get("cache")

		if cache == "" {
			// if the cache is empty, we must be serving the master playlist
			uri, _ = url.Parse(stream.m3u.URI)
		} else {
			originalUrlBytes, err := base64.URLEncoding.DecodeString(cache)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			orig, err := url.Parse(string(originalUrlBytes))
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			uri = orig
		}

		mediaURI := uri.String()

		resp, err := executeRequest("GET", mediaURI, stream.transport, stream.headers, timeout)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		ct := resp.Header.Get("Content-Type")
		_, _, err = contenttype.GetAcceptableMediaTypeFromHeader(ct, supportedMediaTypes)
		if err != nil {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		defer resp.Body.Close()
		m3uPlaylist, err := m3uparser.DecodeFromReader(resp.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var filePrefix string
		switch m3uPlaylist.Type {
		case "master":
			filePrefix = "master.m3u8"
		case "media":
			filePrefix = "media.ts"
		default:
			log.Printf("Unknown m3u8 playlist type: %v\n", m3uPlaylist.Type)
			return
		}

		for i := range m3uPlaylist.Entries {
			uri, _ := url.Parse(m3uPlaylist.Entries[i].URI)

			if uri.Scheme == "" {
				uri.Scheme = resp.Request.URL.Scheme
				uri.Host = resp.Request.URL.Host
				basePath := path.Dir(resp.Request.URL.Path)
				uri.Path = path.Join(basePath, uri.Path)
			}

			remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
			m3uPlaylist.Entries[i].URI = fmt.Sprintf("%s?cache=%s", filePrefix, remap)
		}

		m3uPlaylist.WriteTo(w)

	case "media.ts":
		cache := r.URL.Query().Get("cache")

		if cache == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		originalUrlBytes, err := base64.URLEncoding.DecodeString(cache)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		orig, err := url.Parse(string(originalUrlBytes))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		uri = orig
		mediaURI := uri.String()

		resp, err := executeRequest("GET", mediaURI, stream.transport, stream.headers, timeout)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		ct := resp.Header.Get("Content-Type")

		defer resp.Body.Close()
		w.Header().Set("Content-Type", ct)
		io.Copy(w, resp.Body)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

}

func (stream *M3UStreamSource) MasterPlaylist() string {
	return "master.m3u8"
}
