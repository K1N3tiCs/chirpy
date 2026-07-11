package database

import (
	"context"

	"github.com/google/uuid"
)

type DB interface {
	CreateChirps(ctx context.Context, arg CreateChirpsParams) (Chirp, error)
	GetChirps(ctx context.Context) ([]Chirp, error)
	GetChirpsByID(ctx context.Context, id uuid.UUID) (Chirp, error)
	GetChirpsByUserID(ctx context.Context, userID uuid.UUID) ([]Chirp, error)
	DeleteChirp(ctx context.Context, id uuid.UUID) (Chirp, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error)
	CreateRefreshToken(ctx context.Context, arg CreateRefreshTokenParams) (RefreshToken, error)
	GetUserFromRefreshToken(ctx context.Context, token string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) (RefreshToken, error)
	AddChirpyRedMembership(ctx context.Context, id uuid.UUID) (User, error)
	Reset(ctx context.Context) error
}
