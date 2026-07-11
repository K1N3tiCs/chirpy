package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/K1N3tiCs/chirpy/internal/auth"
	"github.com/K1N3tiCs/chirpy/internal/database"
)

func (cfg *apiConfig) handlerUserUpdate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't decode the body", err)
		return
	}
	if strings.Trim(params.Email, " ") == "" {
		respondWithError(w, http.StatusBadRequest, "email cannot be empty", errors.New("emails cannot be empty"))
		return
	}
	if strings.Trim(params.Password, " ") == "" {
		respondWithError(w, http.StatusBadRequest, "email cannot be empty", errors.New("password cannot be empty"))
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to hash password", err)
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

	update_user_params := database.UpdateUserParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: hashed_password,
	}

	user, err := cfg.db.UpdateUser(r.Context(), update_user_params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update user", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	})
}
