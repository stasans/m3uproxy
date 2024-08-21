package ffmpeg

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
)

var workingDir string

// checkFFmpegAvailability checks if FFmpeg is installed on the system
func isFFmpegAvailable() error {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run()
}

func Initialize() error {

	// Check if FFmpeg is available
	if err := isFFmpegAvailable(); err != nil {
		return errors.New("FFmpeg is not installed")
	}

	// Create a temporary directory to store the HLS segments
	workingDir, _ = os.MkdirTemp("", "hls")

	return nil
}

// generateHLS creates HLS segments and playlist from an image using FFmpeg
func GenerateHLS(imagePath, streamName string) error {
	// Prepare the FFmpeg command
	cmd := exec.Command("ffmpeg", "-loop", "1", "-i", imagePath,
		"-c:v", "libx264", "-t", "10", "-pix_fmt", "yuv420p",
		"-vf", "scale=1280:720,fps=30", "-hls_time", "1", "-hls_list_size", "0",
		"-hls_wrap", "0", "-start_number", "1", filepath.Join(workingDir, streamName+".m3u8"))

	// Run the FFmpeg command
	return cmd.Run()
}

func Cleanup() {
	// Remove the temporary directory
	os.RemoveAll(workingDir)
}

// serveHLS handles the HTTP requests to serve the HLS segments
func ServeHLS(w http.ResponseWriter, r *http.Request) {
	// Get the requested file path
	vars := mux.Vars(r)
	path := vars["stream"]

	// Set the appropriate content type for .m3u8 and .ts files
	if filepath.Ext(path) == ".m3u8" {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	} else if filepath.Ext(path) == ".ts" {
		w.Header().Set("Content-Type", "video/mp2t")
	}

	localPath := filepath.Join(workingDir, filepath.Base(path))
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Serve the requested file
	http.ServeFile(w, r, localPath)
}
