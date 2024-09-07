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
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/a13labs/m3uproxy/pkg/streamstore"
)

func loadAndParsePlaylist(path string) error {

	// If extension is .m3u, load as a m3u file
	// Otherwise, load as a json file
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
		playlist, err = m3uprovider.LoadPlaylist(path)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid file extension")
	}

	log.Printf("Loaded %d streams from %s\n", playlist.StreamCount(), path)

	if err := streamstore.AddStreams(playlist); err != nil {
		return err
	}

	log.Printf("Loaded %d streams\n", streamstore.StreamCount())
	return nil
}

func configureStreams(config *rootCmd.Config) {

	go func() {
		for {
			log.Println("Streams loading started")
			err := loadAndParsePlaylist(config.Playlist)
			if err != nil {
				log.Printf("Failed to load streams: %v\n", err)
			}
			log.Println("Checking streams availability, this may take a while")
			streamstore.MonitorStreams()
			log.Println("Streams loading completed")
			if config.ScanTime == 0 {
				config.ScanTime = 24 * 60 * 60
			}
			<-time.After(time.Duration(config.ScanTime) * time.Second)
		}
	}()
}
