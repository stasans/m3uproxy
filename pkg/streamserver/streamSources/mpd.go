package streamSources

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"

	mpd "github.com/a13labs/m3uproxy/pkg/mpdparser"
	"github.com/elnormous/contenttype"
	"github.com/gorilla/mux"
)

func (stream *MPDStreamSource) Serve(w http.ResponseWriter, r *http.Request, timeout int) {

	vars := mux.Vars(r)

	var uri *url.URL
	rPath := vars["rPath"]
	if rPath[0:4] == "mpd-" {
		// get index of the first slash after mpd/
		index := 4
		for i := 4; i < len(rPath); i++ {
			if rPath[i] == '/' {
				index = i
				break
			}
		}
		encodedUrl := rPath[4:index]
		baseUrl, err := base64.URLEncoding.DecodeString(encodedUrl)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		targetUrl := rPath[index+1:]
		uri, _ = url.Parse(string(baseUrl) + "/" + targetUrl)
	} else if vars["path"] == "master.mpd" {
		// if the cache is empty, we must be serving the master playlist
		uri, _ = url.Parse(stream.m3u.URI)
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
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

	w.Header().Set("Content-Type", ct)

	mpdPlaylist, err := mpd.DecodeFromReader(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for i := range mpdPlaylist.Period {
		for j := range mpdPlaylist.Period[i].AdaptationSets {
			for k := range mpdPlaylist.Period[i].AdaptationSets[j].Representations {

				if len(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL) == 0 {
					uri := new(url.URL)
					uri.Scheme = resp.Request.URL.Scheme
					uri.Host = resp.Request.URL.Host
					basePath := path.Dir(resp.Request.URL.Path)
					uri.Path = path.Join(basePath, uri.Path)
					remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
					mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL = append(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL, &mpd.BaseURL{Value: fmt.Sprintf("mpd-%s/", remap)})
					continue
				} else {
					for l := range mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL {
						currentBaseURL := mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value

						uri, _ := url.Parse(currentBaseURL)

						if uri.Scheme == "" {
							uri.Scheme = resp.Request.URL.Scheme
							uri.Host = resp.Request.URL.Host
							basePath := path.Dir(resp.Request.URL.Path)
							uri.Path = path.Join(basePath, uri.Path)
						}

						remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
						mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value = fmt.Sprintf("mpd-%s/", remap)
					}
				}
			}
		}
	}

	mpdPlaylist.WriteTo(w)
}

func (stream *MPDStreamSource) MasterPlaylist() string {
	return "master.mpd"
}
