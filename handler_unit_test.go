package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"database/sql"

	"github.com/K1N3tiCs/chirpy/internal/database"
	"github.com/google/uuid"
)

// mockDB is an in-memory mock implementing database.DB
type mockDB struct {
	mu       sync.Mutex
	users    map[string]database.User // keyed by email
	chirps   map[uuid.UUID]database.Chirp
	refTokens map[string]database.RefreshToken
	nextTime time.Time
}

func newMockDB() *mockDB {
	return &mockDB{
		users:     make(map[string]database.User),
		chirps:    make(map[uuid.UUID]database.Chirp),
		refTokens: make(map[string]database.RefreshToken),
		nextTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (m *mockDB) tick() time.Time {
	m.nextTime = m.nextTime.Add(time.Second)
	return m.nextTime
}

func (m *mockDB) now() time.Time {
	return m.nextTime
}

func (m *mockDB) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.tick()
	u := database.User{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          arg.Email,
		HashedPassword: arg.HashedPassword,
		IsChirpyRed:    false,
	}
	m.users[arg.Email] = u
	return u, nil
}

func (m *mockDB) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return database.User{}, errors.New("user not found")
	}
	return u, nil
}

func (m *mockDB) UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for email, u := range m.users {
		if u.ID == arg.ID {
			updated := u
			updated.Email = arg.Email
			updated.HashedPassword = arg.HashedPassword
			updated.UpdatedAt = m.tick()
			delete(m.users, email)
			m.users[arg.Email] = updated
			return updated, nil
		}
	}
	return database.User{}, errors.New("user not found")
}

func (m *mockDB) CreateChirps(ctx context.Context, arg database.CreateChirpsParams) (database.Chirp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.tick()
	c := database.Chirp{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      arg.Body,
		UserID:    arg.UserID,
	}
	m.chirps[c.ID] = c
	return c, nil
}

func (m *mockDB) GetChirps(ctx context.Context) ([]database.Chirp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]database.Chirp, 0, len(m.chirps))
	for _, c := range m.chirps {
		result = append(result, c)
	}
	return result, nil
}

func (m *mockDB) GetChirpsByID(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.chirps[id]
	if !ok {
		return database.Chirp{}, errors.New("chirp not found")
	}
	return c, nil
}

func (m *mockDB) GetChirpsByUserID(ctx context.Context, userID uuid.UUID) ([]database.Chirp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []database.Chirp
	for _, c := range m.chirps {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockDB) DeleteChirp(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.chirps[id]
	if !ok {
		return database.Chirp{}, errors.New("chirp not found")
	}
	delete(m.chirps, id)
	return c, nil
}

func (m *mockDB) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.tick()
	rt := database.RefreshToken{
		Token:     arg.Token,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    arg.UserID,
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	}
	m.refTokens[arg.Token] = rt
	return rt, nil
}

func (m *mockDB) GetUserFromRefreshToken(ctx context.Context, token string) (database.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rt, ok := m.refTokens[token]
	if !ok {
		return database.RefreshToken{}, errors.New("refresh token not found")
	}
	if m.now().After(rt.ExpiresAt) {
		return database.RefreshToken{}, errors.New("token expired")
	}
	if rt.RevokedAt.Valid {
		return database.RefreshToken{}, errors.New("token revoked")
	}
	return rt, nil
}

func (m *mockDB) RevokeRefreshToken(ctx context.Context, token string) (database.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rt, ok := m.refTokens[token]
	if !ok {
		return database.RefreshToken{}, errors.New("refresh token not found")
	}
	now := m.tick()
	rt.RevokedAt = sql.NullTime{Time: now, Valid: true}
	rt.UpdatedAt = now
	m.refTokens[token] = rt
	return rt, nil
}

func (m *mockDB) AddChirpyRedMembership(ctx context.Context, id uuid.UUID) (database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for email, u := range m.users {
		if u.ID == id {
			u.IsChirpyRed = true
			u.UpdatedAt = m.tick()
			m.users[email] = u
			return u, nil
		}
	}
	return database.User{}, errors.New("user not found")
}

func (m *mockDB) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]database.User)
	m.chirps = make(map[uuid.UUID]database.Chirp)
	m.refTokens = make(map[string]database.RefreshToken)
	return nil
}

func setupTestCfg(t *testing.T) apiConfig {
	t.Helper()
	return apiConfig{
		db:       newMockDB(),
		platform: "dev",
		secret:   "test-secret-key",
		polkaKey: "test-polka-key",
	}
}

func TestReadiness(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/healthz", nil)
	w := httptest.NewRecorder()
	handlerReadiness(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if w.Body.String() != http.StatusText(http.StatusOK) {
		t.Fatalf("body: got %v, want %v", w.Body.String(), http.StatusText(http.StatusOK))
	}
}

func TestValidateChirp(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		wantCode      int
		wantError     bool
		wantCleaned   string
	}{
		{"valid", "Hello world", http.StatusOK, false, "Hello world"},
		{"too long", string(make([]byte, 141)), http.StatusBadRequest, true, ""},
		{"profanity", "I love kerfuffle and sharbert", http.StatusOK, false, "I love **** and ****"},
		{"case insensitive", "Kerfuffle is bad", http.StatusOK, false, "**** is bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"body": tt.body})
			req := httptest.NewRequest("POST", "/api/validate_chirp", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handlerChirpsValidate(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantCode {
				t.Fatalf("status: got %d, want %d", resp.StatusCode, tt.wantCode)
			}
			var result map[string]string
			json.NewDecoder(resp.Body).Decode(&result)
			if tt.wantError {
				if _, ok := result["error"]; !ok {
					t.Fatal("expected error response")
				}
			} else if result["cleaned_body"] != tt.wantCleaned {
				t.Fatalf("cleaned_body: got %v, want %v", result["cleaned_body"], tt.wantCleaned)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	cfg := setupTestCfg(t)

	body, _ := json.Marshal(map[string]string{"email": "alice@test.com", "password": "secret"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cfg.handlerUsersCreate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var u User
	json.NewDecoder(resp.Body).Decode(&u)
	if u.ID == uuid.Nil {
		t.Fatal("expected non-zero UUID")
	}
	if u.Email != "alice@test.com" {
		t.Fatalf("email: got %v, want %v", u.Email, "alice@test.com")
	}
	if u.Password != "" {
		t.Fatal("password should not be serialized")
	}
}

func TestCreateUser_EmptyPassword(t *testing.T) {
	cfg := setupTestCfg(t)

	body, _ := json.Marshal(map[string]string{"email": "bob@test.com", "password": ""})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cfg.handlerUsersCreate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestLogin(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create user first
	createBody, _ := json.Marshal(map[string]string{"email": "login@test.com", "password": "pass123"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	// Login
	loginBody, _ := json.Marshal(map[string]string{"email": "login@test.com", "password": "pass123"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)

	resp := loginW.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Token == "" {
		t.Fatal("expected access token")
	}
	if result.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if result.Email != "login@test.com" {
		t.Fatalf("email: got %v, want %v", result.Email, "login@test.com")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	cfg := setupTestCfg(t)

	createBody, _ := json.Marshal(map[string]string{"email": "wrong@test.com", "password": "correct"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "wrong@test.com", "password": "wrong"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)

	resp := loginW.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestUpdateUser(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create user
	createBody, _ := json.Marshal(map[string]string{"email": "old@test.com", "password": "pass123"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	// Login to get token
	loginBody, _ := json.Marshal(map[string]string{"email": "old@test.com", "password": "pass123"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Update user
	updateBody, _ := json.Marshal(map[string]string{"email": "new@test.com", "password": "newpass"})
	updateReq := httptest.NewRequest("PUT", "/api/users", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	updateW := httptest.NewRecorder()
	cfg.handlerUserUpdate(updateW, updateReq)

	resp := updateW.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var u User
	json.NewDecoder(resp.Body).Decode(&u)
	if u.Email != "new@test.com" {
		t.Fatalf("email: got %v, want %v", u.Email, "new@test.com")
	}
}

func TestUpdateUser_NoAuth(t *testing.T) {
	cfg := setupTestCfg(t)
	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "pass"})
	req := httptest.NewRequest("PUT", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cfg.handlerUserUpdate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestCreateChirp(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create + login to get token
	createBody, _ := json.Marshal(map[string]string{"email": "chirper@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "chirper@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Create chirp
	chirpBody, _ := json.Marshal(map[string]string{"body": "hello chirp"})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)

	resp := chirpW.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var c Chirp
	json.NewDecoder(resp.Body).Decode(&c)
	if c.Body != "hello chirp" {
		t.Fatalf("body: got %v, want %v", c.Body, "hello chirp")
	}
	if c.ID == uuid.Nil {
		t.Fatal("expected non-zero chirp ID")
	}
}

func TestCreateChirp_WithoutAuth(t *testing.T) {
	cfg := setupTestCfg(t)
	body, _ := json.Marshal(map[string]string{"body": "no auth"})
	req := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cfg.handlerChirpsCreate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestListChirps(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create user + chirps
	createBody, _ := json.Marshal(map[string]string{"email": "list@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)
	var u User
	json.NewDecoder(createW.Body).Decode(&u)

	// Login
	loginBody, _ := json.Marshal(map[string]string{"email": "list@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Create 3 chirps
	for i := 0; i < 3; i++ {
		chirpBody, _ := json.Marshal(map[string]string{"body": "chirp"})
		chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
		chirpReq.Header.Set("Content-Type", "application/json")
		chirpReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
		chirpW := httptest.NewRecorder()
		cfg.handlerChirpsCreate(chirpW, chirpReq)
	}

	// List all
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

func TestListChirps_ByAuthor(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create two users
	for _, email := range []string{"alice@test.com", "bob@test.com"} {
		body, _ := json.Marshal(map[string]string{"email": email, "password": "pass"})
		req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		cfg.handlerUsersCreate(w, req)
		var u User
		json.NewDecoder(w.Body).Decode(&u)
	}

	// Login as bob and create chirps
	loginBody, _ := json.Marshal(map[string]string{"email": "bob@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	chirpBody, _ := json.Marshal(map[string]string{"body": "bob chirp"})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)
	var bobChirp Chirp
	json.NewDecoder(chirpW.Body).Decode(&bobChirp)

	// Get by author
	req := httptest.NewRequest("GET", "/api/chirps?author_id="+bobChirp.UserID.String(), nil)
	w := httptest.NewRecorder()
	cfg.handlerChirpsRetrieve(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var chirps []Chirp
	json.NewDecoder(resp.Body).Decode(&chirps)
	if len(chirps) != 1 {
		t.Fatalf("expected 1 chirp, got %d", len(chirps))
	}
}

func TestGetChirpByID(t *testing.T) {
	cfg := setupTestCfg(t)

	createBody, _ := json.Marshal(map[string]string{"email": "get@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "get@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	chirpBody, _ := json.Marshal(map[string]string{"body": "find me"})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)
	var created Chirp
	json.NewDecoder(chirpW.Body).Decode(&created)

	// Get by ID
	req := httptest.NewRequest("GET", "/api/chirps/"+created.ID.String(), nil)
	req.SetPathValue("chirpID", created.ID.String())
	w := httptest.NewRecorder()
	cfg.handlerChirpsGet(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var c Chirp
	json.NewDecoder(resp.Body).Decode(&c)
	if c.ID != created.ID {
		t.Fatalf("ID: got %v, want %v", c.ID, created.ID)
	}
}

func TestGetChirpByID_NotFound(t *testing.T) {
	cfg := setupTestCfg(t)
	id := uuid.New().String()
	req := httptest.NewRequest("GET", "/api/chirps/"+id, nil)
	req.SetPathValue("chirpID", id)
	w := httptest.NewRecorder()
	cfg.handlerChirpsGet(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteChirp(t *testing.T) {
	cfg := setupTestCfg(t)

	createBody, _ := json.Marshal(map[string]string{"email": "del@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "del@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	chirpBody, _ := json.Marshal(map[string]string{"body": "delete me"})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)
	var created Chirp
	json.NewDecoder(chirpW.Body).Decode(&created)

	// Delete
	delReq := httptest.NewRequest("DELETE", "/api/chirps/"+created.ID.String(), nil)
	delReq.SetPathValue("chirpID", created.ID.String())
	delReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	delW := httptest.NewRecorder()
	cfg.handlerChirpDelete(delW, delReq)
	resp := delW.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestDeleteChirp_NotOwner(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create user alice
	body, _ := json.Marshal(map[string]string{"email": "alice@test.com", "password": "pass"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cfg.handlerUsersCreate(w, req)

	// Create user bob
	body2, _ := json.Marshal(map[string]string{"email": "bob@test.com", "password": "pass"})
	req2 := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	cfg.handlerUsersCreate(w2, req2)

	// Login as alice and create a chirp
	loginBody, _ := json.Marshal(map[string]string{"email": "alice@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var aliceLogin struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW.Body).Decode(&aliceLogin)

	chirpBody, _ := json.Marshal(map[string]string{"body": "alice chirp"})
	chirpReq := httptest.NewRequest("POST", "/api/chirps", bytes.NewReader(chirpBody))
	chirpReq.Header.Set("Content-Type", "application/json")
	chirpReq.Header.Set("Authorization", "Bearer "+aliceLogin.Token)
	chirpW := httptest.NewRecorder()
	cfg.handlerChirpsCreate(chirpW, chirpReq)
	var aliceChirp Chirp
	json.NewDecoder(chirpW.Body).Decode(&aliceChirp)

	// Login as bob and try to delete alice's chirp
	loginBody2, _ := json.Marshal(map[string]string{"email": "bob@test.com", "password": "pass"})
	loginReq2 := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody2))
	loginReq2.Header.Set("Content-Type", "application/json")
	loginW2 := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW2, loginReq2)
	var bobLogin struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginW2.Body).Decode(&bobLogin)

	delReq := httptest.NewRequest("DELETE", "/api/chirps/"+aliceChirp.ID.String(), nil)
	delReq.SetPathValue("chirpID", aliceChirp.ID.String())
	delReq.Header.Set("Authorization", "Bearer "+bobLogin.Token)
	delW := httptest.NewRecorder()
	cfg.handlerChirpDelete(delW, delReq)
	resp := delW.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestRefreshToken(t *testing.T) {
	cfg := setupTestCfg(t)

	createBody, _ := json.Marshal(map[string]string{"email": "refresh@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "refresh@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Exchange refresh token for new JWT
	refreshReq := httptest.NewRequest("POST", "/api/refresh", nil)
	refreshReq.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken)
	refreshW := httptest.NewRecorder()
	cfg.handlerRefreshToken(refreshW, refreshReq)
	resp := refreshW.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var result struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Token == "" {
		t.Fatal("expected new JWT token")
	}
}

func TestRevokeToken(t *testing.T) {
	cfg := setupTestCfg(t)

	createBody, _ := json.Marshal(map[string]string{"email": "revoke@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "revoke@test.com", "password": "pass"})
	loginReq := httptest.NewRequest("POST", "/api/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	cfg.handlerUserLogin(loginW, loginReq)
	var loginResp struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Revoke
	revokeReq := httptest.NewRequest("POST", "/api/revoke", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken)
	revokeW := httptest.NewRecorder()
	cfg.handlerRevokeToken(revokeW, revokeReq)
	resp := revokeW.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Try to use revoked token - should fail
	refreshReq := httptest.NewRequest("POST", "/api/refresh", nil)
	refreshReq.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken)
	refreshW := httptest.NewRecorder()
	cfg.handlerRefreshToken(refreshW, refreshReq)
	resp2 := refreshW.Result()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revoke, got %d", resp2.StatusCode)
	}
}

func TestPolkaWebhook_Upgrade(t *testing.T) {
	cfg := setupTestCfg(t)

	// Create user
	createBody, _ := json.Marshal(map[string]string{"email": "upgrade@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)
	var u User
	json.NewDecoder(createW.Body).Decode(&u)

	// Send webhook
	payload := map[string]any{
		"event": "user.upgraded",
		"data":  map[string]any{"user_id": u.ID.String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/polka/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey test-polka-key")
	w := httptest.NewRecorder()
	cfg.handlerAddChirpyRedMembership(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestPolkaWebhook_BadAPIKey(t *testing.T) {
	cfg := setupTestCfg(t)

	payload := map[string]any{
		"event": "user.upgraded",
		"data":  map[string]any{"user_id": uuid.New().String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/polka/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey wrong-key")
	w := httptest.NewRecorder()
	cfg.handlerAddChirpyRedMembership(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestPolkaWebhook_IgnoredEvent(t *testing.T) {
	cfg := setupTestCfg(t)

	payload := map[string]any{
		"event": "some.other.event",
		"data":  map[string]any{"user_id": uuid.New().String()},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/polka/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey test-polka-key")
	w := httptest.NewRecorder()
	cfg.handlerAddChirpyRedMembership(w, req)

	resp := w.Result()
	// Non-upgrade events are silently accepted
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestMetrics(t *testing.T) {
	cfg := setupTestCfg(t)
	cfg.fileserverHits.Store(42)

	req := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()
	cfg.handlerMetrics(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type: got %v", resp.Header.Get("Content-Type"))
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("42")) {
		t.Fatal("expected hit count in body")
	}
}

func TestReset_Dev(t *testing.T) {
	cfg := setupTestCfg(t)
	cfg.fileserverHits.Store(10)

	// Create some data first
	createBody, _ := json.Marshal(map[string]string{"email": "reset@test.com", "password": "pass"})
	createReq := httptest.NewRequest("POST", "/api/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	cfg.handlerUsersCreate(createW, createReq)

	// Reset
	req := httptest.NewRequest("POST", "/admin/reset", nil)
	w := httptest.NewRecorder()
	cfg.handlerReset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if cfg.fileserverHits.Load() != 0 {
		t.Fatalf("hits: got %d, want 0", cfg.fileserverHits.Load())
	}
}

func TestReset_NotDev(t *testing.T) {
	cfg := setupTestCfg(t)
	cfg.platform = "production"

	req := httptest.NewRequest("POST", "/admin/reset", nil)
	w := httptest.NewRecorder()
	cfg.handlerReset(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestGetCleanedBody(t *testing.T) {
	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"I love kerfuffle", "I love ****"},
		{"Kerfuffle is bad", "**** is bad"},
		{"KERFUFFLE", "****"},
		{"multiple sharbert and fornax", "multiple **** and ****"},
	}
	for _, tt := range tests {
		got := getCleanedBody(tt.input, badWords)
		if got != tt.expected {
			t.Errorf("getCleanedBody(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidateChirpFunc(t *testing.T) {
	tests := []struct {
		body    string
		wantErr bool
	}{
		{"hello", false},
		{string(make([]byte, 140)), false},
		{string(make([]byte, 141)), true},
	}
	for _, tt := range tests {
		_, err := validateChirp(tt.body)
		if tt.wantErr && err == nil {
			t.Error("expected error for long chirp")
		}
		if !tt.wantErr && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestParseSortOrder(t *testing.T) {
	tests := []struct {
		input   string
		want    SortOrder
		wantErr bool
	}{
		{"asc", SortAsc, false},
		{"desc", SortDesc, false},
		{"invalid", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := parseSortOrder(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("parseSortOrder(%q) expected error", tt.input)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parseSortOrder(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
