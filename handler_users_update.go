package main

import (
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/MalyshkinMike/bootdev_http/internal/database"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
)

func (cfg *apiConfig) handlerUserUpdate(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	userId, err := cfg.authenticateUser(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Error authenticateting: %v", err), err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	requestData := requestBody{}
	err = decoder.Decode(&requestData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %v", err), err)
		return
	}
	hashedPassword, err := auth.HashPassword(requestData.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error hashing new password: %v", err), err)
		return
	}
	params := database.UpdateUserParams {
		ID: userId,
		Email: requestData.Email,
		HashedPassword: hashedPassword,
	}
	usrSql, err := cfg.dbQueries.UpdateUser(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error updating user: %v", err), err)
		return
	}
	usrToSend := toUser(usrSql)
	respondWithJson(w, http.StatusOK, usrToSend)
}
