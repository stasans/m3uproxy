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

type MemoryAuthProvide struct {
	AuthProvider
	users map[string]*User
}

func NewMemoryAuthProvider(config json.RawMessage) AuthProvider {
	return &MemoryAuthProvide{
		users: make(map[string]*User),
	}
}

func (a *MemoryAuthProvide) AuthenticateUser(username, password string) bool {
	if user, ok := a.users[username]; ok {
		return user.Password == HashPassword(password)
	}
	return false
}

func (a *MemoryAuthProvide) AddUser(username, password string) error {

	if _, ok := a.users[username]; ok {
		return fmt.Errorf("user already exists")
	}

	a.users[username].Password = HashPassword(password)
	return nil
}

func (a *MemoryAuthProvide) RemoveUser(username string) error {

	if _, ok := a.users[username]; !ok {
		return fmt.Errorf("user does not exist")
	}

	delete(a.users, username)
	return nil
}

func (a *MemoryAuthProvide) GetUsers() ([]string, error) {
	users := make([]string, 0, len(a.users))
	for user := range a.users {
		users = append(users, user)
	}
	return users, nil
}

func (a *MemoryAuthProvide) ChangePassword(username, password string) error {
	if _, ok := a.users[username]; !ok {
		return fmt.Errorf("user does not exist")
	}

	a.users[username].Password = HashPassword(password)
	return nil
}

func (a *MemoryAuthProvide) DropUsers() error {
	a.users = make(map[string]*User)
	return nil
}

func (a *MemoryAuthProvide) LoadUsers() error {
	return fmt.Errorf("not implemented")
}

func (a *MemoryAuthProvide) GetRole(username string) (string, error) {
	if user, ok := a.users[username]; ok {
		return user.Role, nil
	}
	return "", fmt.Errorf("user does not exist")
}

func (a *MemoryAuthProvide) GetUser(username string) (UserView, error) {
	if user, ok := a.users[username]; ok {
		return UserView{
			Username: user.Username,
			Role:     user.Role,
		}, nil
	}
	return UserView{}, fmt.Errorf("user does not exist")
}

func (a *MemoryAuthProvide) SetRole(username, role string) error {
	if user, ok := a.users[username]; ok {
		user.Role = role
		return nil
	}
	return fmt.Errorf("user does not exist")
}
