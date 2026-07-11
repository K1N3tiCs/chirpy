# Chirpy

A social media-style chirp (micro-blog post) API built with Go and PostgreSQL.

## Prerequisites

- Go 1.26+
- PostgreSQL

## Setup

```bash
# Clone and enter the project
git clone <repo-url> && cd chirpy

# Copy and configure environment
cp .env.example .env
# Edit .env with your DB_URL, PLATFORM, SECRET, and POLKA_KEY

# Run database migrations
psql "$DB_URL" -f sql/schema/001_users.sql
psql "$DB_URL" -f sql/schema/002_chirps.sql
psql "$DB_URL" -f sql/schema/003_users.sql
psql "$DB_URL" -f sql/schema/004_refresh_tokens.sql
psql "$DB_URL" -f sql/schema/005_users.sql

# Start the server
go build -o chirpy && ./chirpy
```

Server runs on `http://localhost:8080`.

## Environment Variables

| Variable    | Description                        |
|-------------|------------------------------------|
| `DB_URL`    | PostgreSQL connection string       |
| `PLATFORM`  | `dev` or `production`              |
| `SECRET`    | JWT signing secret                 |
| `POLKA_KEY` | API key for Polka webhook auth     |

## API

Full OpenAPI spec: [`docs/openapi.yaml`](docs/openapi.yaml)

### Quick Reference

| Method | Path                     | Auth              | Description                  |
|--------|--------------------------|-------------------|------------------------------|
| GET    | `/api/healthz`           | —                 | Readiness probe              |
| POST   | `/api/validate_chirp`    | —                 | Validate & profanity-filter  |
| POST   | `/api/users`             | —                 | Create user                  |
| PUT    | `/api/users`             | Bearer JWT        | Update email/password        |
| POST   | `/api/login`             | —                 | Login (get JWT + refresh)    |
| POST   | `/api/refresh`           | Bearer refresh    | Exchange refresh for JWT     |
| POST   | `/api/revoke`            | Bearer refresh    | Revoke refresh token         |
| GET    | `/api/chirps`            | —                 | List chirps (sort, filter)   |
| GET    | `/api/chirps/{id}`       | —                 | Get chirp by ID              |
| POST   | `/api/chirps`            | Bearer JWT        | Create chirp                 |
| DELETE | `/api/chirps/{id}`       | Bearer JWT        | Delete own chirp             |
| POST   | `/api/polka/webhooks`    | ApiKey            | Polka webhook (Chirpy Red)   |
| GET    | `/admin/metrics`         | —                 | View fileserver hit count    |
| POST   | `/admin/reset`           | —                 | Reset DB (dev only)          |

### Example: Create user and post a chirp

```bash
# Create user
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}'
# → returns token and refresh_token

# Post a chirp (use token from login response)
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"body":"Hello, world!"}'

# List chirps sorted by newest
curl "http://localhost:8080/api/chirps?sort=desc"
```

## Running Tests

```bash
# Requires DB_URL env var pointing to a test database
go test ./...
```
