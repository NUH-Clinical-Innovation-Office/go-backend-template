# Contributing

Contributions are welcome. Please follow these guidelines:

## Prerequisites

- Go 1.26+
- golangci-lint
- sqlc

## Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make verify` to ensure all checks pass
5. Submit a pull request

## Code Standards

- Follow Go idioms and effective Go guidelines
- Run `make fmt` before committing
- Ensure `make vet` passes
- All new code should have appropriate tests

## Testing

- Unit tests: `make test-unit`
- Integration tests: `make test-integration` (requires Docker)
- Full test suite: `make test`

## Database Changes

1. Add migration files to `migrations/`
2. Update sqlc queries in `sql/queries/`
3. Run `make sqlc-gen` to regenerate code
4. Verify with `make sqlc-compile`

## Commit Messages

Follow conventional commits:
- `feat: add new endpoint`
- `fix: resolve auth bug`
- `docs: update README`
- `refactor: simplify router setup`