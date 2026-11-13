package main

import (
	"net/http"
	"github.com/google/uuid"
	"fmt"
)

func (cfg *apiConfig) handlerChirpDelete(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("id")
	userId, err := cfg.authenticateUser(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Error authenticateting: %v", err), err)
		return
	}
	chirpUuid, err := uuid.Parse(chirpId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error converting id to uuid: %v", err), err)
		return
	}
	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpUuid)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Error getting chirp from db: %v", err), err)
		return
	}
	if chirp.UserID != userId {
		respondWithError(w, http.StatusForbidden, fmt.Sprintf("Error trying to delete other user's chirp. Chrip's user id: %v, authenticated user id: %v", chirp.UserID, userId), nil)
		return
	}
	err = cfg.dbQueries.DeleteChirp(r.Context(), chirpUuid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error deleting chirp: %v", err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
