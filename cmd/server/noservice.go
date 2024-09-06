package server

import (
	"log"
	"net/http"

	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
)

var noServiceStream *ffmpeg.HLSStream
var noServiceAvailable = false

func startNoServiceStream(config *rootCmd.Config) {
	if noServiceStream != nil {
		log.Fatalf("No service stream already started\n")
	}
	log.Printf("Generating HLS for no service image\n")
	noServiceStream = ffmpeg.GenerateImageHLS(config.NoServiceImage, "no_service")
	if err := noServiceStream.Start(); err != nil {
		log.Fatalf("Failed to start no service stream: %v\n", err)
		noServiceAvailable = false
		return
	}
	log.Printf("No service stream started\n")
	noServiceAvailable = true
}

func stopNoServiceStream() {
	if !noServiceAvailable {
		if err := noServiceStream.Stop(); err != nil {
			log.Fatalf("Failed to stop no service stream: %v\n", err)
		}
		noServiceStream.Cleanup()
		noServiceAvailable = false
	}
	log.Printf("No service stream stopped\n")
}

func noServiceHandler(w http.ResponseWriter, r *http.Request) {
	if !noServiceAvailable {
		http.NotFound(w, r)
		return
	}
	noServiceStream.Serve(w, r)
}
