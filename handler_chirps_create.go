package main

import (
	"net/http"
	"fmt"
	"github.com/MalyshkinMike/bootdev_http/internal/database"
	"github.com/google/uuid"
	"time"
	"encoding/json"
)

type Chirp struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func toChirp(row database.Chirp) Chirp {
	return Chirp {
		ID: row.ID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Body: row.Body,
		UserID: row.UserID,
	}
}

func toChirps(rows []database.Chirp) []Chirp {
	chirps := make([]Chirp, len(rows))
	for i, row := range rows {
		chirps[i] = toChirp(row)
	}
	return chirps
}

func (cfg *apiConfig) handlerChirpCreate(w http.ResponseWriter, r *http.Request) {
	type chirpBody struct {
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	userId, err := cfg.authenticateUser(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Error authenticateting: %v", err), err)
		return
	}

	
	decoder := json.NewDecoder(r.Body)
	chirp := chirpBody{}
	err = decoder.Decode(&chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding json: %v", err), err)
		return
	}
	if !validateChirp(chirp.Body) {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Chirp is not valid"), nil)
		return
	}
	params := database.CreateChirpParams { Body: chirp.Body, UserID: userId }
	chrp, err := cfg.dbQueries.CreateChirp(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating chirp: %v", err), err)
		return
	}
	chirpToSend := toChirp(chrp)
	
	respondWithJson(w, http.StatusCreated, chirpToSend)
}
