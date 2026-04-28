# Go Backend Template

A production-ready Go backend template with Chi router, sqlc, and OpenTelemetry.

## Table of Contents

- [Project Description](#project-description)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Project Architecture](#project-architecture)
- [Project Structure](#project-structure)
- [Commands](#commands)
- [Contributing](#contributing)
- [License](#license)
- [Documentation](#documentation)

## Project Description

A production-ready Go backend template with Chi router, sqlc for type-safe SQL, PostgreSQL database, JWT authentication, approved users gate, OpenTelemetry distributed tracing, and Zap structured logging.

## Prerequisites

- Go 1.26+
- PostgreSQL 16+
- Docker and docker-compose (optional)

## Installation

### Using Docker Compose (Recommended)

```bash
# Build and start all services (PostgreSQL, Jaeger, API)
docker-compose up --build

# Run migrations (in a new terminal)
make migrate-up

# View logs
docker-compose logs -f api
```

### Local Development

```bash
# Generate SQLC code
make sqlc-gen

# Run migrations
make migrate-up

# Start the API
make run
```

## Configuration

Copy `.env.example` to `.env` and configure:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server bind host | `"0.0.0.0"` |
| `SERVER_PORT` | Server port | `8080` |
| `SERVER_READ_TIMEOUT` | Read timeout | `30s` |
| `SERVER_WRITE_TIMEOUT` | Write timeout | `30s` |
| `DATABASE_URL` | PostgreSQL connection URL | **required** |
| `DATABASE_MAX_OPEN_CONNS` | Max open connections | `25` |
| `DATABASE_MAX_IDLE_CONNS` | Max idle connections | `5` |
| `DATABASE_CONN_MAX_LIFETIME` | Connection max lifetime | `5m` |
| `JWT_SECRET_KEY` | JWT signing secret | **required** |
| `JWT_EXPIRE_MINUTES` | JWT expiry in minutes | `1440` |
| `BCRYPT_COST` | Bcrypt hashing cost | `12` |
| `LOG_LEVEL` | Log level | `"info"` |
| `LOG_FORMAT` | Log format (`json` or `console`) | `"json"` |
| `TRACING_ENABLED` | Enable OpenTelemetry tracing | `true` |
| `SERVICE_NAME` | Service name for telemetry | `"go-backend-template"` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint | `"http://localhost:4318"` |
| `OTEL_EXPORTER_OTLP_INSECURE` | Allow insecure OTLP | `true` |
| `RATE_LIMIT_REQUESTS` | Requests per period | `100` |
| `RATE_LIMIT_DURATION` | Rate limit period | `1m` |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `*` |
| `CORS_ALLOWED_METHODS` | Allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS` |
| `CORS_ALLOWED_HEADERS` | Allowed request headers | `Accept,Authorization,Content-Type` |
| `CORS_ALLOW_CREDENTIALS` | Allow credentials | `true` |
| `CORS_MAX_AGE` | Preflight cache max age | `3600` |

## Project Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         API Entry (cmd/api)                         │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Chi Router (internal/router)                      │
│  ┌─────────────┐ ┌──────────┐ ┌────────┐ ┌───────────┐             │
│  │ RequestID   │ │ RealIP   │ │ Logger │ │ Timeout   │  + CORS     │
│  │ Middleware  │ │          │ │        │ │ (30s)     │             │
│  └─────────────┘ └──────────┘ └────────┘ └───────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
             ┌──────────┐   ┌──────────┐    ┌──────────────┐
             │  Auth    │   │  Todos   │    │   Admin      │
             │ Handler  │   │ Handler  │    │   Routes     │
             └────┬─────┘   └────┬─────┘    └──────┬───────┘
                  │              │                  │
                  ▼              ▼                  ▼
           ┌──────────┐   ┌──────────┐    ┌──────────────┐
           │  Auth    │   │  Todo    │    │   Auth       │
           │ Service  │   │ Service  │    │   Service    │
           └────┬─────┘   └────┬─────┘    └──────┬───────┘
                │              │                  │
                ▼              ▼                  ▼
           ┌──────────┐   ┌──────────┐    ┌──────────────┐
           │  Auth    │   │  Todo    │    │   Auth        │
           │ Repository│   │ Repository│   │   Repository  │
           └────┬─────┘   └────┬─────┘    └──────┬───────┘
                │              │                  │
                └──────────────┼──────────────────┘
                               ▼
                    ┌─────────────────────┐
                    │   PostgreSQL (pgx)   │
                    │   + sqlc generated  │
                    └─────────────────────┘
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| **API Entry** | `cmd/api/` | HTTP server entry point |
| **Migrate CLI** | `cmd/migrate/` | Database migration runner |
| **Router** | `internal/router/` | Chi router + middleware |
| **Auth Handler** | `internal/auth/` | Auth endpoints (register/login/admin) |
| **Todo Handler** | `internal/todo/` | Todo CRUD endpoints |
| **Auth Service** | `internal/auth/` | Business logic for auth |
| **Todo Service** | `internal/todo/` | Business logic for todos |
| **Domain** | `internal/domain/` | Domain models (User, Todo, etc.) |
| **Config** | `internal/config/` | Configuration loading |
| **Middleware** | `internal/middleware/` | Auth + CORS + rate limiting |
| **Observability** | `internal/observability/` | OpenTelemetry setup |
| **Logging** | `internal/logging/` | Zap logger setup |
| **HTTP Utils** | `internal/http/` | Response helpers |

## Project Structure

```
.
├── cmd/
│   ├── api/          # API entry point
│   └── migrate/      # Migration CLI tool
├── internal/
│   ├── auth/         # Authentication handlers, services, repositories
│   ├── config/       # Configuration loading from environment
│   ├── db/           # Database connection pool + sqlc generated code
│   ├── domain/       # Domain models (User, ApprovedUser, Todo, Role)
│   ├── http/         # HTTP utilities (RespondJSON, RespondError)
│   ├── logging/      # Zap logger setup with trace context
│   ├── middleware/   # HTTP middleware (auth, CORS, rate limit, timeout)
│   ├── observability/ # OpenTelemetry tracing setup
│   ├── router/       # Chi router configuration and middleware
│   ├── todo/         # Todo feature handlers, services, repositories
│   └── validator/    # Input validation (email, password, etc.)
├── migrations/       # Database migration files (golang-migrate)
├── sql/
│   └── queries/      # SQLC query definitions
├── docs/             # Generated documentation
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── sqlc.yaml
```

## Commands

```bash
# Build and run
make build            # Build the API binary
make run              # Run the API server

# Testing
make test             # Run all tests (unit + integration)
make test-unit        # Run unit tests only
make test-integration # Run integration tests only (requires Docker)
make test-coverage    # Run tests with coverage report

# Code quality
make lint             # Run golangci-lint
make fmt              # Format code
make vet              # Run go vet
make tidy             # Tidy go modules

# Database
make migrate-up       # Apply all migrations
make migrate-down     # Rollback last migration
make migrate-reset    # Reset database
make migrate-version  # Show current migration version

# Code generation
make sqlc-gen         # Generate sqlc code
make sqlc-compile     # Validate sqlc schema/queries

# Docker
make docker-build     # Build Docker image

# Full verification
make verify           # Run fmt, vet, lint, sqlc-compile, test
make ci               # Run full CI pipeline

# Cleanup
make clean            # Clean build artifacts
```

## Contributing

Contributions are welcome. Please ensure code passes `make verify` before submitting PRs.

## License

MIT

## Documentation

- [API Reference](docs/api.md) - Complete API endpoint documentation
- [Features](docs/features.md) - Feature inventory with implementation status