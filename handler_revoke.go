package main

import (
	"net/http"
	"fmt"
	"github.com/MalyshkinMike/bootdev_http/internal/auth"
)

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	//revoke refresh token by setting revoked_at = currentTimestamp
	err = cfg.dbQueries.RevokeToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error revoking refresh_token: %v", err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
