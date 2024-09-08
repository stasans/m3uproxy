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
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
)

func monitorWorker(stream <-chan int, stop <-chan bool, wg *sync.WaitGroup, timeout int) {

	defer wg.Done()
	for s := range stream {
		select {
		case <-stop:
			return
		default:
			streams[s].HealthCheck(timeout)
			if !streams[s].active {
				log.Printf("Stream '%s' is offline.\n", streams[s].m3u.Title)
			}
		}
	}
}

func loadAndParsePlaylist(path string) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("File %s not found\n", path)
		return nil
	}

	extension := filepath.Ext(path)
	var playlist *m3uparser.M3UPlaylist
	var err error
	if extension == ".m3u" {
		log.Printf("Loading M3U file %s\n", path)
		playlist, err = m3uparser.ParseM3UFile(path)
		if err != nil {
			return err
		}
	} else if extension == ".json" {
		log.Printf("Loading JSON file %s\n", path)
		playlist, err = m3uprovider.LoadFromFile(path)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid file extension")
	}

	log.Printf("Loaded %d streams from %s\n", playlist.StreamCount(), path)

	if err := AddStreams(playlist); err != nil {
		return err
	}

	log.Printf("Loaded %d streams\n", len(streams))
	return nil
}

func verifyAuth(r *http.Request) (string, string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", false
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 || authParts[0] != "Basic" {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(authParts[1])
	if err != nil {
		return "", "", false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return "", "", false
	}

	return credentials[0], credentials[1], true
}

func updatePlaylistAndMonitor(config StreamServerConfig, stopServer chan bool, quit chan bool) {
	log.Println("Streams loading started")
	err := loadAndParsePlaylist(config.Playlist)
	if err != nil {
		log.Printf("Failed to load streams: %v\n", err)
	}
	log.Println("Checking streams availability, this may take a while")
	MonitorStreams(stopServer, quit)
	log.Println("Streams loading completed")

}
