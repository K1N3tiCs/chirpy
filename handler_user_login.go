package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/K1N3tiCs/chirpy/internal/auth"
	"github.com/K1N3tiCs/chirpy/internal/database"
)

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if len(params.Password) == 0 {
		respondWithError(w, http.StatusBadRequest, "Please provide the password", errors.New("provide a valid password"))
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "User doesn't exist", err)
		return
	}

	passwordMatch, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !passwordMatch {
		respondWithError(w, http.StatusUnauthorized, "Wrong Password", err)
		return
	}

	expirationTime := time.Hour

	token, err := auth.MakeJWT(user.ID, cfg.secret, expirationTime)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't generate token", err)
		return
	}

	refresh_token := auth.MakeRefreshToken()
	refresh_token_params := database.CreateRefreshTokenParams{
		Token:  refresh_token,
		UserID: user.ID,
	}

	ref_token, err := cfg.db.CreateRefreshToken(r.Context(), refresh_token_params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could't generate refresh token", err)
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
		Token:        token,
		RefreshToken: ref_token.Token,
	})
}
