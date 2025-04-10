package types

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type M3U8StreamSource struct {
	BaseStreamSource
}

func (s *M3U8StreamSource) parseUrl(r *http.Request) (*url.URL, error) {
	orig := r.URL.Query().Get("o")
	if orig == "" {
		return url.Parse(s.m3u.URI)
	}

	originalUrlBytes, err := base64.URLEncoding.DecodeString(orig)
	if err != nil {
		return nil, err
	}

	return url.Parse(string(originalUrlBytes))
}

func (s *M3U8StreamSource) remap(body []byte, w http.ResponseWriter, uri *url.URL) {
	buf := bufio.NewReader(bytes.NewReader(body))

	// Validate header
	line, err := buf.ReadString('\n')
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(line, "#EXTM3U") {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write([]byte(line))
	var in_entry = false
	var playlistType = "master"

	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		if line != "" && !strings.HasPrefix(line, "#") {
			if in_entry {
				line = strings.TrimRight(line, "\r\n")
				if !strings.HasPrefix(line, "http") {
					p := path.Dir(uri.EscapedPath())
					line = uri.Scheme + "://" + path.Join(uri.Host, path.Join(p, line))
				}

				remap := base64.URLEncoding.EncodeToString([]byte(line))

				switch playlistType {
				case "master":
					w.Write([]byte(fmt.Sprintf("master.m3u8?o=%s\n", remap)))
				case "media":
					w.Write([]byte(fmt.Sprintf("media/media.ts?o=%s\n", remap)))
				default:
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				in_entry = false
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			continue
		}
		w.Write([]byte(line))

		line = strings.TrimPrefix(line, "#")
		if strings.HasPrefix(line, "EXTINF") || strings.HasPrefix(line, "EXT-X-STREAM-INF") {
			in_entry = true
			continue
		}

		if strings.HasPrefix(line, "EXT-X-INDEPENDENT-SEGMENTS") {
			playlistType = "master"
		} else if strings.HasPrefix(line, "EXT-X-MEDIA-SEQUENCE") {
			playlistType = "media"
		}
	}
}

func (s *M3U8StreamSource) MasterPlaylist() string {
	return "master.m3u8"
}

func (s *M3U8StreamSource) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {

	uri, err := s.parseUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, _, ct, err := s.conn.Get("GET", uri.String())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", ct.String())
	if s.disableRemap {
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", ct.String())
	s.remap(body, w, uri)
}

func (s *M3U8StreamSource) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {

	uri, err := s.parseUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, _, ct, err := s.conn.Get("GET", uri.String())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", ct.String())
	w.Write(body)
}
