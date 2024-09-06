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
package userstore

import (
	"fmt"
	"sync"
	"time"

	"github.com/a13labs/m3uproxy/pkg/userstore/auth"
)

var (
	tokenValidityCache map[string]time.Time
	tokenUserCache     map[string]string
	tokenStoreMux      sync.Mutex
)

const tokenValidity = 24 * time.Hour

func GenerateToken(username, password string) (string, error) {
	if AuthenticateUser(username, password) {
		token := auth.HashPassword(username)
		tokenStoreMux.Lock()
		tokenValidityCache[token] = time.Now().Add(tokenValidity)
		tokenUserCache[token] = username
		tokenStoreMux.Unlock()
		return token, nil
	}
	return "", fmt.Errorf("invalid credentials")
}

func ValidateToken(username, token string) bool {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	if user, ok := tokenUserCache[token]; ok {
		if user == username {
			if validity, ok := tokenValidityCache[token]; ok {
				if time.Now().Before(validity) {
					return true
				}
			}
		}
	}
	return false
}

func ValidateSingleToken(token string) bool {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	if _, ok := tokenUserCache[token]; ok {
		if validity, ok := tokenValidityCache[token]; ok {
			if time.Now().Before(validity) {
				return true
			}
		}
	}
	return false
}

func InvalidateToken(token string) {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	delete(tokenValidityCache, token)
	delete(tokenUserCache, token)
}

func InvalidateAllTokens() {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	tokenValidityCache = make(map[string]time.Time)
	tokenUserCache = make(map[string]string)
}

func InvalidateExpiredTokens() {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	for token, validity := range tokenValidityCache {
		if time.Now().After(validity) {
			delete(tokenValidityCache, token)
			delete(tokenUserCache, token)
		}
	}
}

func GetTokenValidity(token string) time.Time {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	if validity, ok := tokenValidityCache[token]; ok {
		return validity
	}
	return time.Time{}
}

func GetTokenUser(token string) string {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	if user, ok := tokenUserCache[token]; ok {
		return user
	}
	return ""
}

func GetActiveToken(user string) string {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	for token, u := range tokenUserCache {
		if u == user {
			return token
		}
	}
	return ""
}
