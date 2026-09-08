package auth

import (
	"testing"
	"regexp"
	"strings"
	"github.com/google/uuid"
	"time"
)

func TestHashPassword(t *testing.T) {
	hashRX, err := regexp.Compile(`^\$argon2id\$v=19\$m=65536,t=1,p=[0-9]{1,4}\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}`)
	if err != nil {
		t.Fatal(err)
	}

	hash1, err := HashPassword("pa$$word")
	if err != nil {
		t.Fatal(err)
	}

	if !hashRX.MatchString(hash1) {
		t.Errorf("hash %q not in correct format", hash1)
	}

	hash2, err := HashPassword("pa$$word")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(hash1, hash2) == 0 {
		t.Error("hashes must be unique")
	}
	
}

func TestCheckPasswordHash(t *testing.T) {
	hash, err := HashPassword("pa$$word")
	if err != nil {
		t.Fatal(err)
	}

	match, err := CheckPasswordHash("pa$$word", hash)
	if err != nil {
		t.Fatal(err)
	}
	
	if !match {
		t.Error("expected password and hash to match")
	}

	match, err = CheckPasswordHash("somEthiNgEl$e", hash)

	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Error("expected password and hash not to match")
	}
}

func TestMakeJWT(t *testing.T) {
	secret1 := "my$Ecret"
	secret2 := "anotHer$ecret"
	userId1 := uuid.New()
	userId2 := uuid.New()
	expiresIn, err := time.ParseDuration("1h")
	if err != nil {
		t.Fatal(err)
	}
	token1, err := MakeJWT(userId1, secret1, expiresIn)
	if err != nil {
		t.Fatal(err)
	}
	token2, err := MakeJWT(userId1, secret1, expiresIn)
	if err != nil {
		t.Fatal(err)
	}
	token3, err := MakeJWT(userId1, secret2, expiresIn)
	if err != nil {
		t.Fatal(err)
	}
	token4, err := MakeJWT(userId2, secret1, expiresIn)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(token1, token2) != 0 {
		t.Error("Expected same tokens of same input to match")
	}

	if strings.Compare(token1, token3) == 0 {
		t.Error("Expected different secrets to generate different tokens")
	}

	if strings.Compare(token1, token4) == 0 {
		t.Error("Expected different user id to generate different tokens")
	}

}

func TestValidateJWT(t *testing.T) {
	secret := "$tronG$Ecret"
	userId := uuid.New()
	expired, err := time.ParseDuration("1ms")
	if err != nil {
		t.Fatal(err)
	}
	active, err := time.ParseDuration("1h")
	if err != nil {
		t.Fatal(err)
	}
	second, err := time.ParseDuration("1s")
	if err != nil {
		t.Fatal(err)
	}
	token_valid, err := MakeJWT(userId, secret, active)
	if err != nil {
		t.Fatal(err)
	}
	token_expired, err := MakeJWT(userId, secret, expired)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(second)
	returnedId, err := ValidateJWT(token_valid, secret)
	if err != nil && err.Error() == "token has invalid claims: token is expired" {
		t.Fatal("Valid token expired")
	} else if err != nil { 
		t.Fatal(err)
	}
	if returnedId != userId {
		t.Error("expected token to not change after validation")
	}
	expiredId, err := ValidateJWT(token_expired, secret)
	if err != nil && err.Error() != "token has invalid claims: token is expired" {
		t.Fatal(err)
	}
	if expiredId == userId {
		t.Error("expected expired token to not return userID")
	}
	
}
