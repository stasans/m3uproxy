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
package server

import (
	"io"
	"log"
	"m3u-proxy/pkg/ffmpeg"
	"m3u-proxy/pkg/userstore"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

var (
	m3uFilePath        = "streams.m3u"
	epgFilePath        = "epg.xml"
	usersFilePath      = "users.json"
	noServiceImage     = "no_service_pt.png"
	logFile            = ""
	noServiceAvailable = false
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the M3U proxy server",
	Long:  `Start the M3U proxy server that proxies M3U playlists and EPG data.`,
	Run: func(cmd *cobra.Command, args []string) {
		setupLogging()

		log.Printf("Starting M3U Proxy Server\n")
		log.Printf("M3U file: %s\n", m3uFilePath)
		log.Printf("EPG file: %s\n", epgFilePath)

		err := loadM3U(m3uFilePath)
		if err != nil {
			log.Fatalf("Failed to load M3U file: %v\n", err)
			return
		}

		r := mux.NewRouter()
		r.HandleFunc("/channels.m3u", channelsHandler).Methods("GET")
		r.HandleFunc("/epg.xml", epgHandler).Methods("GET")

		if ffmpeg.Initialize() == nil {
			log.Printf("FFmpeg initialized successfully. Registering /stream for streams.\n")
			r.HandleFunc("/stream/{stream:.*}", ffmpeg.ServeHLS).Methods("GET")
			if _, err := os.Stat(noServiceImage); os.IsNotExist(err) {
				log.Fatalf("No service image not found at %s\n", noServiceImage)
			} else {
				log.Printf("Generating HLS for no service image\n")
				ffmpeg.GenerateHLS(noServiceImage, "no_service")
				noServiceAvailable = true
			}
		}
		r.HandleFunc("/channel/{token}/{channelId}/{extraReq:.*}", proxyHandler).Methods("GET")
		r.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
			return r.URL.Path == "/"
		}).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/channels.m3u", http.StatusMovedPermanently)
		})
		http.Handle("/", r)
		log.Println("Server running on :8080")
		err = http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	},
}

func init() {
	serverCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&m3uFilePath, "m3u", "m", "streams.m3u", "Path to the M3U file (local or remote)")
	startCmd.Flags().StringVarP(&epgFilePath, "epg", "e", "epg.xml", "Path to the EPG file (local or remote)")
	startCmd.Flags().StringVarP(&usersFilePath, "users", "u", "users.json", "Path to the users JSON file")
	startCmd.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	startCmd.Flags().StringVarP(&noServiceImage, "no-service-image", "i", "no_service_pt.png", "Path to the no service image")
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]

	ok := userstore.ValidateSingleToken(token)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("Unauthorized access to channel stream %s: missing token\n", r.URL.Path)
		return
	}

	channelID, err := strconv.Atoi(vars["channelId"])
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		log.Printf("Invalid channel ID: %s\n", vars["channelID"])
		return
	}

	if channelID < 0 || channelID >= len(channels) {
		http.Error(w, "Channel not found", http.StatusNotFound)
		log.Printf("Channel %d not found\n", channelID)
		return
	}

	channel := channels[channelID]

	// Request the channel stream with the same headers as the client request
	serviceURL := ""

	extraReq := vars["extraReq"]
	if extraReq == "stream" {
		serviceURL = channel.Entry.URI
	} else {
		cacheData := channelsCache[channelID]
		serviceURL = cacheData.baseURL + "/" + extraReq

		if r.URL.RawQuery != "" {
			serviceURL += "?" + r.URL.RawQuery
		}
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		if noServiceAvailable {
			http.Redirect(w, r, "/stream/no_service.m3u8", http.StatusMovedPermanently)
		} else {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
		}
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if noServiceAvailable {
			http.Redirect(w, r, "/stream/no_service.m3u8", http.StatusMovedPermanently)
		} else {
			http.Error(w, "Failed to fetch channel", http.StatusInternalServerError)
		}
		return
	}

	if resp.StatusCode != http.StatusOK {
		if noServiceAvailable {
			http.Redirect(w, r, "/stream/no_service.m3u8", http.StatusMovedPermanently)
		} else {
			http.Error(w, "Failed to fetch channel", resp.StatusCode)
		}
		return
	}

	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func channelsHandler(w http.ResponseWriter, r *http.Request) {

	username, password, ok := verifyAuth(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream: invalid credentials\n")
		return
	}

	token, err := userstore.GenerateToken(username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream: invalid credentials\n")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(generateM3U(r.Host, token)))
	log.Printf("Generated M3U playlist for user %s\n", username)
}

func epgHandler(w http.ResponseWriter, r *http.Request) {
	content, err := loadContent(epgFilePath)
	if err != nil {
		http.Error(w, "EPG file not found", http.StatusNotFound)
		log.Printf("EPG file not found at %s\n", epgFilePath)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
	log.Printf("EPG data served successfully\n")
}
