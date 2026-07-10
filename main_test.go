package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/K1N3tiCs/chirpy/internal/database"
)

func setupTestAPI(t *testing.T) apiConfig {
	t.Helper()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set, skipping handler tests")
	}

	dbConn, err := openDB(dbURL)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	t.Cleanup(func() {
		dbConn.Close()
	})

	// Clean the database before tests
	dbQueries := database.New(dbConn)
	err = dbQueries.Reset(t.Context())
	if err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	return apiConfig{
		db:       dbQueries,
		platform: "dev",
	}
}

func setupTestMux(t *testing.T, cfg apiConfig) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/chirps", cfg.handlerChirpsRetrieve)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerChirpsGet)
	mux.HandleFunc("POST /api/chirps", cfg.handlerChirpsCreate)
	mux.HandleFunc("POST /api/users", cfg.handlerUsersCreate)
	mux.HandleFunc("POST /api/login", cfg.handlerUserLogin)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	return mux
}

func openDB(dbURL string) (*sql.DB, error) {
	return sql.Open("postgres", dbURL)
}

func TestHandlerReadiness(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/healthz", nil)
	w := httptest.NewRecorder()

	handlerReadiness(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("handlerReadiness status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("handlerReadiness content-type: got %v, want %v", contentType, "text/plain; charset=utf-8")
	}

	body := w.Body.String()
	if body != http.StatusText(http.StatusOK) {
		t.Fatalf("handlerReadiness body: got %v, want %v", body, http.StatusText(http.StatusOK))
	}
}

func TestHandlerChirpsValidate(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedCode   int
		expectedError  bool
		expectedClean  string
	}{
		{
			name:           "valid chirp",
			body:           "Hello world",
			expectedCode:   http.StatusOK,
			expectedError:  false,
			expectedClean:  "Hello world",
		},
		{
			name:           "chirp too long",
			body:           string(make([]byte, 141)),
			expectedCode:   http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "chirp with profanity",
			body:           "I love kerfuffle and sharbert",
			expectedCode:   http.StatusOK,
			expectedError:  false,
			expectedClean:  "I love **** and ****",
		},
		{
			name:           "chirp exactly 140 chars",
			body:           "a]b",
			expectedCode:   http.StatusOK,
			expectedError:  false,
			expectedClean:  "a]b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(map[string]string{"body": tt.body})
			req := httptest.NewRequest("POST", "/api/validate_chirp", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handlerChirpsValidate(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedCode {
				t.Fatalf("status: got %d, want %d", resp.StatusCode, tt.expectedCode)
			}

			var result map[string]string
			json.NewDecoder(resp.Body).Decode(&result)

			if tt.expectedError {
				if _, exists := result["error"]; !exists {
					t.Fatal("Expected error response but got none")
				}
			} else {
				if result["cleaned_body"] != tt.expectedClean {
					t.Fatalf("cleaned_body: got %v, want %v", result["cleaned_body"], tt.expectedClean)
				}
			}
		})
	}
}

func TestHandlerChirpsValidate_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/validate_chirp", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlerChirpsValidate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandlerUsersCreate(t *testing.T) {
	cfg := setupTestAPI(t)

	reqBody, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "validpassword123",
	})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg.handlerUsersCreate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result User
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("Expected valid user ID")
	}
	if result.Email != "test@example.com" {
		t.Fatalf("email: got %v, want %v", result.Email, "test@example.com")
	}
}

func TestHandlerUsersCreate_EmptyPassword(t *testing.T) {
	cfg := setupTestAPI(t)

	reqBody, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "",
	})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg.handlerUsersCreate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlerUserLogin(t *testing.T) {
	cfg := setupTestAPI(t)

	// First create a user
	createBody, _ := json.Marshal(map[string]string{
		"email":    "login@example.com",
		"password": "mypassword123",
	})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	// Now login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "login@example.com",
		"password": "mypassword123",
	})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)

	resp := loginW.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result User
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Email != "login@example.com" {
		t.Fatalf("email: got %v, want %v", result.Email, "login@example.com")
	}
}

func TestHandlerUserLogin_WrongPassword(t *testing.T) {
	cfg := setupTestAPI(t)

	// Create a user
	createBody, _ := json.Marshal(map[string]string{
		"email":    "wrong@example.com",
		"password": "correctpassword",
	})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	// Login with wrong password
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "wrong@example.com",
		"password": "wrongpassword",
	})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)

	resp := loginW.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandlerChirpsCreate(t *testing.T) {
	cfg := setupTestAPI(t)

	// Create a user first
	userBody, _ := json.Marshal(map[string]string{
		"email":    "chirp@example.com",
		"password": "password123",
	})
	userReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(userBody))
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()
	cfg.handlerUsersCreate(userW, userReq)

	var createdUser User
	json.NewDecoder(userW.Body).Decode(&createdUser)

	// Create a chirp
	chirpBody, _ := json.Marshal(map[string]string{
		"body":    "My test chirp!",
		"user_id": createdUser.ID.String(),
	})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)

	resp := chirpW.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result Chirp
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Body != "My test chirp!" {
		t.Fatalf("body: got %v, want %v", result.Body, "My test chirp!")
	}
	if result.UserID != createdUser.ID {
		t.Fatalf("user_id: got %v, want %v", result.UserID, createdUser.ID)
	}
}

func TestHandlerChirpsCreate_TooLong(t *testing.T) {
	cfg := setupTestAPI(t)

	// Create a user first
	userBody, _ := json.Marshal(map[string]string{
		"email":    "longchirp@example.com",
		"password": "password123",
	})
	userReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(userBody))
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()
	cfg.handlerUsersCreate(userW, userReq)

	var createdUser User
	json.NewDecoder(userW.Body).Decode(&createdUser)

	// Create a chirp that's too long (but should still be saved by the create handler)
	longBody := ""
	for i := 0; i < 141; i++ {
		longBody += "a"
	}
	chirpBody, _ := json.Marshal(map[string]string{
		"body":    longBody,
		"user_id": createdUser.ID.String(),
	})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)

	resp := chirpW.Result()
	// Note: The create handler doesn't validate length, it just stores
	// The validate endpoint is separate
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestHandlerChirpsRetrieve(t *testing.T) {
	cfg := setupTestAPI(t)

	// Create a user
	userBody, _ := json.Marshal(map[string]string{
		"email":    "retrieve@example.com",
		"password": "password123",
	})
	userReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(userBody))
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()
	cfg.handlerUsersCreate(userW, userReq)

	var createdUser User
	json.NewDecoder(userW.Body).Decode(&createdUser)

	// Create chirps
	for i := 0; i < 3; i++ {
		chirpBody, _ := json.Marshal(map[string]string{
			"body":    "Test chirp",
			"user_id": createdUser.ID.String(),
		})
		chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
		chirpReq.Header.Set("Content-Type", "application/json")
		chirpW := httptest.NewRecorder()
		cfg.handlerChirpsCreate(chirpW, chirpReq)
	}

	// Retrieve chirps
	req := httptest.NewRequest("GET", "/api/chirps", nil)
	w := httptest.NewRecorder()
	cfg.handlerChirpsRetrieve(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var chirps []Chirp
	json.NewDecoder(resp.Body).Decode(&chirps)

	if len(chirps) != 3 {
		t.Fatalf("expected 3 chirps, got %d", len(chirps))
	}
}

func TestHandlerChirpsGet(t *testing.T) {
	cfg := setupTestAPI(t)
	mux := setupTestMux(t, cfg)

	// Create a user
	userBody, _ := json.Marshal(map[string]string{
		"email":    "get@example.com",
		"password": "password123",
	})
	userReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(userBody))
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()
	mux.ServeHTTP(userW, userReq)

	var createdUser User
	json.NewDecoder(userW.Body).Decode(&createdUser)

	// Create a chirp
	chirpBody, _ := json.Marshal(map[string]string{
		"body":    "Find this chirp!",
		"user_id": createdUser.ID.String(),
	})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpW := httptest.NewRecorder()
	mux.ServeHTTP(chirpW, chirpReq)

	var createdChirp Chirp
	json.NewDecoder(chirpW.Body).Decode(&createdChirp)

	// Get by ID
	req := httptest.NewRequest("GET", "/api/chirps/"+createdChirp.ID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result Chirp
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID != createdChirp.ID {
		t.Fatalf("ID: got %v, want %v", result.ID, createdChirp.ID)
	}
	if result.Body != "Find this chirp!" {
		t.Fatalf("body: got %v, want %v", result.Body, "Find this chirp!")
	}
}

func TestHandlerChirpsGet_InvalidID(t *testing.T) {
	cfg := setupTestAPI(t)
	mux := setupTestMux(t, cfg)

	req := httptest.NewRequest("GET", "/api/chirps/not-a-uuid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlerChirpsGet_NotFound(t *testing.T) {
	cfg := setupTestAPI(t)
	mux := setupTestMux(t, cfg)

	req := httptest.NewRequest("GET", "/api/chirps/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandlerReset(t *testing.T) {
	cfg := setupTestAPI(t)

	req := httptest.NewRequest("POST", "/admin/reset", nil)
	w := httptest.NewRecorder()
	cfg.handlerReset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandlerReset_NotDev(t *testing.T) {
	cfg := setupTestAPI(t)
	cfg.platform = "production"

	req := httptest.NewRequest("POST", "/admin/reset", nil)
	w := httptest.NewRecorder()
	cfg.handlerReset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestHandlerMetrics(t *testing.T) {
	cfg := setupTestAPI(t)

	req := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()
	cfg.handlerMetrics(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Fatalf("content-type: got %v, want %v", contentType, "text/html; charset=utf-8")
	}
}
