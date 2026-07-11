package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/K1N3tiCs/chirpy/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerAddChirpyRedMembership(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Failed To Get API Key", err)
		return
	}

	if apiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "Bad API Key", errors.New("bad api key"))
		return
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to decode the body", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusNoContent, "")
		return
	}

	_, err = cfg.db.AddChirpyRedMembership(r.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "failed to fetch the user", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, "")
}
