package ffmpeg

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
)

type HLSStream struct {
	Name      string
	ffmpegCmd *exec.Cmd
	localPath string
}

// serveHLS handles the HTTP requests to serve the HLS segments
func (s *HLSStream) Serve(w http.ResponseWriter, r *http.Request) {
	// Get the requested file path
	vars := mux.Vars(r)
	path := vars["path"]

	// Set the appropriate content type for .m3u8 and .ts files
	if filepath.Ext(path) == ".m3u8" {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	} else if filepath.Ext(path) == ".ts" {
		w.Header().Set("Content-Type", "video/mp2t")
	} else {
		http.NotFound(w, r)
		return
	}

	localPath := filepath.Join(s.localPath, filepath.Base(path))
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Serve the requested file
	http.ServeFile(w, r, localPath)
}

// Start starts the HLS stream
func (s *HLSStream) Start() error {
	return s.ffmpegCmd.Start()
}

// Stop stops the HLS stream
func (s *HLSStream) Stop() error {
	return s.ffmpegCmd.Process.Kill()
}

// Cleanup removes the HLS segments
func (s *HLSStream) Cleanup() {
	os.RemoveAll(s.localPath)
}
