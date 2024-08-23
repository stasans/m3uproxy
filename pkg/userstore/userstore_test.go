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
	"testing"
)

func TestGenerateToken(t *testing.T) {
	// Test generating a token for a valid user
	AddUser("john", "password123")
	token, err := GenerateToken("john", "password123")
	if err != nil {
		t.Errorf("Failed to generate token: %v", err)
	}

	// Verify that the generated token is not empty
	if token == "" {
		t.Error("Generated token is empty")
	}

	// Test generating a token for an invalid user
	_, err = GenerateToken("invaliduser", "password123")
	if err == nil {
		t.Error("Expected error when generating token for invalid user")
	}
	DropUsers()
}

func TestAuthenticateUser(t *testing.T) {
	// Test authenticating a valid user
	AddUser("john", "password123")
	validUser := "john"
	validPassword := "password123"
	if !AuthenticateUser(validUser, validPassword) {
		t.Errorf("Failed to authenticate valid user: %s", validUser)
	}

	// Test authenticating an invalid user
	invalidUser := "invaliduser"
	invalidPassword := "password123"
	if AuthenticateUser(invalidUser, invalidPassword) {
		t.Errorf("Authenticated invalid user: %s", invalidUser)
	}

	// Test authenticating a valid user with incorrect password
	incorrectPassword := "incorrectpassword"
	if AuthenticateUser(validUser, incorrectPassword) {
		t.Errorf("Authenticated valid user with incorrect password: %s", validUser)
	}
	DropUsers()
}

func TestChangePassword(t *testing.T) {
	// Test changing the password for an existing user
	AddUser("john", "password123")
	username := "john"
	newPassword := "newpassword123"
	err := ChangePassword(username, newPassword)
	if err != nil {
		t.Errorf("Failed to change password for user: %v", err)
	}

	// Test changing the password for a non-existing user
	nonExistingUser := "nonexistinguser"
	err = ChangePassword(nonExistingUser, newPassword)
	if err == nil {
		t.Errorf("Expected error when changing password for non-existing user: %s", nonExistingUser)
	}
	DropUsers()
}

func TestAddUser(t *testing.T) {
	// Test adding a new user
	username := "newuser"
	password := "password123"
	err := AddUser(username, password)
	if err != nil {
		t.Errorf("Failed to add new user: %v", err)
	}

	// Test adding a user with an existing username
	err = AddUser(username, "newpassword123")
	if err == nil {
		t.Errorf("Expected error when adding user with existing username: %s", username)
	}
	DropUsers()
}

func TestRemoveUser(t *testing.T) {
	// Test removing an existing user
	AddUser("john", "password123")
	username := "john"
	err := RemoveUser(username)
	if err != nil {
		t.Errorf("Failed to remove user: %v", err)
	}

	// Test removing a non-existing user
	nonExistingUser := "nonexistinguser"
	err = RemoveUser(nonExistingUser)
	if err == nil {
		t.Errorf("Expected error when removing non-existing user: %s", nonExistingUser)
	}
	DropUsers()
}

func TestGetUsers(t *testing.T) {
	// Test listing users when there are users in the store
	AddUser("john", "password123")
	users, err := GetUsers()
	if err != nil {
		t.Errorf("Failed to list users: %v", err)
	}

	// Verify that the list of users is not empty
	if len(users) == 0 {
		t.Error("List of users is empty")
	}
	DropUsers()
}

func TestInvalidateToken(t *testing.T) {
	// Test invalidating an existing token
	AddUser("john", "password123")

	token, err := GenerateToken("john", "password123")
	if err != nil {
		t.Errorf("Failed to generate token: %v", err)
	}
	InvalidateToken(token)

	// Verify that the token is no longer valid
	if ValidateToken("john", token) {
		t.Errorf("Token was not invalidated: %s", token)
	}

	// Test invalidating a non-existing token
	nonExistingToken := "nonexistingtoken"
	InvalidateToken(nonExistingToken)
	DropUsers()
}
