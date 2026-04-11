# Go Backend Template Design

**Date:** 2026-04-11
**Status:** Approved

## Overview

A reusable Go backend template built with Chi, sqlc, and PostgreSQL. Feature-first project structure based on `pre-consult-backend`. Includes auth (with approved-users gate), user-scoped todo CRUD, health endpoint, Zap logging, and OpenTelemetry tracing.

---

## Project Structure

```
go-backend-template/
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point: wires deps, starts HTTP server
│   └── migrate/
│       └── main.go              # Migration runner (up/down/reset/force/version)
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based config, validation
│   ├── db/
│   │   ├── db.go                # pgx connection pool
│   │   ├── sqlc/                # Generated code (sqlc generate)
│   │   └── dbutil/
│   │       └── convert.go       # pgtype ↔ Go type helpers
│   ├── domain/
│   │   └── user.go              # User, ApprovedUser, Role — shared by auth + middleware
│   ├── http/
│   │   └── response.go          # RespondJSON, RespondError
│   ├── logging/
│   │   └── logger.go            # Zap logger factory + WithTraceContext
│   ├── middleware/
│   │   ├── auth.go              # RequireAuth, RequireAdmin, OptionalAuth
│   │   └── http.go              # RateLimiter, context keys, extractBearerToken
│   ├── observability/
│   │   └── otel.go              # OTel tracer provider setup
│   ├── router/
│   │   └── router.go            # Chi router, all routes, middleware stack
│   ├── auth/
│   │   ├── handler.go           # HTTP handlers for auth + admin routes
│   │   ├── service.go           # Business logic, JWT, bcrypt, interfaces
│   │   ├── repository.go        # DB queries via sqlc
│   │   ├── models.go            # Request/response DTOs, toXxxResponse helpers
│   │   ├── handler_test.go
│   │   └── service_test.go
│   ├── todo/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   ├── handler_test.go
│   │   └── service_test.go
│   └── integration/
│       ├── main_test.go         # testcontainers postgres setup, shared router
│       ├── auth_test.go
│       ├── todo_test.go
│       └── helpers_test.go
├── migrations/
│   ├── 000001_create_extensions.{up,down}.sql
│   ├── 000002_create_approved_users_table.{up,down}.sql
│   ├── 000003_create_users_table.{up,down}.sql
│   ├── 000004_create_roles_tables.{up,down}.sql
│   └── 000005_create_todos_table.{up,down}.sql
├── sql/
│   └── queries/
│       ├── auth.sql
│       └── todo.sql
├── docs/superpowers/specs/
├── .env.example
├── .golangci.yml
├── .commitlintrc.yml
├── .dockerignore
├── .gitignore
├── docker-compose.yml
├── Dockerfile
├── lefthook.yml
├── Makefile
├── sqlc.yaml
└── README.md
```

**Key structural decisions:**
- `internal/domain/user.go` is the only shared domain type — `User`, `ApprovedUser`, `Role`. All other types live in their feature's `models.go`.
- The `admin` feature from pre-consult is merged into the `auth` feature — approved-users handlers live in `internal/auth/` and are registered under `/admin` routes. Same user management concern, one package.
- No `internal/types/` or LiveKit-specific packages.

---

## API Routes

**Global middleware stack:** `RequestID → RealIP → Logger(Zap+trace) → Recoverer → Timeout(60s) → CORS`

```
GET  /                          # {"message": "go-backend-template", "version": "..."}
GET  /health                    # {"status": "ok"}

# Auth (rate-limited)
POST /auth/signup               # public
POST /auth/login                # public
POST /auth/token                # public — OAuth2-compatible form login

# Auth (protected)
GET  /auth/me                   # RequireAuth
POST /auth/change-password      # RequireAuth

# Admin
POST /admin/approved-users      # RequireAdmin — create approved user
GET  /admin/approved-users      # RequireAdmin — list approved users

# Todos (all RequireAuth, user-scoped)
GET    /todos                   # list current user's todos
POST   /todos                   # create todo
GET    /todos/{id}              # get todo (user must own it)
PUT    /todos/{id}              # update todo (user must own it)
DELETE /todos/{id}              # delete todo (user must own it)
```

---

## Database Schema

**approved_users**
```sql
id         UUID PK DEFAULT gen_random_uuid()
email      VARCHAR(255) NOT NULL UNIQUE
first_name VARCHAR(255) NOT NULL
created_by_user_id UUID FK → users(id) ON DELETE SET NULL  -- added post-users
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

**users**
```sql
id               UUID PK DEFAULT gen_random_uuid()
approved_user_id UUID NOT NULL UNIQUE FK → approved_users(id) ON DELETE RESTRICT
hashed_password  VARCHAR(255) NOT NULL
is_active        BOOLEAN NOT NULL DEFAULT TRUE
created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

**roles** + **user_roles** (many-to-many)
```sql
roles: id, name (UNIQUE), description, created_at
user_roles: id, user_id FK, role_id FK, UNIQUE(user_id, role_id)
-- seeded: 'admin', 'user'
```

**todos**
```sql
id         UUID PK DEFAULT gen_random_uuid()
user_id    UUID NOT NULL FK → users(id) ON DELETE CASCADE
title      VARCHAR(255) NOT NULL
completed  BOOLEAN NOT NULL DEFAULT false
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

---

## Data Flow & Key Patterns

### Dependency injection (cmd/api/main.go)
```
config.Load()
  → db.New()
  → auth.NewRepository(pool) → auth.NewService(repo, cfg.Auth) → auth.NewHandler(svc, logger)
  → todo.NewRepository(pool) → todo.NewService(repo)           → todo.NewHandler(svc, logger)
  → router.New(RouterConfig{...})
  → startHTTPServer()
```

### Layered architecture (per feature)
```
Handler  (HTTP: parse, validate, respond)
  ↓
Service  (business logic, error mapping, interfaces)
  ↓
Repository  (sqlc queries, repo sentinel errors)
  ↓
DB (pgx pool + sqlc generated code)
```

### Error handling
- Repository defines sentinel errors: `ErrRepoEmailNotApproved`, `ErrRepoUserAlreadyExists`
- Service maps repo errors → service errors: `ErrEmailNotApproved`, `ErrUserAlreadyExists`, `ErrNotFound`, `ErrForbidden`
- Handler maps service errors → HTTP status via `errors.Is()`
- All error responses: `{"detail": "message"}`

### Todo ownership enforcement
- Handler extracts `userID` from context (set by `RequireAuth` middleware)
- All repository queries include `WHERE user_id = $userID` — database enforces ownership, no separate check needed
- Attempting to access another user's todo returns 404 (not 403) to avoid leaking existence

### Auth flow
1. Admin creates `approved_users` entry (email whitelist)
2. User calls `POST /auth/signup` — fails if email not in `approved_users`
3. User calls `POST /auth/login` → JWT returned (`{"access_token": "...", "token_type": "bearer"}`)
4. `RequireAuth` middleware: extract Bearer token → validate JWT → fetch fresh user from DB → inject `*domain.User` into context
5. `RequireAdmin`: wraps `RequireAuth` + checks `user.HasRole("admin")`

---

## Configuration

**Required** (server refuses to start without): `DATABASE_URL`, `JWT_SECRET_KEY`

**All env vars:**

| Variable | Default | Notes |
|---|---|---|
| `SERVER_HOST` | `0.0.0.0` | |
| `SERVER_PORT` | `8080` | |
| `SERVER_READ_TIMEOUT` | `30s` | |
| `SERVER_WRITE_TIMEOUT` | `30s` | |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | |
| `DATABASE_URL` | — | Required |
| `DATABASE_MAX_OPEN_CONNS` | `25` | |
| `DATABASE_MAX_IDLE_CONNS` | `5` | |
| `DATABASE_CONN_MAX_LIFETIME` | `5m` | |
| `JWT_SECRET_KEY` | — | Required |
| `JWT_EXPIRE_MINUTES` | `1440` | 24h |
| `BCRYPT_COST` | `12` | |
| `LOG_LEVEL` | `info` | debug/info/warn/error/fatal |
| `LOG_FORMAT` | `json` | json or console |
| `TRACING_ENABLED` | `true` | |
| `SERVICE_NAME` | `go-backend-template` | |
| `SERVICE_VERSION` | `1.0.0` | |
| `ENVIRONMENT` | `development` | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | |
| `OTEL_TRACE_SAMPLING_RATIO` | `1.0` | |
| `RATE_LIMIT_REQUESTS` | `10` | |
| `RATE_LIMIT_DURATION` | `1m` | |
| `CORS_ALLOWED_ORIGINS` | `""` | empty = allow all |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,DELETE,OPTIONS` | |
| `CORS_ALLOWED_HEADERS` | `Accept,Authorization,Content-Type` | |
| `CORS_ALLOW_CREDENTIALS` | `true` | |
| `CORS_MAX_AGE` | `3600` | |

---

## Infrastructure

**docker-compose.yml** (local dev):
- `postgres:18-alpine` with healthcheck
- `db-migrate` service (runs migrations once, `restart: no`)
- `api` service (depends on postgres healthy + migrate completed)

**Dockerfile:** multi-stage Alpine, CGO disabled, non-root user (`appuser:appgroup`), builds both `api` and `migrate` binaries.

**Makefile targets:** `build`, `build-migrate`, `run`, `test`, `test-unit`, `test-integration`, `test-coverage`, `lint`, `fmt`, `vet`, `tidy`, `clean`, `install-tools`, `sqlc-gen`, `sqlc-compile`, `migrate-up`, `migrate-down`, `migrate-reset`, `migrate-version`, `migrate-force`, `docker-build`, `verify`, `ci`

**sqlc.yaml:** PostgreSQL engine, schema from `migrations/`, queries from `sql/queries/`, output to `internal/db/sqlc/`, UUID override to `github.com/google/uuid`.

**Tooling:** golangci-lint, lefthook (commit-msg → commitlint), conventional commits enforced.

---

## Module Name

`github.com/your-org/go-backend-template` — teams replace `your-org` and the repo name when adopting the template.

---

## Testing Strategy

- **Unit tests** alongside each feature (`handler_test.go`, `service_test.go`) using mocked interfaces
- **Integration tests** in `internal/integration/` using testcontainers (real Postgres, full HTTP stack)
- Integration tests gated behind `//go:build integration` tag
- `make test-unit` — fast, no Docker required
- `make test-integration` — requires Docker, 120s timeout
