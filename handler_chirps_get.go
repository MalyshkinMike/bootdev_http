package main

import (
	"github.com/MalyshkinMike/bootdev_http/internal/database"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"github.com/google/uuid"
	"sort"
)

func userIdFromRequest(r *http.Request) (uuid.UUID, error) {
	author_id := r.URL.Query().Get("author_id")
	if author_id == "" {
		return uuid.Nil, nil
	}
	userId, err := uuid.Parse(author_id)
	if err != nil {
		return uuid.Nil, err
	}
	return userId, nil
}

func sortingOrderFromRequest(r *http.Request) string {
	sorting_order := r.URL.Query().Get("sort")
	if sorting_order == "" {
		return "asc"
	}
	return sorting_order
}

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	
	var sqlChirps []database.Chirp
	userId, err := userIdFromRequest(r)
	sorting_order := sortingOrderFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error processing author id: %v", err), err)
		return
	}
	if userId == uuid.Nil {
		sqlChirps, err = cfg.dbQueries.ListChirps(r.Context())
	} else {
		sqlChirps, err = cfg.dbQueries.ListChirpsByUser(r.Context(), userId)
	}
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("No chirps found; %v", err), err)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error listing chirps: %v", err), err)
		return
	}
	chirps := toChirps(sqlChirps)
	if sorting_order == "asc" {
		sort.Slice(chirps, func (i, j int) bool { return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)})
	} else {
		sort.Slice(chirps, func (i, j int) bool { return chirps[j].CreatedAt.Before(chirps[i].CreatedAt)})
	}
	respondWithJson(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerChirpGet(w http.ResponseWriter, r *http.Request) {
	chirp_id := r.PathValue("id")
	chirp_uuid, err := uuid.Parse(chirp_id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error converting id to uuid: %v", err), err)
		return
	}
	sqlChirp, err := cfg.dbQueries.GetChirp(r.Context(), chirp_uuid)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Error getting chirp: %v", err), err)
		return
	}
	chirp := toChirp(sqlChirp)
	respondWithJson(w, http.StatusOK, chirp)
}
