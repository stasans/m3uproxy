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
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/auth/authproviders"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"
	"github.com/gorilla/mux"
)

func registerAPIRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/authenticate", basicAuth(authenticateRequest))
	r.HandleFunc("/api/v1/reload", adminAccess(reloadRequest))
	r.HandleFunc("/api/v1/config", adminAccess(configAPIRequest))
	r.HandleFunc("/api/v1/playlist", adminAccess(playlistAPIRequest))
	r.HandleFunc("/api/v1/users", adminAccess(usersAPIRequest))
	r.HandleFunc("/api/v1/user/{id}", adminAccess(userAPIRequest))
}

func authenticateRequest(w http.ResponseWriter, r *http.Request) {

	authHeader := r.Header.Get("Authorization")
	authParts := strings.SplitN(authHeader, " ", 2)
	token := authParts[1]

	role, err := auth.GetRoleFromToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := auth.GetUserFromToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := fmt.Sprintf(`{"role": "%s", "user": "%s", "token": "%s"}`, role, user, token)
	w.Write([]byte(resp))
}

func configAPIRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(Config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	case http.MethodPut:
		newConfig := ServerConfig{}
		err := json.NewDecoder(r.Body).Decode(&newConfig)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = SaveServerConfig(newConfig)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func usersAPIRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		users, err := auth.GetUsers()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func userAPIRequest(w http.ResponseWriter, r *http.Request) {

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
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	case http.MethodPost:
		vars := mux.Vars(r)
		username := vars["id"]
		newUser := authproviders.User{}
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = auth.AddUser(username, newUser.Password)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if newUser.Role != "" {
			err = auth.SetRole(username, newUser.Role)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		w.WriteHeader(http.StatusCreated)
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
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if newUser.Role != "" {
			err = auth.SetRole(username, newUser.Role)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func playlistAPIRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		file, err := os.Open(Config.Playlist)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		config := m3uprovider.PlaylistConfig{}
		err = json.NewDecoder(file).Decode(&config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data, err := json.Marshal(config)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
		return
	case http.MethodPost:
		playlist := m3uprovider.PlaylistConfig{}
		err := json.NewDecoder(r.Body).Decode(&playlist)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = SavePlaylist(playlist)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func reloadRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		w.WriteHeader(http.StatusNoContent)
		Restart()
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
