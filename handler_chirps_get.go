package main

import (
	"net/http"
	"sort"

	"github.com/google/uuid"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

func (cfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "" {
		sortOrder = string(SortAsc)
	}
	authorID := r.URL.Query().Get("author_id")
	if authorID != "" {
		authorID, err := uuid.Parse(authorID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "provide proper author id", err)
			return
		}

		dbChirps, err := cfg.db.GetChirpsByUserID(r.Context(), authorID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "there are no chirps for the requested author", err)
			return
		}

		chirps := []Chirp{}
		for _, dbChirp := range dbChirps {
			chirps = append(chirps, Chirp{
				ID:        dbChirp.ID,
				CreatedAt: dbChirp.CreatedAt,
				UpdatedAt: dbChirp.UpdatedAt,
				UserID:    dbChirp.UserID,
				Body:      dbChirp.Body,
			})
		}

		if sortOrder == string(SortDesc) {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
			})
		}

		respondWithJSON(w, http.StatusOK, chirps)
		return
	}

	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
		return
	}

	if sortOrder == string(SortDesc) {
		sort.Slice(dbChirps, func(i, j int) bool {
			return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt)
		})
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID:    dbChirp.UserID,
			Body:      dbChirp.Body,
		})
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	dbChirp, err := cfg.db.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get chirp", err)
		return
	}

	respondWithJSON(w, http.StatusOK, Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	})
}
