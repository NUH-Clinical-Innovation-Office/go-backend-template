// Package integration provides integration tests for the API.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/your-org/go-backend-template/internal/auth"
	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/db"
	dbSQLC "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/logging"
	"github.com/your-org/go-backend-template/internal/todo"
)

var (
	testContainer *postgres.PostgresContainer
	testConfig    *config.Config
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start PostgreSQL container
	var err error
	testContainer, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start PostgreSQL container: %v\n", err)
		os.Exit(1)
	}

	// Get connection string
	connStr, err := testContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	// Setup test config
	testConfig = &config.Config{
		Database: config.DatabaseConfig{
			URL:             connStr,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Auth: config.AuthConfig{
			JWTSecretKey:     "test-secret-key-for-integration-tests",
			JWTExpireMinutes: 60,
			BcryptCost:       4, // Fast for tests
		},
	}

	// Run migrations
	if err := runMigrations(connStr); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Cleanup
	if err := testContainer.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

func runMigrations(connStr string) error {
	// Simple migration runner - reads SQL files and executes them
	ctx := context.Background()
	pool, err := db.New(ctx, testConfig.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Read and execute migrations in order
	migrations := []string{
		"migrations/000001_create_approved_users.up.sql",
		"migrations/000002_create_users.up.sql",
		"migrations/000003_create_roles.up.sql",
		"migrations/000004_create_user_roles.up.sql",
		"migrations/000005_create_todos.up.sql",
	}

	for _, migration := range migrations {
		sql, err := os.ReadFile(migration)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migration, err)
		}

		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", migration, err)
		}
	}

	return nil
}

func setupTestDeps(t *testing.T) (*db.Pool, *dbSQLC.Queries, *auth.Service, *auth.Repository, *todo.Service, *todo.Repository, *auth.Handler, *todo.Handler) {
	t.Helper()

	ctx := context.Background()
	logger, _ := logging.New("debug", "console")

	pool, err := db.New(ctx, testConfig.Database)
	if err != nil {
		t.Fatalf("Failed to create db pool: %v", err)
	}

	queries := dbSQLC.New(pool.Pool)

	authRepo := auth.NewRepository(queries)
	authService := auth.NewService(authRepo, testConfig.Auth.JWTSecretKey, time.Duration(testConfig.Auth.JWTExpireMinutes)*time.Minute)
	authHandler := auth.NewHandler(authService, logger)

	todoRepo := todo.NewRepository(queries)
	todoService := todo.NewService(todoRepo)
	todoHandler := todo.NewHandler(todoService)

	return pool, queries, authService, authRepo, todoService, todoRepo, authHandler, todoHandler
}
