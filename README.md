# Go Backend Template

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue)](https://golang.org/doc/go1.26)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](LICENSE)

A production-ready Go backend template with Chi router, sqlc, and OpenTelemetry.

## Table of Contents

- [Project Description](#project-description)
- [Renaming this template](#renaming-this-template)
- [CI/CD Setup](#cicd-setup)
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
| Integration Tests | testcontainers-go for real database testing |

## Renaming this template

The module path is `github.com/your-org/go-backend-template` as a
placeholder. To rename it for your fork:

```sh
make rename-org NEW_ORG=github.com/acme
```

This rewrites every Go import and runs `go mod tidy`. After it finishes,
update the placeholders in `.env.example` (`CORS_ALLOWED_ORIGINS`,
`SERVICE_NAME`, the OTel endpoint, etc.) and the `DOCKER_IMAGE` variable
in the Makefile. Commit the rename as the first change so it is easy to
revert.

## CI/CD Setup

The template ships self-contained `reusable-*.yml` workflows. If your project
uses the shared NUH pipeline (GHCR images plus AWS EKS), run the CI setup script
to generate workflows wired to the reusable workflows in
[`NUH-Clinical-Innovation-Office/ci-workflows`](https://github.com/NUH-Clinical-Innovation-Office/ci-workflows):

```sh
chmod +x scripts/setup-ci.sh
./scripts/setup-ci.sh
```

The script will:

- Read the module path from `go.mod` to default the project name
- Ask for the build architecture (`arm64` or `amd64`) and pick the matching runner
- Ask whether the module needs private NUH module access (`gh_private` plus a `GH_PAT` secret)
- Offer an integration test job, but only if the Makefile defines `test-integration`
- Offer an OpenAPI codegen drift and spec lint job, but only if `make openapi` and the spec exist
- Optionally add an AWS EKS development deploy job and a weekly GHCR cleanup workflow
- Show a configuration summary before writing anything

**What gets generated in `.github/workflows/`:**

| Workflow            | Trigger              | Purpose                                              |
| ------------------- | -------------------- | ---------------------------------------------------- |
| `ci.yml`            | Push to `main` + PRs | Lint, test, build; publish, scan, and deploy on push |
| `deploy.yml`        | Called by `ci.yml`   | Helm deploy to AWS EKS (only if deploy enabled)      |
| `image-cleanup.yml` | Weekly cron          | Prune old GHCR image versions (optional)             |

Unlike the frontend template, a single `ci.yml` handles both `push` and
`pull_request`. Jobs that publish images or deploy are gated on
`github.event_name == 'push' && github.ref == 'refs/heads/main'`, so pull
requests run checks only and never touch the registry or the cluster.

For how the shared pipeline works — tag pinning, image tag outputs, OIDC
deploys, and PR isolation — see the
[`ci-workflows` README](https://github.com/NUH-Clinical-Innovation-Office/ci-workflows#how-the-generated-pipeline-works).

Because the shared pipeline replaces the bundled `reusable-*.yml` workflows, the
script moves the existing ones to `.github/workflows-backup/` first — nothing is
deleted. Delete that directory once you are satisfied with the new pipeline.

**Before the first run**, in your GitHub repository settings:

- Allow Actions to access the `ci-workflows` repository (Settings → Actions → General)
- Set `GH_PAT` if you enabled private Go module access
- If you enabled deploys, set `AWS_DEV_DEPLOY_ROLE_ARN` (OIDC role) and optionally `IRSA_ROLE_ARN`
- If you enabled deploys, create a `development` environment (Settings → Environments)

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
                    │  Middleware: CORS, Auth, Timeout   │
                    └──────────────┬──────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
        ▼                          ▼                          ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│  Auth Handler │         │ Todo Handler   │         │ Admin Handler  │
└───────┬───────┘         └───────┬───────┘         └───────┬───────┘
        │                          │                          │
        ▼                          ▼                          ▼
┌───────────────┐         ┌───────────────┐
│   Auth Svc    │         │  Todo Svc     │
└───────┬───────┘         └───────┬───────┘
        │                          │
        ▼                          ▼
┌───────────────┐         ┌───────────────┐
│  SQLC Queries │         │  SQLC Queries │
└───────┬───────┘         └───────┬───────┘
        │                          │
        ▼                          ▼
┌─────────────────────────────────────────────┐
│              PostgreSQL Database             │
└─────────────────────────────────────────────┘
```

- **Router**: Chi mux with middleware chain for CORS, auth, request ID, real IP, timeout, and logging
- **Handlers**: HTTP request handlers delegate to services
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

Copyright 2025 NUH Department of Medicine

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.

## Documentation

- [Features](docs/features.md) — Feature inventory and status
- [API Reference](docs/api.md) — API endpoint documentation