/*
Copyright © 2024 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package streamserver

import (
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/elnormous/contenttype"

	"github.com/gorilla/mux"
)

type streamStruct struct {
	index            int
	m3u              m3uparser.M3UEntry
	active           bool
	mux              *sync.Mutex
	headers          map[string]string
	httpProxy        string
	forceKodiHeaders bool
	radio            bool
	transport        *http.Transport
	disableRemap     bool
}

var supportedMediaTypes = []contenttype.MediaType{
	contenttype.NewMediaType("application/vnd.apple.mpegurl"),
	contenttype.NewMediaType("application/x-mpegurl"),
	contenttype.NewMediaType("audio/x-mpegurl"),
	contenttype.NewMediaType("audio/mpeg"),
	contenttype.NewMediaType("audio/aacp"),
	contenttype.NewMediaType("audio/aac"),
	contenttype.NewMediaType("audio/mp4"),
	contenttype.NewMediaType("audio/mp3"),
	contenttype.NewMediaType("audio/ac3"),
	contenttype.NewMediaType("audio/x-aac"),
	contenttype.NewMediaType("video/mp2t"),
	contenttype.NewMediaType("video/m2ts"),
	contenttype.NewMediaType("video/mp4"),
	contenttype.NewMediaType("binary/octet-stream"),
}

func (stream *streamStruct) healthCheck() {
	resp, err := executeRequest("GET", stream.m3u.URI, stream.transport, stream.headers)
	if err != nil {
		return
	}
	resp.Body.Close()

	stream.mux.Lock()
	if resp.Request.URL.String() != stream.m3u.URI {
		stream.m3u.URI = resp.Request.URL.String()
	}
	stream.mux.Unlock()

	streamActive := verifyStream(stream.m3u.URI, stream.transport, stream.headers)

	stream.mux.Lock()
	stream.active = streamActive
	stream.mux.Unlock()
}

func (stream *streamStruct) serve(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	cache := r.URL.Query().Get("cache")

	var uri *url.URL
	if cache == "" {
		if vars["path"] != "master.m3u8" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
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

	serveAndRemap(uri.String(), stream.transport, stream.headers, w)
}

func streamRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.\n", r.URL.Path)
		return
	}

	streamID, err := strconv.Atoi(vars["streamId"])
	if err != nil {
		http.Error(w, "Invalid stream ID", http.StatusBadRequest)
		return
	}

	streamsMutex.Lock()
	defer streamsMutex.Unlock()

	if streamID < 0 || streamID >= len(streams) {
		http.Error(w, "Invalid stream ID", http.StatusBadRequest)
		return
	}

	stream := streams[streamID]
	if !stream.active {
		noServiceStream.Serve(w, r)
		return
	}
	stream.serve(w, r)
}

func registerStreamsRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/{token}/{streamId}/{path:.*}", streamRequest)
	return r
}
