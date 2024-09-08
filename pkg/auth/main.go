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
