//go:build integration

// Package integration provides integration tests for the API.
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	// Check if using external database (e.g., from docker-compose in Makefile)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		// Use external database - migrations should already be run
		testConfig = &config.Config{
			Database: config.DatabaseConfig{
				URL:             dbURL,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			Auth: config.AuthConfig{
				JWTSecretKey:     os.Getenv("JWT_SECRET"),
				JWTExpireMinutes: 60,
				BcryptCost:       4, // Fast for tests
			},
		}

		// Verify database connection and migrations
		if err := verifyDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "Database verification failed: %v\n", err)
			os.Exit(1)
		}

		code := m.Run()
		os.Exit(code)
		return
	}

	// Use testcontainers for standalone test runs
	if err := setupTestContainer(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup test container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Cleanup
	if err := testContainer.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

func verifyDatabase() error {
	ctx := context.Background()
	pool, err := db.New(ctx, testConfig.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Simple query to verify connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

func setupTestContainer(ctx context.Context) error {
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
		return fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	// Get connection string
	connStr, err := testContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
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
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runMigrations(connStr string) error {
	return runMigrationsWithConnStr(connStr)
}

func runMigrationsWithConnStr(connStr string) error {
	// Simple migration runner - reads SQL files and executes them
	ctx := context.Background()

	// Create temporary config for migrations
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             connStr,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}

	pool, err := db.New(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Get repo root using caller file path (works from any test directory)
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	// Read and execute migrations in order
	migrations := []string{
		"migrations/000001_create_approved_users.up.sql",
		"migrations/000002_create_users.up.sql",
		"migrations/000003_create_roles.up.sql",
		"migrations/000004_create_user_roles.up.sql",
		"migrations/000005_create_todos.up.sql",
		"migrations/000006_seed_data.up.sql",
	}

	for _, migration := range migrations {
		migrationPath := filepath.Join(repoRoot, migration)
		sql, err := os.ReadFile(migrationPath)
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
	authService := auth.NewService(authRepo, testConfig.Auth.JWTSecretKey, time.Duration(testConfig.Auth.JWTExpireMinutes)*time.Minute, 4)
	authHandler := auth.NewHandler(authService, logger)

	todoRepo := todo.NewRepository(queries)
	todoService := todo.NewService(todoRepo)
	todoHandler := todo.NewHandler(todoService)

	return pool, queries, authService, authRepo, todoService, todoRepo, authHandler, todoHandler
}
