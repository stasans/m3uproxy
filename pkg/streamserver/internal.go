package streamserver

import (
	"io"
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
