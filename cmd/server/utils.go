package server

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func loadContent(filePath string) (string, error) {
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		// Load content from URL
		resp, err := http.Get(filePath)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	} else {
		// Load content from local file
		file, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		body, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
}

func setupLogging() {
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}
}

func verifyAuth(r *http.Request) (string, string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", false
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 || authParts[0] != "Basic" {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(authParts[1])
	if err != nil {
		return "", "", false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return "", "", false
	}

	return credentials[0], credentials[1], true
}
