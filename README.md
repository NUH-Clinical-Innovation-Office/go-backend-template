# Go Backend Template

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue)](https://golang.org/doc/go1.26)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

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

A production-ready Go backend template implementing a REST API with JWT authentication, PostgreSQL database, and comprehensive observability. Designed for rapid project bootstrapping with industry best practices built in.

| Feature | Description |
|---------|-------------|
| Chi Router | Lightweight, idiomatic HTTP routing |
| sqlc | Type-safe SQL code generation |
| PostgreSQL | Primary database with pgx driver |
| JWT Authentication | Secure token-based auth with bcrypt password hashing |
| Approved Users Gate | Email whitelist for controlled registration |
| OpenTelemetry | Distributed tracing with Jaeger/OTLP support |
| Zap Logging | Structured, high-performance logging |
| Database Migrations | Using golang-migrate |
| Docker Support | Multi-stage builds and docker-compose |
| CORS Middleware | Cross-origin request handling with configurable origins |
| Request ID Middleware | Unique request ID per request for tracing |
| Real IP Middleware | Extracts real client IP from proxy headers |
| Timeout Middleware | 30-second request timeout protection |
| Recoverer Middleware | Chi panic recovery — converts panics to 500s |
| Optional Auth Middleware | Attaches user to context if a valid token is present (wired but unused) |
| Integration Tests | testcontainers-go for real database testing |

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

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=120s
SERVER_SHUTDOWN_TIMEOUT=10s

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/go_backend_template?sslmode=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m

# JWT
JWT_SECRET_KEY=your-super-secret-jwt-key-change-in-production
JWT_EXPIRE_MINUTES=1440
BCRYPT_COST=12

# Logging
LOG_LEVEL=info
LOG_FORMAT=console

# OpenTelemetry
TRACING_ENABLED=true
SERVICE_NAME=go-backend-template
SERVICE_VERSION=1.0.0
ENVIRONMENT=development
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_TRACE_SAMPLING_RATIO=1.0
OTEL_EXPORTER_OTLP_INSECURE=true

# Rate Limit Configuration (for future use)
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_DURATION=1m

# CORS Configuration
CORS_ALLOWED_ORIGINS=*
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Accept,Authorization,Content-Type
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

## Project Architecture

The application follows a layered architecture:

```
                    ┌─────────────────────────────────────┐
                    │            Router (Chi)             │
                    │ Middleware: CORS, Auth, Timeout,   │
                    │ RequestID, RealIP, Recoverer       │
                    └──────────────┬──────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
        ▼                          ▼                          ▼
┌──────────────────────┐  ┌───────────────┐  ┌──────────────────────┐
│  Auth Handler        │  │ Todo Handler  │  │ Admin Handler        │
│  /auth/* + /me       │  │  /todos/*     │  │  /admin/approved-    │
│  + /admin/approved-  │  │               │  │  users/* (admin)     │
│  users/* (admin)     │  │               │  │                      │
└──────────┬───────────┘  └───────┬───────┘  └──────────┬───────────┘
           │                       │                     │
           ▼                       ▼                     ▼
   ┌───────────────┐       ┌───────────────┐    (delegates to Auth Svc)
   │   Auth Svc    │       │  Todo Svc     │
   └───────┬───────┘       └───────┬───────┘
           │                       │
           ▼                       ▼
   ┌───────────────┐       ┌───────────────┐
   │  SQLC Queries │       │  SQLC Queries │
   └───────┬───────┘       └───────┬───────┘
           │                       │
           ▼                       ▼
┌─────────────────────────────────────────────┐
│              PostgreSQL Database             │
└─────────────────────────────────────────────┘
```

- **Router**: Chi mux with middleware chain for request ID, real IP, logging, recovery, timeout, and CORS
- **Handlers**: HTTP request handlers delegate to services; auth and admin endpoints share the same `auth.Handler` (admin is gated by the `RequireAdmin` middleware)
- **Services**: Business logic layer
- **SQLC**: Type-safe database queries generated from SQL

## Project Structure

```
.
├── cmd/
│   ├── api/          # API entry point
│   └── migrate/      # Migration CLI tool
├── internal/
│   ├── auth/         # Authentication feature
│   ├── config/       # Configuration loading
│   ├── db/           # Database connection and SQLC generated code
│   ├── domain/       # Shared domain models
│   ├── http/         # HTTP utilities
│   ├── integration/  # Integration tests
│   ├── logging/      # Logging setup
│   ├── middleware/   # HTTP middleware
│   ├── observability/# OpenTelemetry setup
│   ├── router/       # Router configuration
│   ├── todo/         # Todo feature (example CRUD)
│   └── validator/    # Request validation
├── migrations/       # Database migrations
├── sql/
│   └── queries/      # SQLC query definitions
├── docs/             # Documentation
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── sqlc.yaml
```

## Commands

```bash
# Build
make build          # Build the API binary
make build-migrate  # Build the migration binary

# Run
make run            # Run the API server

# Test
make test           # Run all tests (unit + integration via testcontainers)
make test-unit      # Run unit tests only (excludes integration tests)
make test-integration # Run integration tests only (requires Docker)
make test-coverage  # Run tests with coverage report

# Lint & Format
make lint           # Run golangci-lint
make fmt            # Format code
make vet            # Run go vet

# Database
make migrate-up     # Run all pending migrations
make migrate-down   # Rollback last migration
make migrate-reset  # Reset database (down then up)
make migrate-version # Show current migration version

# SQLC
make sqlc-gen        # Generate sqlc code
make sqlc-compile   # Validate sqlc schema/queries without generating

# Docker
make docker-build   # Build Docker image

# CI/CD
make verify          # Run all verification steps (fmt, vet, lint, sqlc-compile, test)
make ci             # Run full CI pipeline

# Maintenance
make tidy            # Tidy go modules
make clean           # Clean build artifacts
make install-tools   # Install required tools
```

## Contributing

See [docs/contributing/](docs/contributing/) for development guidelines.

## License

MIT

## Documentation

- [Features](docs/features.md) — Feature inventory and status
- [API Reference](docs/api.md) — API endpoint documentation