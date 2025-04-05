package sources

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/a13labs/m3uproxy/pkg/m3uparser"
	"github.com/a13labs/m3uproxy/pkg/sources/types"
)

type Sources struct {
	sources      []types.StreamSource
	mux          *sync.Mutex
	activeSource types.StreamSource
}

func NewSources() Sources {
	return Sources{
		sources:      make([]types.StreamSource, 0),
		mux:          &sync.Mutex{},
		activeSource: nil,
	}
}

func (s *Sources) Children() []types.StreamSource {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.sources
}

func (s *Sources) SourceExists(entry m3uparser.M3UEntry) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, source := range s.sources {
		if source.Url() == entry.URI {
			return true
		}
	}
	return false
}

func (s *Sources) AddSource(entry m3uparser.M3UEntry, timeout int) (bool, error) {
	if s.SourceExists(entry) {
		return false, fmt.Errorf("source already exists")
	}

	source, err := types.NewSource(entry, timeout)
	if err != nil {
		return false, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.sources = append(s.sources, source)
	return true, nil
}

func (s *Sources) GetActiveSource() types.StreamSource {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.activeSource
}

func (s *Sources) HealthCheck() error {
	var activeSource types.StreamSource = nil
	for _, source := range s.sources {
		_ = source.HealthCheck()
		if source.Active() {
			activeSource = source
			break
		}
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.activeSource = activeSource
	if activeSource == nil {
		return fmt.Errorf("no active stream source found")
	}
	return nil
}

func (s *Sources) ServeManifest(w http.ResponseWriter, r *http.Request, timeout int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.activeSource == nil {
		http.Error(w, "No active stream source", http.StatusServiceUnavailable)
		return
	}

	s.activeSource.ServeManifest(w, r, timeout)
}

func (s *Sources) ServeMedia(w http.ResponseWriter, r *http.Request, timeout int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.activeSource == nil {
		http.Error(w, "No active stream source", http.StatusServiceUnavailable)
		return
	}

	s.activeSource.ServeMedia(w, r, timeout)
}

func (s *Sources) Active() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.activeSource != nil
}

func (s *Sources) MediaName() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.MediaName()
	}
	return ""
}

func (s *Sources) MasterPlaylist() string {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.MasterPlaylist()
	}
	return ""
}

func (s *Sources) M3UTags() m3uparser.M3UTags {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.M3UTags()
	}
	return nil
}

func (s *Sources) IsRadio() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.activeSource != nil {
		return s.activeSource.IsRadio()
	}
	return false
}
