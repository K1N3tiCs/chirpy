package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/K1N3tiCs/chirpy/internal/auth"
)

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
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
		respondWithError(w, http.StatusBadRequest, "User doesn't exist", errors.New("users doesn't exist"))
		return
	}

	passwordMatch, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !passwordMatch {
		respondWithError(w, http.StatusUnauthorized, "Wrong Password", errors.New("enter correct password"))
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
	})
}
