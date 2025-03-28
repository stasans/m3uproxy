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

func registerAPIRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/api/v1/authenticate", basicAuth(authenticateRequest))
	r.HandleFunc("/api/v1/reload", adminAccess(reloadRequest))
	r.HandleFunc("/api/v1/config", adminAccess(configAPIRequest))
	r.HandleFunc("/api/v1/playlist", adminAccess(playlistAPIRequest))
	r.HandleFunc("/api/v1/users", adminAccess(usersAPIRequest))
	r.HandleFunc("/api/v1/user/{id}", adminAccess(userAPIRequest))
	return r
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
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
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
