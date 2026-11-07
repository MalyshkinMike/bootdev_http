package auth

import (
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"errors"
	"strings"
	"net/http"
	"crypto/rand"
	"encoding/hex"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims {
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject: userID.String(),
	})

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(tokem *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	} else if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
			id, err := uuid.Parse(claims.Subject)
			if err != nil {
				return uuid.UUID{}, nil
			}
			return id, nil
	} else {
		return uuid.UUID{}, errors.New("Unknown claims type")
	}
}


func GetBearerToken(headers http.Header) (string, error) {
	auth_header := headers.Get("Authorization")
	if auth_header == "" {
		return "", errors.New("No Authorization header found")
	}
	token := strings.Split(auth_header, " ")
	if len(token) != 2 {
		return "", errors.New("Got Invalid Authorization header content")
	}
	return token[1], nil
}

func GetAPIKey(headers http.Header) (string, error) {
	auth_header := headers.Get("Authorization")
	if auth_header == "" {
		return "", errors.New("No Authorization header found")
	}
	key := strings.Split(auth_header, " ")
	if len(key) != 2 {
		return "", errors.New("Got Invalid Authorization header content")
	}
	if key[0] != "ApiKey" {
		return "", errors.New("Authorization contains not an api key")
	}
	return key[1], nil
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	encoded_key := hex.EncodeToString(key)
	return encoded_key
}
