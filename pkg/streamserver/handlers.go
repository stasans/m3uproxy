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
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/gorilla/mux"
)

func handleGetStream(w http.ResponseWriter, r *http.Request) {

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

func handleGetPlaylist(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	authParts := strings.SplitN(authHeader, " ", 2)
	token := authParts[1]

	streamsMutex.Lock()
	defer streamsMutex.Unlock()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("#EXTM3U\n"))
	for i, stream := range streams {
		if serverConfig.HideInactive && !stream.active {
			continue
		}

		protocol := "http"
		if serverConfig.UseHttps && r.TLS == nil {
			protocol = "https"
		}

		uri := fmt.Sprintf("%s://%s/%s/%d/%s", protocol, r.Host, token, i, m3uPlaylist)
		if stream.disableRemap {
			uri = stream.m3u.URI
		}

		entry := m3uparser.M3UEntry{
			URI:   uri,
			Title: stream.m3u.Title,
			Tags:  make([]m3uparser.M3UTag, 0),
		}
		entry.Tags = append(entry.Tags, stream.m3u.Tags...)
		if !stream.radio {
			if stream.forceKodiHeaders || (serverConfig.KodiSupport && !stream.radio) {
				entry.AddTag("KODIPROP", "inputstream=inputstream.adaptive")
				entry.AddTag("KODIPROP", "inputstream.adaptive.manifest_type=hls")
			}
		}
		w.Write([]byte(entry.String() + "\n"))
	}
}

func handleGetEpg(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content, err := loadContent(serverConfig.Epg)
	if err != nil {
		http.Error(w, "EPG file not found", http.StatusNotFound)
		log.Printf("EPG file not found at %s\n", serverConfig.Epg)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

func handlerGetPlayer(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content, err := loadContent("assets/player.html")
	if err != nil {
		http.Error(w, "Player file not found", http.StatusNotFound)
		log.Printf("Player file not found at %s\n", "player.html")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

func handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	authHeader := r.Header.Get("Authorization")
	authParts := strings.SplitN(authHeader, " ", 2)
	token := authParts[1]

	role, err := auth.GetRoleFromToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := auth.GetUserFromToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp := fmt.Sprintf(`{"role": "%s", "user": "%s", "token": "%s"}`, role, user, token)
	w.Write([]byte(resp))
	w.WriteHeader(http.StatusOK)
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if streams == nil {
		http.Error(w, "Streams not loaded", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
