package streamserver

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

func epgRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content, err := loadContent(Config.Epg)
	if err != nil {
		http.Error(w, "EPG file not found", http.StatusNotFound)
		log.Printf("EPG file not found at %s\n", Config.Epg)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

func registerEpgRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/epg.xml", epgRequest)
	return r
}

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
