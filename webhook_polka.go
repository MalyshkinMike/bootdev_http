package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
	"errors"
	"github.com/google/uuid"
	"database/sql"
)

func (cfg *apiConfig) webhookPolka(w http.ResponseWriter, r *http.Request) {
	type RequestBody struct {
		Event string `json:"event"`
		Data json.RawMessage `json:"data"`
	}
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Error getting api key: %v", err), err)
		return
	}
	if apiKey != cfg.polka_key {
		respondWithError(w, http.StatusUnauthorized, "Api key not valid", nil)
		return
	}
	var requestBody RequestBody
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&requestBody)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding request body: %v", err), err)
		return
	}

	switch requestBody.Event {
	case "user.upgraded":
		type UpgradeUserData struct {
			UserId uuid.UUID `json:"user_id"`
		}
		var upgradeUserData UpgradeUserData
		err = json.Unmarshal(requestBody.Data, &upgradeUserData)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding data: %v", err), err)
			return
		}
		_, err = cfg.dbQueries.UpgradeUserToChirpyRed(r.Context(), upgradeUserData.UserId)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, fmt.Sprintf("Error upgrading User: %v", err), err)
			return
		} else if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error upgrading User: %v", err), err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusNoContent)
		return
	}
}
