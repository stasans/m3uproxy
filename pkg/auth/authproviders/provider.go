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
package authproviders

import (
	"encoding/json"
	"fmt"
)

type AuthProvider interface {
	AuthenticateUser(username, password string) bool
	AddUser(username, password string) error
	RemoveUser(username string) error
	GetUsers() ([]string, error)
	ChangePassword(username, password string) error
	DropUsers() error
	LoadUsers() error
}

type AuthProviderFactory func(config json.RawMessage) AuthProvider

var authProvider AuthProvider

func SetAuthProviderFactory(factory AuthProviderFactory, config json.RawMessage) {
	authProvider = factory(config)
}

func GetAuthProvider() AuthProvider {
	return authProvider
}

func AuthenticateUser(username, password string) bool {
	return authProvider.AuthenticateUser(username, password)
}

func AddUser(username, password string) error {
	return authProvider.AddUser(username, password)
}

func RemoveUser(username string) error {
	return authProvider.RemoveUser(username)
}

func GetUsers() ([]string, error) {
	return authProvider.GetUsers()
}

func ChangePassword(username, password string) error {
	return authProvider.ChangePassword(username, password)
}

func DropUsers() error {
	return authProvider.DropUsers()
}

func InitializeAuthProvider(provider string, config json.RawMessage) error {
	switch provider {
	case "file":
		SetAuthProviderFactory(NewFileAuthProvider, config)
	case "memory":
		SetAuthProviderFactory(NewMemoryAuthProvider, config)
	case "null":
		SetAuthProviderFactory(NewNullAuthProvider, config)
	default:
		return fmt.Errorf("unsupported auth provider")
	}
	return nil
}
