# Go Backend Template

A production-ready Go backend template with Chi router, sqlc, and OpenTelemetry.

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
- [Development](#development)
- [Observability](#observability)
- [Database](#database)
- [License](#license)

## Features

- **Chi Router**: Lightweight, idiomatic HTTP routing
- **sqlc**: Type-safe SQL code generation
- **PostgreSQL**: Primary database with pgx driver
- **JWT Authentication**: Secure token-based auth with bcrypt password hashing
- **Approved Users Gate**: Email whitelist for controlled registration
- **OpenTelemetry**: Distributed tracing with Jaeger/OTLP support
- **Zap Logging**: Structured, high-performance logging
- **Database Migrations**: Using golang-migrate
- **Docker Support**: Multi-stage builds and docker-compose
- **CORS Middleware**: Cross-origin request handling with configurable origins
- **Request ID Middleware**: Unique request ID per request for tracing
- **Real IP Middleware**: Extracts real client IP from proxy headers
- **Timeout Middleware**: 30-second request timeout protection
- **Integration Tests**: testcontainers-go for real database testing

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
│   ├── logging/      # Logging setup
│   ├── middleware/   # HTTP middleware
│   ├── observability/# OpenTelemetry setup
│   ├── router/       # Router configuration
│   └── todo/         # Todo feature (example CRUD)
├── migrations/       # Database migrations
├── sql/
│   └── queries/      # SQLC query definitions
├── docs/             # Generated documentation
├── code/             # Additional code samples
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── sqlc.yaml
```

## Quick Start

### Prerequisites

- Go 1.26+
- PostgreSQL 16+
- Docker and docker-compose (optional)

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

### Manual Docker Build

```bash
# Build and start
docker-compose up --build
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Server
SERVER_HOST=localhost
SERVER_PORT=8080

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/go_backend_template?sslmode=disable

# JWT
JWT_SECRET_KEY=your-super-secret-jwt-key-change-in-production
JWT_EXPIRE_MINUTES=1440

# Logging
LOG_LEVEL=info
LOG_FORMAT=console

# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=go-backend-template
OTEL_TRACES_SAMPLE_RATE=1.0
```

## API Endpoints

### Authentication

| Method | Endpoint                | Description       | Auth |
| ------ | ----------------------- | ----------------- | ---- |
| POST   | `/api/v1/auth/register` | Register new user | No   |
| POST   | `/api/v1/auth/login`    | Login             | No   |

### User (Authenticated)

| Method | Endpoint     | Description           | Auth     |
| ------ | ------------ | --------------------- | -------- |
| GET    | `/api/v1/me` | Get current user info | Required |

### Todos (User-Scoped)

| Method | Endpoint             | Description    | Auth     |
| ------ | -------------------- | -------------- | -------- |
| GET    | `/api/v1/todos`      | List all todos | Required |
| POST   | `/api/v1/todos`      | Create todo    | Required |
| GET    | `/api/v1/todos/{id}` | Get todo by ID | Required |
| PUT    | `/api/v1/todos/{id}` | Update todo    | Required |
| DELETE | `/api/v1/todos/{id}` | Delete todo    | Required |

### Admin (Approved Users Management)

| Method | Endpoint                            | Description          | Auth  |
| ------ | ----------------------------------- | -------------------- | ----- |
| GET    | `/api/v1/admin/approved-users`      | List approved users  | Admin |
| POST   | `/api/v1/admin/approved-users`      | Create approved user | Admin |
| POST   | `/api/v1/admin/approved-users/bulk` | Bulk create          | Admin |
| DELETE | `/api/v1/admin/approved-users/{id}` | Delete approved user | Admin |

### Health

| Method | Endpoint  | Description  | Auth |
| ------ | --------- | ------------ | ---- |
| GET    | `/health` | Health check | No   |
| GET    | `/`       | API info     | No   |

## Development

```bash
# Run all tests (unit + integration)
make test

# Run unit tests only
make test-unit

# Run integration tests only (requires Docker)
make test-integration

# Run linter
make lint

# Format code
make fmt

# Run go vet
make vet

# Generate SQLC code
make generate

# Build binaries
make build

# Verify (fmt, vet, lint, sqlc-compile, test)
make verify

# Run full CI pipeline
make ci

# Clean artifacts
make clean
```

## Observability

### Tracing

Traces are exported to Jaeger via OTLP. Access the UI at `http://localhost:16686`.

### Logging

Structured JSON logs (in production) with trace context:

```json
{
  "level": "info",
  "msg": "request completed",
  "timestamp": "2024-01-01T00:00:00Z",
  "trace_id": "abc123",
  "span_id": "def456",
  "method": "GET",
  "path": "/api/v1/todos",
  "status": 200
}
```

## Database

### Running Migrations

```bash
# Apply all migrations
make migrate-up

# Rollback one migration
make migrate-down

# Reset database
make migrate-reset

# Check version
make migrate-version
```

### SQLC

Queries are defined in `sql/queries/`. After modifying:

```bash
make generate
```

## License

MIT
