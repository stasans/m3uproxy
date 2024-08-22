package userstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Users struct {
	Users []User `json:"users"`
}

var (
	usersFilePath      = "users.json"
	tokenValidityCache = make(map[string]time.Time)
	tokenUserCache     = make(map[string]string)
	tokenStoreMux      sync.Mutex
	userStoreMux       sync.Mutex
)

const (
	tokenValidity = 24 * time.Hour
)

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func loadUsers(filePath string) (*Users, error) {
	userStoreMux.Lock()
	defer userStoreMux.Unlock()
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	content := string(body)
	var users Users
	if err := json.Unmarshal([]byte(content), &users); err != nil {
		return nil, err
	}
	return &users, nil
}

func SetUsersFilePath(filePath string) {
	usersFilePath = filePath
}

func AuthenticateUser(username, password string) bool {
	users, err := loadUsers(usersFilePath)
	if err != nil {
		return false
	}
	hashedPassword := hashPassword(password)
	for _, user := range users.Users {
		if user.Username == username && user.Password == hashedPassword {
			return true
		}
	}
	return false
}

func GenerateToken(username, password string) (string, error) {
	if !AuthenticateUser(username, password) {
		return "", fmt.Errorf("invalid credentials")
	}
	token := hex.EncodeToString([]byte(fmt.Sprintf("%x", sha256.Sum256([]byte(time.Now().String())))))
	tokenStoreMux.Lock()
	tokenValidityCache[token] = time.Now().Add(tokenValidity)
	tokenUserCache[token] = username
	tokenStoreMux.Unlock()
	InvalidateExpiredTokens()
	return token, nil
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

func ValidateToken(username, token string) bool {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	expiry, exists := tokenValidityCache[token]
	if time.Now().After(expiry) {
		delete(tokenValidityCache, token)
		delete(tokenUserCache, token)
		return false
	}
	if !exists {
		return false
	}
	if tokenUserCache[token] != username {
		return false
	}
	return true
}

func ValidateSingleToken(token string) bool {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	expiry, exists := tokenValidityCache[token]
	if time.Now().After(expiry) {
		delete(tokenValidityCache, token)
		delete(tokenUserCache, token)
		return false
	}
	if !exists {
		return false
	}
	return true
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
	for token, expiry := range tokenValidityCache {
		if time.Now().After(expiry) {
			delete(tokenValidityCache, token)
			delete(tokenUserCache, token)
		}
	}
}

func GetTokenValidity(token string) time.Time {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	return tokenValidityCache[token]
}

func GetTokenUser(token string) string {
	tokenStoreMux.Lock()
	defer tokenStoreMux.Unlock()
	return tokenUserCache[token]
}

func AddUser(username, password string) error {
	users, err := loadUsers(usersFilePath)
	if err != nil {
		return err
	}
	// check if user already exists
	userStoreMux.Lock()
	defer userStoreMux.Unlock()
	for _, user := range users.Users {
		if user.Username == username {
			return fmt.Errorf("user already exists")
		}
	}
	users.Users = append(users.Users, User{Username: username, Password: hashPassword(password)})
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.Create(usersFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func RemoveUser(username string) error {
	users, err := loadUsers(usersFilePath)
	if err != nil {
		return err
	}
	userStoreMux.Lock()
	defer userStoreMux.Unlock()
	for i, user := range users.Users {
		if user.Username == username {
			users.Users = append(users.Users[:i], users.Users[i+1:]...)
			data, err := json.MarshalIndent(users, "", "  ")
			if err != nil {
				return err
			}
			file, err := os.Create(usersFilePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = file.Write(data)
			return err
		}
	}
	return fmt.Errorf("user not found")
}

func GetUsers() ([]string, error) {
	users, err := loadUsers(usersFilePath)
	if err != nil {
		return nil, err
	}
	var usernames []string
	for _, user := range users.Users {
		usernames = append(usernames, user.Username)
	}
	return usernames, nil
}

func ChangePassword(username, password string) error {
	users, err := loadUsers(usersFilePath)
	if err != nil {
		return err
	}
	userStoreMux.Lock()
	defer userStoreMux.Unlock()
	for i, user := range users.Users {
		if user.Username == username {
			users.Users[i].Password = hashPassword(password)
			data, err := json.MarshalIndent(users, "", "  ")
			if err != nil {
				return err
			}
			file, err := os.Create(usersFilePath)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = file.Write(data)
			return err
		}
	}
	return fmt.Errorf("user not found")
}

func DropUsers() error {
	userStoreMux.Lock()
	defer userStoreMux.Unlock()
	file, err := os.Create(usersFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte("{\"users\":[]}"))
	return err
}
