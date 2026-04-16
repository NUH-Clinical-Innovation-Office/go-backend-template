.PHONY: help build build-migrate run test test-unit test-integration test-coverage lint fmt vet tidy clean install-tools sqlc-gen sqlc-compile migrate-up migrate-down migrate-reset migrate-version migrate-force docker-build verify ci

# Variables
BINARY_NAME=go-backend-template
MAIN_PATH=./cmd/api
MIGRATE_PATH=./cmd/migrate
DOCKER_IMAGE=go-backend-template:latest

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the API binary
	@go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

build-migrate: ## Build the migration binary
	@go build -o bin/migrate $(MIGRATE_PATH)

run: ## Run the API server
	@go run $(MAIN_PATH)

test: ## Run all tests (unit + integration via testcontainers)
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-unit: ## Run unit tests only (excludes integration tests)
	@go test -v -short -race -coverprofile=coverage.out -covermode=atomic ./...

test-integration: ## Run integration tests only (requires Docker - uses testcontainers)
	@go test -v -tags integration -race -timeout 120s ./internal/integration/...

test-coverage: test ## Run tests with coverage report
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint
	@golangci-lint run ./...

fmt: ## Format code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

tidy: ## Tidy go modules
	@go mod tidy

clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -f coverage.out coverage.html

install-tools: ## Install required tools
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0

sqlc-gen: ## Generate sqlc code (re-run after any new migration is added)
	@sqlc generate

sqlc-compile: ## Validate sqlc schema/queries without generating
	@sqlc compile

migrate-up: ## Run all pending migrations
	@go run $(MIGRATE_PATH) up

migrate-down: ## Rollback last migration
	@go run $(MIGRATE_PATH) down

migrate-reset: ## Reset database (down then up)
	@go run $(MIGRATE_PATH) reset

migrate-version: ## Show current migration version
	@go run $(MIGRATE_PATH) version

migrate-force: ## Force migration to specific version (requires VERSION=number)
	@go run $(MIGRATE_PATH) force -version $(VERSION)

docker-build: ## Build Docker image
	@docker build -t $(DOCKER_IMAGE) .

verify: fmt vet lint sqlc-compile test ## Run all verification steps
	@echo "All verification steps completed successfully!"

ci: verify ## Run CI pipeline
	@echo "CI pipeline completed successfully!"
