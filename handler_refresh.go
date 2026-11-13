package main

import (
	"net/http"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
	"fmt"
	"time"
)
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type responceBody struct {
		Token string `json:"token"`
	}
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting bearer token: %v", err), err)
		return
	}
	sqltoken, err := cfg.dbQueries.GetToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Token not found: %v", err), err)
		return
	}
	// check expiration
	if sqltoken.ExpiresAt.Compare(time.Now()) <= 0 {
		respondWithError(w, http.StatusUnauthorized, "Token expired", nil)
		return
	}
	// check for revoking
	if sqltoken.RevokedAt.Valid && sqltoken.RevokedAt.Time.Compare(time.Now()) >= 0 {
		respondWithError(w, http.StatusUnauthorized, "Token revoked", nil)
		return
	}
	userId := sqltoken.UserID
	hour, err := time.ParseDuration("1h")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error parsing duration: %v", err), err)
		return
	}
	JWTToken, err := auth.MakeJWT(userId, cfg.secret_string, hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error making access token: %v", err), err)
		return
	}
	responce := responceBody{
		Token: JWTToken,
	}
	respondWithJson(w, http.StatusOK, responce)
}
