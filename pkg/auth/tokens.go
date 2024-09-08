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
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func createJWT(userID string) (string, error) {
	// Define token expiration time
	expirationTime := time.Now().Add(time.Hour * time.Duration(authConfig.ExpirationTime)) // 1 hour expiry

	// Define the claims
	claims := jwt.MapClaims{
		"sub":  userID,                // Subject or user ID
		"exp":  expirationTime.Unix(), // Expiration time
		"iat":  time.Now().Unix(),     // Issued at time
		"role": "user",                // Custom claim (e.g., user role)
	}

	// Create a new token using the HS256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with your secret key
	tokenString, err := token.SignedString([]byte(authConfig.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func verifyJWT(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(authConfig.SecretKey), nil
	})

	// Handle errors
	if err != nil {
		return nil, err
	}

	// Extract claims if the token is valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}

func CreateToken(userId, password string) (string, error) {
	if CheckCredentials(userId, password) {
		return createJWT(userId)
	}
	return "", fmt.Errorf("invalid credentials")
}

func VerifyUserToken(userId, token string) bool {
	claims, err := verifyJWT(token)
	if err != nil {
		return false
	}
	if sub, ok := claims["sub"].(string); ok {
		if sub == userId {
			return true
		}
	}
	return false
}

func VerifyToken(token string) bool {
	_, err := verifyJWT(token)
	return err == nil
}
