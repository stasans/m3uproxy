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
	"log"
	"time"

	"github.com/a13labs/m3uproxy/pkg/channelstore"
	"github.com/a13labs/m3uproxy/pkg/m3uparser"
)

func loadChannels(filePath string) error {

	playlist, err := m3uparser.ParseM3UFile(filePath)
	if err != nil {
		return err
	}

	if err := channelstore.LoadPlaylist(playlist); err != nil {
		return err
	}

	log.Printf("Loaded %d channels\n", channelstore.GetChannelCount())
	return nil
}

func reloadChannels(scanTime int) {
	for {
		log.Println("Reloading channels")
		err := loadChannels(m3uFilePath)
		if err != nil {
			log.Printf("Failed to reload channels: %v\n", err)
		}
		log.Println("Checking channels availability, this may take a while")
		channelstore.CheckChannels()
		log.Println("Channels reloaded")
		<-time.After(time.Duration(scanTime) * time.Second)
	}
}
