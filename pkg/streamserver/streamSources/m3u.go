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
	"github.com/gorilla/mux"
)

func readMP3Url(r *http.Request, stream *M3UStreamSource) (*url.URL, error) {
	orig := r.URL.Query().Get("o")
	if orig == "" {
		return url.Parse(stream.m3u.URI)
	}

	originalUrlBytes, err := base64.URLEncoding.DecodeString(orig)
	if err != nil {
		return nil, err
	}

	return url.Parse(string(originalUrlBytes))
}

func remapM3UPlaylist(resp *http.Response) (*m3uparser.M3UPlaylist, error) {
	m3uPlaylist, err := m3uparser.DecodeFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var filePrefix string
	switch m3uPlaylist.Type {
	case "master":
		filePrefix = "master.m3u8"
	case "media":
		filePrefix = "media.ts"
	default:
		log.Printf("Unknown m3u8 playlist type: %v\n", m3uPlaylist.Type)
		return nil, fmt.Errorf("unknown m3u8 playlist type: %v", m3uPlaylist.Type)
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
		m3uPlaylist.Entries[i].URI = fmt.Sprintf("%s?o=%s", filePrefix, remap)
	}

	return m3uPlaylist, nil
}

func (stream *M3UStreamSource) Serve(w http.ResponseWriter, r *http.Request, timeout int) {

	vars := mux.Vars(r)

	switch vars["path"] {
	case "master.m3u8":
		uri, err := readMP3Url(r, stream)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp, err := executeRequest("GET", uri.String(), stream.transport, stream.headers, timeout)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer resp.Body.Close()

		m3uPlaylist, err := remapM3UPlaylist(resp)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		m3uPlaylist.WriteTo(w)

	case "media.ts":
		uri, err := readMP3Url(r, stream)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp, err := executeRequest("GET", uri.String(), stream.transport, stream.headers, timeout)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
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
