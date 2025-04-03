package streamSources

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func readM3U8Url(r *http.Request, stream *M3UStreamSource) (*url.URL, error) {
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

func remapM3U8Playlist(resp *http.Response, w http.ResponseWriter) {
	buf := bufio.NewReader(resp.Body)

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

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
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
					basePath := path.Dir(resp.Request.URL.Path)
					line = resp.Request.URL.Scheme + "://" + path.Join(resp.Request.URL.Host, path.Join(basePath, line))
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

func (stream *M3UStreamSource) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {

	uri, err := readM3U8Url(r, stream)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := executeRequest("GET", uri.String(), stream.client, stream.headers)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	if stream.disableRemap {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		io.Copy(w, resp.Body)
		return
	}

	remapM3U8Playlist(resp, w)
}

func (stream *M3UStreamSource) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {

	uri, err := readM3U8Url(r, stream)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := executeRequest("GET", uri.String(), stream.client, stream.headers)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
	return

}

func (stream *M3UStreamSource) MasterPlaylist() string {
	return "master.m3u8"
}
