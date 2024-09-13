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
	"bufio"
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
	"github.com/grafov/m3u8"
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

func getUserCredentials(r *http.Request) (string, string, bool) {
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

func getJWTToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 || authParts[0] != "Bearer" {
		return "", false
	}

	return authParts[1], true
}

func updatePlaylistAndMonitor(config StreamServerConfig, stopServer chan bool, quit chan bool) {
	log.Println("Streams loading started")
	err := loadAndParsePlaylist(config.Playlist)
	if err != nil {
		log.Printf("Failed to load streams: %v\n", err)
	}
	log.Println("Checking streams availability, this may take a while")
	if config.NumWorkers == 1 {
		MonitorStreamsNoWorkers()
	} else {
		MonitorStreams(stopServer, quit)
	}
	log.Println("Streams loading completed")

}

func readStream(path string, stream *Stream, client *http.Client) bool {

	req, err := stream.HttpRequest(StreamRequestOptions{
		Path:   path,
		Method: "GET",
	})
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	contentType := resp.Header.Get("Content-Type")
	parts := strings.Split(contentType, ";")
	if len(parts) > 1 {
		contentType = strings.TrimRight(parts[0], " ")
	}
	contentType = strings.ToLower(contentType)
	switch contentType {
	case "application/vnd.apple.mpegurl":

		p, listType, err := m3u8.DecodeWith(bufio.NewReader(resp.Body), true, []m3u8.CustomDecoder{})
		if err != nil {
			log.Printf("Failed to decode m3u8 playlist: %v\n", err)
			return false
		}
		resp.Body.Close()

		switch listType {
		case m3u8.MASTER:
			if resp.Request.URL.String() != stream.m3u.URI {

				stream.m3u.URI = resp.Request.URL.String()
				if strings.LastIndex(resp.Request.URL.Path, "/") != -1 {
					stream.prefix = resp.Request.URL.Path[:strings.LastIndex(resp.Request.URL.Path, "/")]
				} else {
					stream.prefix = ""
				}
			}

			playlist := p.(*m3u8.MasterPlaylist)
			if playlist == nil {
				return false
			}

			if len(playlist.Variants) == 0 {
				return false
			}

			variant := playlist.Variants[0]
			if variant == nil {
				return false
			}

			if !readStream(variant.URI, stream, client) {
				return false
			}
			stream.masterPlaylist = playlist
			return true

		case m3u8.MEDIA:
			mediaPlaylist := p.(*m3u8.MediaPlaylist)
			segment := mediaPlaylist.Segments[0]
			if segment == nil {
				return false
			}
			return readStream(segment.URI, stream, client)
		default:
			log.Printf("Unknown m3u8 playlist type: %v\n", listType)
			return false
		}
	case "application/x-mpegurl":
		fallthrough
	case "audio/x-mpegurl":
		fallthrough
	case "audio/mpeg":
		fallthrough
	case "audio/aacp":
		fallthrough
	case "audio/aac":
		fallthrough
	case "audio/mp4":
		fallthrough
	case "audio/x-aac":
		fallthrough
	case "video/mp2t":
		fallthrough
	case "binary/octet-stream":
		return true
	default:
		return false
	}
}
