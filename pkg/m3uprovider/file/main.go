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

func NewM3UFileProvider(config string) *M3UFileProvider {

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
