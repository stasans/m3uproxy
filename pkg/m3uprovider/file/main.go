package file

import (
	"encoding/json"
	"log"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider/types"
)

type M3UFileConfig struct {
	Source string `json:"source"`
}

type M3UFileProvider struct {
	types.M3UProvider
	playlist m3uparser.M3UPlaylist
}

func NewM3UFileProvider(config json.RawMessage) *M3UFileProvider {

	cfg := M3UFileConfig{}
	err := json.Unmarshal([]byte(config), &cfg)
	if err != nil {
		log.Println("Error parsing config")
		return nil
	}

	log.Printf("Parsing M3U file: %s", cfg.Source)
	playlist, err := m3uparser.ParseM3UFile(cfg.Source)
	if err != nil {
		log.Printf("Error parsing M3U file: %s", err)
		return nil
	}
	log.Printf("M3U file parsed: %d entries", len(playlist.Entries))

	return &M3UFileProvider{
		playlist: *playlist,
	}
}

func (p *M3UFileProvider) GetPlaylist() *m3uparser.M3UPlaylist {

	return &p.playlist
}
