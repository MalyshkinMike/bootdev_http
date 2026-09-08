package auth

import (
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"errors"
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
