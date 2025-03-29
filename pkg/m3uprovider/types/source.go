package types

import "github.com/a13labs/m3uproxy/pkg/m3uparser"

type M3UProvider interface {
	GetPlaylist() *m3uparser.M3UPlaylist
}
