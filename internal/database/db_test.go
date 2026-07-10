package database

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *Queries {
	t.Helper()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set, skipping database tests")
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	t.Cleanup(func() {
		dbConn.Close()
	})

	return New(dbConn)
}

func cleanDB(t *testing.T, q *Queries) {
	t.Helper()
	err := q.Reset(context.Background())
	if err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	params := CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "hashedpassword123",
	}

	user, err := q.CreateUser(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Fatal("CreateUser should return a valid UUID")
	}
	if user.Email != "test@example.com" {
		t.Fatalf("CreateUser email mismatch: got %v, want %v", user.Email, "test@example.com")
	}
	if user.HashedPassword != "hashedpassword123" {
		t.Fatalf("CreateUser hashed password mismatch: got %v, want %v", user.HashedPassword, "hashedpassword123")
	}
	if user.CreatedAt.IsZero() {
		t.Fatal("CreateUser should set created_at")
	}
	if user.UpdatedAt.IsZero() {
		t.Fatal("CreateUser should set updated_at")
	}
}

func TestGetUserByEmail(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	// Create a user first
	createParams := CreateUserParams{
		Email:          "lookup@example.com",
		HashedPassword: "hashedpassword456",
	}
	createdUser, err := q.CreateUser(context.Background(), createParams)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	// Look up by email
	user, err := q.GetUserByEmail(context.Background(), "lookup@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail returned error: %v", err)
	}

	if user.ID != createdUser.ID {
		t.Fatalf("GetUserByEmail ID mismatch: got %v, want %v", user.ID, createdUser.ID)
	}
	if user.Email != "lookup@example.com" {
		t.Fatalf("GetUserByEmail email mismatch: got %v, want %v", user.Email, "lookup@example.com")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	_, err := q.GetUserByEmail(context.Background(), "nonexistent@example.com")
	if err == nil {
		t.Fatal("GetUserByEmail should return error for non-existent user")
	}
}

func TestCreateChirps(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	// Create a user first (required for chirps FK)
	userParams := CreateUserParams{
		Email:          "chirper@example.com",
		HashedPassword: "hashedpassword789",
	}
	user, err := q.CreateUser(context.Background(), userParams)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	chirpParams := CreateChirpsParams{
		Body:   "Hello, this is my first chirp!",
		UserID: user.ID,
	}

	chirp, err := q.CreateChirps(context.Background(), chirpParams)
	if err != nil {
		t.Fatalf("CreateChirps returned error: %v", err)
	}

	if chirp.ID == uuid.Nil {
		t.Fatal("CreateChirps should return a valid UUID")
	}
	if chirp.Body != "Hello, this is my first chirp!" {
		t.Fatalf("CreateChirps body mismatch: got %v, want %v", chirp.Body, "Hello, this is my first chirp!")
	}
	if chirp.UserID != user.ID {
		t.Fatalf("CreateChirps user_id mismatch: got %v, want %v", chirp.UserID, user.ID)
	}
}

func TestGetChirps(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	// Create a user
	userParams := CreateUserParams{
		Email:          "multi@example.com",
		HashedPassword: "hashedpassword",
	}
	user, err := q.CreateUser(context.Background(), userParams)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	// Create multiple chirps
	for i := 0; i < 3; i++ {
		chirpParams := CreateChirpsParams{
			Body:   "Chirp number",
			UserID: user.ID,
		}
		_, err := q.CreateChirps(context.Background(), chirpParams)
		if err != nil {
			t.Fatalf("CreateChirps returned error: %v", err)
		}
	}

	chirps, err := q.GetChirps(context.Background())
	if err != nil {
		t.Fatalf("GetChirps returned error: %v", err)
	}

	if len(chirps) != 3 {
		t.Fatalf("GetChirps should return 3 chirps, got %d", len(chirps))
	}
}

func TestGetChirpsByID(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	// Create a user
	userParams := CreateUserParams{
		Email:          "single@example.com",
		HashedPassword: "hashedpassword",
	}
	user, err := q.CreateUser(context.Background(), userParams)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	// Create a chirp
	chirpParams := CreateChirpsParams{
		Body:   "Find me by ID!",
		UserID: user.ID,
	}
	createdChirp, err := q.CreateChirps(context.Background(), chirpParams)
	if err != nil {
		t.Fatalf("CreateChirps returned error: %v", err)
	}

	// Get by ID
	chirp, err := q.GetChirpsByID(context.Background(), createdChirp.ID)
	if err != nil {
		t.Fatalf("GetChirpsByID returned error: %v", err)
	}

	if chirp.ID != createdChirp.ID {
		t.Fatalf("GetChirpsByID ID mismatch: got %v, want %v", chirp.ID, createdChirp.ID)
	}
	if chirp.Body != "Find me by ID!" {
		t.Fatalf("GetChirpsByID body mismatch: got %v, want %v", chirp.Body, "Find me by ID!")
	}
}

func TestGetChirpsByID_NotFound(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	fakeID := uuid.New()
	_, err := q.GetChirpsByID(context.Background(), fakeID)
	if err == nil {
		t.Fatal("GetChirpsByID should return error for non-existent chirp")
	}
}

func TestReset(t *testing.T) {
	q := setupTestDB(t)
	cleanDB(t, q)

	// Create some data
	userParams := CreateUserParams{
		Email:          "reset@example.com",
		HashedPassword: "hashedpassword",
	}
	user, err := q.CreateUser(context.Background(), userParams)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	chirpParams := CreateChirpsParams{
		Body:   "This should be deleted",
		UserID: user.ID,
	}
	_, err = q.CreateChirps(context.Background(), chirpParams)
	if err != nil {
		t.Fatalf("CreateChirps returned error: %v", err)
	}

	// Reset
	err = q.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}

	// Verify chirps are gone (cascade delete)
	chirps, err := q.GetChirps(context.Background())
	if err != nil {
		t.Fatalf("GetChirps returned error after reset: %v", err)
	}
	if len(chirps) != 0 {
		t.Fatalf("GetChirps should return 0 chirps after reset, got %d", len(chirps))
	}
}
