package main

import (
	"net/http"

	"github.com/K1N3tiCs/chirpy/internal/auth"
)

func (cfg *apiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "provide a valid refresh token", err)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to revoke the token", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, "")
}
