package streamSources

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
	"github.com/valyala/fasthttp"
)

func readMPDUrl(r *http.Request) (*url.URL, error) {
	vars := mux.Vars(r)
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

func remapMPDPlaylist(resp *fasthttp.Response, uri *url.URL) (*mpd.MPD, error) {
	mpdPlaylist, err := mpd.DecodeFromReader(bytes.NewReader(resp.Body()))
	if err != nil {
		return nil, err
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
	return mpdPlaylist, nil
}

func (stream *MPDStreamSource) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {

	// if the cache is empty, we must be serving the master playlist
	uri, _ := url.Parse(stream.m3u.URI)

	resp, err := executeRequest("GET", uri.String(), stream.client, stream.headers)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fasthttp.ReleaseResponse(resp)

	ct, valid := contentTypeAllowed(resp)
	if !valid {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	mpdPlaylist, err := remapMPDPlaylist(resp, uri)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", ct.String())
	mpdPlaylist.WriteTo(w)

}

func (stream *MPDStreamSource) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {

	uri, err := readMPDUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := executeRequest("GET", uri.String(), stream.client, stream.headers)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fasthttp.ReleaseResponse(resp)

	ct, valid := contentTypeAllowed(resp)
	if !valid {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	w.Header().Set("Content-Type", ct.String())
	w.Write(resp.Body())
}

func (stream *MPDStreamSource) MasterPlaylist() string {
	return "master.mpd"
}
