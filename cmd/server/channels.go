package server

import (
	"log"
	"m3u-proxy/pkg/channelstore"
	"m3u-proxy/pkg/m3uparser"
)

func loadChannels(filePath string, checkOnline bool) error {

	playlist, err := m3uparser.ParseM3UFile(filePath)
	if err != nil {
		return err
	}

	if err := channelstore.LoadPlaylist(playlist, checkOnline); err != nil {
		return err
	}

	log.Printf("Loaded %d channels\n", channelstore.GetChannelCount())
	return nil
}
