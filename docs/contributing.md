# Contributing

## Development Setup

### Prerequisites

- Go 1.23+
- PostgreSQL 16+ (or Docker for containerized development)
- [golangci-lint](https://golangci-lint.run/usage/install/)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)

### Initial Setup

```bash
# Clone the repository
git clone <repository-url>
cd go-backend-template

# Install required tools
make install-tools

# Copy environment configuration
cp .env.example .env

# Run database migrations
make migrate-up
```

### Development Workflow

```bash
# Start the API server
make run

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Code Generation

After modifying SQL queries in `sql/queries/`, regenerate the database layer:

```bash
make sqlc-gen
```

After creating new migrations in `migrations/`, apply them:

```bash
make migrate-up
```

## Project Structure

```
cmd/
  api/          # API server entrypoint
  migrate/      # Migration CLI tool
internal/
  auth/         # Authentication (handler, service, repository)
  config/       # Configuration loading
  db/           # Database connection and sqlc generated code
  domain/       # Shared domain models
  http/         # HTTP utilities
  logging/      # Logging setup
  middleware/   # HTTP middleware (auth, CORS, logging, etc.)
  observability/# OpenTelemetry setup
  router/       # Router configuration
  todo/         # Todo feature (example CRUD)
  validator/    # Request validation utilities
migrations/     # Database migration files
sql/
  queries/      # sqlc query definitions
```

## Testing

### Unit Tests

```bash
make test-unit
```

### Integration Tests

Integration tests use testcontainers and require Docker:

```bash
make test-integration
```

### All Tests with Coverage

```bash
make test-coverage
```

## Code Standards

- All code must pass `make verify` before committing
- Run `make lint` to check for style and safety issues
- Use `make fmt` to format code before committing
- All new code should include unit tests
- Integration tests are required for new API endpoints

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Please format your commits accordingly:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Opening Pull Requests

1. Create a feature branch from `main`
2. Make your changes and add tests
3. Run `make verify` to ensure all checks pass
4. Push and open a Pull Request against `main`
5. Ensure CI passes
6. Request review from maintainers

## Getting Help

If you encounter issues or have questions:

1. Check the [API documentation](./api.md)
2. Review the [features documentation](./features.md)
3. Open an issue for bugs or feature requests