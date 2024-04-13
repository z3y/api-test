package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateRandomAuthKey() (string, error) {
	// Create a byte slice of the specified length
	key := make([]byte, 256)

	// Read random bytes from the cryptographic random number generator
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	// Encode the random bytes to base64
	keyBase64 := base64.URLEncoding.EncodeToString(key)

	return keyBase64, nil
}

func CreateJwt(userId string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userId,
		"exp": time.Now().UTC().Add(time.Hour * 24).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(secretKey)

	return tokenString, err
}

func ValidateAndParseJwt(tokenString string) (jwt.MapClaims, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok {
		return claims, nil
	}
	return nil, err
}
