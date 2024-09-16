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
	"encoding/json"
	"net/http"
	"os"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/auth/authproviders"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/gorilla/mux"
)

func registerRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/config", adminAccess(configApiRequest))
	r.HandleFunc("/api/v1/playlist", adminAccess(playlistApiRequest))
	r.HandleFunc("/api/v1/users", adminAccess(usersApiRequest))
	r.HandleFunc("/api/v1/user/{id}", adminAccess(userApiRequest))
}

func configApiRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(serverConfig)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func usersApiRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		users, err := auth.GetUsers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func userApiRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		vars := mux.Vars(r)
		username := vars["id"]
		user, err := auth.GetUser(username)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	case http.MethodPut:
		vars := mux.Vars(r)
		username := vars["id"]
		newUser := authproviders.User{}
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = auth.ChangePassword(username, newUser.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if newUser.Role != "" {
			err = auth.SetRole(username, newUser.Role)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	case http.MethodDelete:
		vars := mux.Vars(r)
		username := vars["id"]
		err := auth.RemoveUser(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func playlistApiRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		file, err := os.Open(serverConfig.Playlist)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		config := m3uprovider.PlaylistConfig{}
		err = json.NewDecoder(file).Decode(&config)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(config)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	case http.MethodPost:
		config := m3uprovider.PlaylistConfig{}
		err := json.NewDecoder(r.Body).Decode(&config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, err := os.Create(serverConfig.Playlist)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()
		err = json.NewEncoder(file).Encode(config)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		updateTimer.Reset(0)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
