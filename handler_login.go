package main

import (
	"net/http"
	"github.com/MalyshkinMike/bootdev_http/internal/database"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
	"github.com/google/uuid"
	"encoding/json"
	"fmt"
	"time"
)

func (cfg *apiConfig) authenticateUser(header http.Header) (uuid.UUID, error) {
	accessToken, err := auth.GetBearerToken(header)
	if err != nil {
		return uuid.Nil, err
	}
	userId, err := auth.ValidateJWT(accessToken, cfg.secret_string)
	if err != nil {
		return uuid.Nil, err
	}
	return userId, nil
}


func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type loginBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
		// ExpiresInSeconds int `json:"expires_in_seconds"`
	}
	type respondBody struct {
		ID uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Email string `json:"email"`
    Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}
	request_body := loginBody{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request_body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decoding json body: %v", err), err)
		return
	}
	// if request_body.ExpiresInSeconds > 60 {
	// 	request_body.ExpiresInSeconds = 60
	// }
	userSql, err := cfg.dbQueries.GetUserByEmail(r.Context(), request_body.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error fetching user from db: %v", err), err)
		return
	}
	match, err := auth.CheckPasswordHash(request_body.Password, userSql.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error comparing password: %v", err), err)
		return
	}
	if match {
		durationString := fmt.Sprintf("%vs", 60)
		duration, err := time.ParseDuration(durationString)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error parsing duration: %v", err), err)
			return
		}
		token, err := auth.MakeJWT(userSql.ID, cfg.secret_string, duration)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating token: %v", err), err)
			return
		}
		refresh_token := auth.MakeRefreshToken()
		create_token_params := database.CreateTokenParams {
			Token: refresh_token,
			UserID: userSql.ID,
			ExpiresAt: time.Now().AddDate(0, 0, 60),
		}
		_, err = cfg.dbQueries.CreateToken(r.Context(), create_token_params)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating refresh_token in db: %v", err), err)
			return
		}
		userFromSql := respondBody {
			ID: userSql.ID,
			CreatedAt: userSql.CreatedAt,
			UpdatedAt: userSql.UpdatedAt,
			Email: userSql.Email,
			Token: token,
			RefreshToken: refresh_token,
			IsChirpyRed: userSql.IsChirpyRed,
		}
		respondWithJson(w, http.StatusOK, userFromSql)
	} else {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
	}
}
