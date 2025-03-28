package auth

import (
	"encoding/json"
	"errors"

	"github.com/a13labs/m3uproxy/pkg/auth/authproviders"
)

type AuthConfig struct {
	Provider       string          `json:"provider"`
	SecretKey      string          `json:"secret_key"`
	ExpirationTime int             `json:"expiration_time,omitempty"`
	Settings       json.RawMessage `json:"settings"`
}

var authConfig AuthConfig

func InitializeAuth(data json.RawMessage) error {

	err := json.Unmarshal(data, &authConfig)
	if err != nil {
		return err
	}

	if authConfig.Provider == "" {
		return errors.New("auth provider is required")
	}

	if len(authConfig.SecretKey) == 0 {
		return errors.New("secret key is required")
	}

	if authConfig.ExpirationTime == 0 {
		authConfig.ExpirationTime = 24
	}

	return authproviders.InitializeAuthProvider(authConfig.Provider, authConfig.Settings)
}

func GetRole(username string) (string, error) {
	return authproviders.GetRole(username)
}

func CheckCredentials(username, password string) bool {
	return authproviders.AuthenticateUser(username, password)
}

func AddUser(username, password string) error {
	return authproviders.AddUser(username, password)
}

func RemoveUser(username string) error {
	return authproviders.RemoveUser(username)
}

func GetUsers() ([]string, error) {
	return authproviders.GetUsers()
}

func ChangePassword(username, password string) error {
	return authproviders.ChangePassword(username, password)
}

func DropUsers() error {
	return authproviders.DropUsers()
}

func GetUser(username string) (authproviders.UserView, error) {
	return authproviders.GetUser(username)
}

func SetRole(username, role string) error {
	return authproviders.SetRole(username, role)
}
