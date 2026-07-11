package main

import (
	"errors"
	"net/http"

	"github.com/K1N3tiCs/chirpy/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerChirpDelete(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	access_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "provide access token", err)
		return
	}

	userID, err := auth.ValidateJWT(access_token, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "provide a valid access_token", err)
		return
	}

	chirp, err := cfg.db.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Unable to find chirp", err)
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "chirp doesn't belong to you", errors.New("chirp do not belong to you"))
		return
	}

	_, err = cfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Unable to find chirp", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, "")
}
