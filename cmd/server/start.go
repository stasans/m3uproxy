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
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"m3u-proxy/pkg/m3uparser"
	"m3u-proxy/pkg/userstore"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

type Channel struct {
	Entry m3uparser.M3UEntry `json:"entry"`
	Name  string             `json:"name"`
}

type channelsCacheData struct {
	baseURL string
	active  bool
}

var (
	m3uFilePath   = "streams.m3u"
	epgFilePath   = "epg.xml"
	usersFilePath = "users.json"
	logFile       = ""
)

var (
	channels      = make([]Channel, 0)
	channelsCache = make([]channelsCacheData, 0)
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
		r.HandleFunc("/{token}/{channelId}/{extraReq:.*}", proxyHandler).Methods("GET")
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
}

func setupLogging() {
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}

func loadContent(filePath string) (string, error) {
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		// Load content from URL
		resp, err := http.Get(filePath)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	} else {
		// Load content from local file
		file, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		body, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
}

func validateURL(urlPath string) bool {
	_, err := url.Parse(urlPath)
	return err == nil
}

func validateChannel(channel Channel, checkOnline bool) bool {
	if channel.Entry.URI == "" || channel.Name == "" {
		return false
	}
	if !validateURL(channel.Entry.URI) {
		return false
	}
	if checkOnline {
		client := http.Client{
			Timeout: 3 * time.Second,
		}
		req, err := client.Get(channel.Entry.URI)
		if err != nil {
			return false
		}
		if req.StatusCode != http.StatusOK {
			return false
		}
	}

	return true
}

func loadM3U(filePath string) error {

	playlist, err := m3uparser.ParseM3UFile(filePath)
	if err != nil {
		return err
	}

	for _, entry := range playlist.Entries {
		if entry.URI == "" {
			continue
		}
		channel := Channel{
			Entry: entry,
			Name:  entry.Title,
		}
		if !validateChannel(channel, false) {
			log.Printf("Invalid channel: %s\n", channel.Name)
			continue
		}
		parsedURL, _ := url.Parse(channel.Entry.URI)
		baseURL := parsedURL.Scheme + "://" + parsedURL.Host
		if strings.LastIndex(parsedURL.Path, "/") != -1 {
			baseURL += parsedURL.Path[:strings.LastIndex(parsedURL.Path, "/")]
		}

		log.Printf("Loaded channel: %s\n", channel.Name)
		channels = append(channels, channel)
		channelsCache = append(channelsCache, channelsCacheData{baseURL: baseURL, active: true})

	}

	log.Printf("Loaded %d channels\n", len(channels))
	return nil
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	// token := vars["token"]
	channelID, err := strconv.Atoi(vars["channelId"])

	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		log.Printf("Invalid channel ID: %s\n", vars["channelID"])
		return
	}

	// if !validateToken(token) {
	// 	http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
	// 	log.Printf("Unauthorized access attempt to channel %d with token %s\n", channelID, token)
	// 	return
	// }

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
		log.Printf("Requesting channel '%d' streams: %s\n", channelID, serviceURL)
	} else {
		cacheData := channelsCache[channelID]
		serviceURL = cacheData.baseURL + "/" + extraReq

		if r.URL.RawQuery != "" {
			serviceURL += "?" + r.URL.RawQuery
		}
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		log.Printf("Failed to create request for channel %d: %v\n", channelID, err)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
			log.Printf("Header: %s: %s\n", key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch channel", http.StatusInternalServerError)
		log.Printf("Failed to fetch channel %d: %v\n", channelID, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch channel", resp.StatusCode)
		log.Printf("Failed to fetch channel %d: %v\n", channelID, resp.StatusCode)
		return
	}

	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func generateM3U(host, token string) string {
	m3uContent := "#EXTM3U\n"
	for i, channel := range channels {

		for _, tag := range channel.Entry.Tags {
			m3uContent += fmt.Sprintf("#%s:%s\n", tag.Tag, tag.Value)
		}
		m3uContent += fmt.Sprintf("http://%s/%s/%d/stream\n", host, token, i)
	}
	return m3uContent
}

func channelsHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		log.Printf("Unauthorized access to /channels: missing Authorization header\n")
		return
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 || authParts[0] != "Basic" {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		log.Printf("Unauthorized access to /channels: invalid Authorization header format\n")
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(authParts[1])
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Invalid base64 encoding in Authorization header", http.StatusUnauthorized)
		log.Printf("Unauthorized access to /channels: invalid base64 encoding\n")
		return
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Invalid credentials format", http.StatusUnauthorized)
		log.Printf("Unauthorized access to /channels: invalid credentials format\n")
		return
	}

	username, password := credentials[0], credentials[1]

	token, err := userstore.GenerateToken(username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		log.Printf("Unauthorized access to /channels: invalid credentials\n")
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
