package streamserver

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func setupLogging(logFile string) {
	log.SetFormatter(&logrus.JSONFormatter{})
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Warn("Failed to log to file, using default stderr")
	}
}
