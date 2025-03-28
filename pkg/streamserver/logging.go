package streamserver

import (
	"log"
	"os"
)

func setupLogging(out string) {
	if out != "" {
		file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}
