package main

import (
	"net/http"
	"time"

	"github.com/K1N3tiCs/chirpy/internal/auth"
)

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "provide a valid refresh token", err)
		return
	}

	ref_token, err := cfg.db.GetUserFromRefreshToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "you are not authorized", err)
		return
	}

	new_access_token, err := auth.MakeJWT(ref_token.UserID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to generate new token", err)
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: new_access_token,
	})
}
