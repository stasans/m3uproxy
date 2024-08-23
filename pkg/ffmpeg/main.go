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
package ffmpeg

import (
	"errors"
	"log"
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
func GenerateHLS(imagePath, streamName string) {
	// Prepare the FFmpeg command
	go func() {
		cmd := exec.Command("ffmpeg", "-loop", "1", "-i", imagePath,
			"-c:v", "libx264", "-pix_fmt", "yuv420p",
			"-vf", "scale=1280:720,fps=30", "-hls_time", "1", "-hls_list_size", "40",
			"-hls_flags", "delete_segments", "-start_number", "1", filepath.Join(workingDir, streamName+".m3u8"))

		log.Printf("running command: %v", cmd)
		// Run the FFmpeg command
		cmd.Run()
	}()
}

func Cleanup() {
	// Remove the temporary directory
	os.RemoveAll(workingDir)
}

// serveHLS handles the HTTP requests to serve the HLS segments
func ServeHLS(w http.ResponseWriter, r *http.Request) {
	// Get the requested file path
	vars := mux.Vars(r)
	path := vars["path"]

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
