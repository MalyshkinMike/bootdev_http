package main

import (
	"github.com/google/uuid"
	"time"
	"net/http"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
	"github.com/MalyshkinMike/bootdev_http/internal/database"
	"fmt"
	"encoding/json"
)

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}


func toUser(row database.User) User {
	return User {
		ID: row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Email: row.Email,
		IsChirpyRed: row.IsChirpyRed,
	}
}


func (cfg *apiConfig) handlerUserCreate(w http.ResponseWriter, r *http.Request) {
	type userBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	user := userBody{}
	err := decoder.Decode(&user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding parameters: %s", err), err)
		return
	}
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error hashing password: %v", err), err)
		return
	}
	userParams := database.CreateUserParams { 
		Email: user.Email, 
		HashedPassword: hashedPassword, 
	}
	usr, err := cfg.dbQueries.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating user: %s", err), err)
		return
	}
	userForSend := toUser(usr)
	respondWithJson(w, http.StatusCreated, userForSend)	
}
