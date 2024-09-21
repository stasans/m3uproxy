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
