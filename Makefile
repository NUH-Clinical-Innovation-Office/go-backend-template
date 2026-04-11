.PHONY: help build test run migrate docker clean lint

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the API and migration binaries"
	@echo "  test         - Run tests"
	@echo "  run          - Run the API locally"
	@echo "  migrate-up   - Run all database migrations"
	@echo "  migrate-down - Rollback one migration"
	@echo "  migrate-reset - Reset database (down then up)"
	@echo "  docker-up    - Start all services with docker-compose"
	@echo "  docker-down  - Stop all services"
	@echo "  docker-logs  - Show docker-compose logs"
	@echo "  lint         - Run linter"
	@echo "  generate     - Generate SQLC code"
	@echo "  clean        - Remove build artifacts"

# Build the application
build:
	go build -o bin/api ./cmd/api
	go build -o bin/migrate ./cmd/migrate

# Run tests
test:
	go test -v -race ./...

# Run the API locally
run:
	go run ./cmd/api

# Database migrations
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-reset:
	go run ./cmd/migrate reset

migrate-force:
	go run ./cmd/migrate force $(VERSION)

migrate-version:
	go run ./cmd/migrate version

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-build:
	docker-compose build

# Linting
lint:
	golangci-lint run ./...

# Code generation
generate:
	sqlc generate

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Start development environment (database only)
dev-up:
	docker-compose up -d postgres

dev-down:
	docker-compose down
