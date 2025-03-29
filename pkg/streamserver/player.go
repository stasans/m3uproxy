package streamserver

import (
	"archive/zip"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Variable to hold the in-memory zip file structure
var playerFiles map[string]*zip.File

func playerRequest(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Remove leading '/' from request path
	filePath := strings.TrimPrefix(r.URL.Path, "/player/")

	// Lookup the requested file in the zip archive
	if zipFile, found := playerFiles[filePath]; found {
		// Open the zip file for reading
		fileReader, err := zipFile.Open()
		if err != nil {
			http.Error(w, "Could not open file", http.StatusInternalServerError)
			return
		}
		defer fileReader.Close()

		// Get the file's content type
		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		w.Header().Set("Content-Type", contentType)

		// Copy the file contents to the response writer
		if _, err := io.Copy(w, fileReader); err != nil {
			http.Error(w, "Could not serve file", http.StatusInternalServerError)
		}
	} else {
		http.NotFound(w, r)
	}

}

func CachePlayer() error {
	if playerFiles != nil {
		return nil
	}

	playerFiles = make(map[string]*zip.File)
	if _, err := os.Stat("assets/player.zip"); os.IsNotExist(err) {
		return err
	}

	zipReader, err := zip.OpenReader("assets/player.zip")
	if err != nil {
		return err
	}

	// Store each file in the map with its name as the key
	for _, f := range zipReader.File {
		playerFiles[f.Name] = f
	}

	return nil
}

func registerPlayerRoutes(r *mux.Router) *mux.Router {
	if err := CachePlayer(); err != nil {
		return r
	}
	r.HandleFunc("/player", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/player/index.html", http.StatusSeeOther)
	})
	r.PathPrefix("/player/").HandlerFunc(playerRequest)
	return r
}
