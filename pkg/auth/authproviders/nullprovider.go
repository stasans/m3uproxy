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

type NullAuthProvider struct {
	AuthProvider
}

func NewNullAuthProvider(config json.RawMessage) AuthProvider {
	return &NullAuthProvider{}
}

func (a *NullAuthProvider) AuthenticateUser(username, password string) bool {
	return true
}

func (a *NullAuthProvider) AddUser(username, password string) error {
	return nil
}

func (a *NullAuthProvider) RemoveUser(username string) error {
	return nil
}

func (a *NullAuthProvider) GetUsers() ([]string, error) {
	users := make([]string, 0)
	return users, nil
}

func (a *NullAuthProvider) ChangePassword(username, password string) error {
	return nil
}

func (a *NullAuthProvider) DropUsers() error {
	return nil
}

func (a *NullAuthProvider) LoadUsers() error {
	return fmt.Errorf("not implemented")
}

func (a *NullAuthProvider) GetRole(username string) (string, error) {
	if username == "admin" {
		return "admin", nil
	}
	return "viewer", nil
}

func (a *NullAuthProvider) GetUser(username string) (UserView, error) {
	return UserView{}, fmt.Errorf("not implemented")
}

func (a *NullAuthProvider) SetRole(username, role string) error {
	return fmt.Errorf("not implemented")
}
