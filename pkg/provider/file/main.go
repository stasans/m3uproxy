package file

import (
	"encoding/json"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/provider/types"
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
		return nil
	}

	playlist, err := m3uparser.ParseM3UFile(cfg.Source)
	if err != nil {
		return nil
	}

	return &M3UFileProvider{
		playlist: *playlist,
	}
}

func (p *M3UFileProvider) GetPlaylist() *m3uparser.M3UPlaylist {

	return &p.playlist
}
