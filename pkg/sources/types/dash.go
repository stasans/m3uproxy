package types

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	mpd "github.com/a13labs/m3uproxy/pkg/mpdparser"
	"github.com/gorilla/mux"
)

type MPDStreamSource struct {
	BaseStreamSource
}

func (s *MPDStreamSource) parseUrl(r *http.Request) (*url.URL, error) {

	vars := mux.Vars(r)

	if vars["path"] == "master.mpd" {
		return url.Parse(s.m3u.URI)
	}

	parts := strings.Split(vars["path"], "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path")
	}
	encodedUrl := parts[0]
	baseUrl, err := base64.URLEncoding.DecodeString(encodedUrl)
	if err != nil {
		return nil, err
	}

	index := strings.Index(vars["path"], "/")
	targetUrl := vars["path"][index+1:]
	uri, err := url.Parse(string(baseUrl) + "/" + targetUrl)
	if err != nil {
		return nil, err
	}

	return uri, nil
}

func (s *MPDStreamSource) remap(body []byte, w http.ResponseWriter, uri *url.URL) {
	mpdPlaylist, err := mpd.DecodeFromReader(bytes.NewReader(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for i := range mpdPlaylist.Period {
		for j := range mpdPlaylist.Period[i].AdaptationSets {
			for k := range mpdPlaylist.Period[i].AdaptationSets[j].Representations {

				if len(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL) == 0 {
					uri := new(url.URL)
					uri.Scheme = uri.Scheme
					uri.Host = string(uri.Host)
					basePath := path.Dir(string(uri.EscapedPath()))
					uri.Path = path.Join(basePath, uri.Path)
					remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
					mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL = append(mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL, &mpd.BaseURL{Value: fmt.Sprintf("media/%s/", remap)})
					continue
				}

				for l := range mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL {
					currentBaseURL := mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value

					uri, _ := url.Parse(currentBaseURL)

					if uri.Scheme == "" {
						uri.Scheme = uri.Scheme
						uri.Host = string(uri.Host)
						basePath := path.Dir(string(uri.EscapedPath()))
						uri.Path = path.Join(basePath, uri.Path)
					}

					remap := base64.URLEncoding.EncodeToString([]byte(uri.String()))
					mpdPlaylist.Period[i].AdaptationSets[j].Representations[k].BaseURL[l].Value = fmt.Sprintf("media/%s/", remap)
				}
			}
		}
	}

	mpdPlaylist.WriteTo(w)
}

func (s *MPDStreamSource) MasterPlaylist() string {
	return "master.mpd"
}

func (s *MPDStreamSource) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {

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

func (s *MPDStreamSource) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {

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
