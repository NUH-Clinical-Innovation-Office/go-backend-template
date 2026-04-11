# Go Backend Template Implementation Plan

**Goal:** Build a reusable Go backend template with Chi, sqlc, PostgreSQL, auth (approved-users gate), and user-scoped todo CRUD.

**Architecture:** Feature-first layered structure. Each feature (auth, todo) has handler/service/repository/models. Shared infrastructure in internal/{config,db,domain,http,logging,middleware,observability,router}.

**Tech Stack:** Go 1.26, chi/v5, pgx/v5, sqlc, golang-migrate, jwt/jwt/v5, bcrypt, zap, OpenTelemetry, testify, testcontainers-go.

---

## Files to Create

**Core infrastructure (13 files):**
- cmd/api/main.go
- cmd/migrate/main.go
- internal/config/config.go
- internal/db/db.go
- internal/db/dbutil/convert.go
- internal/domain/user.go
- internal/http/response.go
- internal/logging/logger.go
- internal/middleware/auth.go
- internal/middleware/http.go
- internal/observability/otel.go
- internal/router/router.go
- go.mod

**Auth feature (6 files):**
- internal/auth/handler.go
- internal/auth/service.go
- internal/auth/repository.go
- internal/auth/models.go
- internal/auth/handler_test.go
- internal/auth/service_test.go

**Todo feature (6 files):**
- internal/todo/handler.go
- internal/todo/service.go
- internal/todo/repository.go
- internal/todo/models.go
- internal/todo/handler_test.go
- internal/todo/service_test.go

**Integration tests (4 files):**
- internal/integration/main_test.go
- internal/integration/auth_test.go
- internal/integration/todo_test.go
- internal/integration/helpers_test.go

**Migrations (10 files):**
- migrations/000001_create_extensions.{up,down}.sql
- migrations/000002_create_enum_types.{up,down}.sql
- migrations/000003_create_approved_users_table.{up,down}.sql
- migrations/000004_create_users_table.{up,down}.sql
- migrations/000005_create_roles_tables.{up,down}.sql
- migrations/000006_create_todos_table.{up,down}.sql

**SQL queries (2 files):**
- sql/queries/auth.sql
- sql/queries/todo.sql

**Infrastructure (11 files):**
- .env.example
- .golangci.yml
- .commitlintrc.yml
- .dockerignore
- .gitignore
- docker-compose.yml
- Dockerfile
- lefthook.yml
- Makefile
- sqlc.yaml
- README.md

---

## Task 1: Module & Core Infrastructure

**Files to create/modify:**
- Create: go.mod
- Create: cmd/api/main.go
- Create: cmd/migrate/main.go
- Create: internal/config/config.go

- [ ] **Step 1: Create go.mod**

```go
module github.com/your-org/go-backend-template

go 1.26.1

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/joho/godotenv v1.5.1
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.49.0
	golang.org/x/time v0.15.0
)
```

- [ ] **Step 2: Run go mod tidy**

```bash
go mod tidy
```

Expected: Downloads dependencies, creates go.sum

- [ ] **Step 3: Create cmd/api/main.go**

```go
// Command api is the main entry point for the API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/db"
	"github.com/your-org/go-backend-template/internal/logging"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	logger, err := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("starting go-backend-template API")

	// Connect to database
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	logger.Info("database connection established")

	// TODO: Wire dependencies and build router
	_ = pool
	_ = logger

	// Start HTTP server
	return startHTTPServer(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World"))
	}), logger)
}

func startHTTPServer(cfg *config.Config, handler http.Handler, logger *zap.Logger) error {
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("HTTP server starting",
			zap.String("addr", server.Addr),
			zap.Int("port", cfg.Server.Port),
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
			return fmt.Errorf("graceful shutdown: %w", err)
		}

		logger.Info("HTTP server stopped")
		return nil
	}
}
```

- [ ] **Step 4: Create cmd/migrate/main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	command := flag.String("command", "up", "Migration command (up, down, reset, force, version)")
	version := flag.Int("version", 0, "Target version for force command")
	migrationsPath := flag.String("path", "file://migrations", "Path to migrations directory")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	m, err := migrate.New(*migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			fmt.Fprintf(os.Stderr, "warning: error closing migrate: src=%v, db=%v\n", srcErr, dbErr)
		}
	}()

	switch *command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migration up failed: %w", err)
		}
		fmt.Println("migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migration down failed: %w", err)
		}
		fmt.Println("migration down completed")
	case "reset":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migration down failed: %w", err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migration up failed: %w", err)
		}
		fmt.Println("migration reset completed")
	case "force":
		if *version == 0 {
			return fmt.Errorf("-version flag is required for force command")
		}
		if err := m.Force(*version); err != nil {
			return fmt.Errorf("migration force failed: %w", err)
		}
		fmt.Printf("migration forced to version %d\n", *version)
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		if dirty {
			fmt.Printf("version: %d (dirty)\n", v)
		} else {
			fmt.Printf("version: %d\n", v)
		}
	default:
		return fmt.Errorf("unknown command: %s", *command)
	}

	return nil
}
```

- [ ] **Step 5: Create internal/config/config.go**

```go
// Package config handles application configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Auth          AuthConfig
	Logging       LoggingConfig
	Observability ObservabilityConfig
	RateLimit     RateLimitConfig
	CORS          CORSConfig
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWTSecretKey     string
	JWTExpireMinutes int
	Algorithm        string
	BcryptCost       int
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string
	Format string
}

// ObservabilityConfig contains observability settings
type ObservabilityConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	SamplingRatio  float64
	TracingEnabled bool
	OTLPInsecure   bool
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Requests int
	Duration time.Duration
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	p := &envParser{}

	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            p.int("SERVER_PORT", 8080),
			ReadTimeout:     p.duration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    p.duration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     p.duration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: p.duration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    p.int("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    p.int("DATABASE_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: p.duration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Auth: AuthConfig{
			JWTSecretKey:     getEnv("JWT_SECRET_KEY", ""),
			JWTExpireMinutes: p.int("JWT_EXPIRE_MINUTES", 1440),
			Algorithm:        "HS256",
			BcryptCost:       p.int("BCRYPT_COST", 12),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Observability: ObservabilityConfig{
			TracingEnabled: p.bool("TRACING_ENABLED", true),
			ServiceName:    getEnv("SERVICE_NAME", "go-backend-template"),
			ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
			Environment:    getEnv("ENVIRONMENT", "development"),
			SamplingRatio:  p.float("OTEL_TRACE_SAMPLING_RATIO", 1.0),
			OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
			OTLPInsecure:   p.bool("OTEL_EXPORTER_OTLP_INSECURE", true),
		},
		RateLimit: RateLimitConfig{
			Requests: p.int("RATE_LIMIT_REQUESTS", 10),
			Duration: p.duration("RATE_LIMIT_DURATION", time.Minute),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getCommaSeparatedEnv("CORS_ALLOWED_ORIGINS", []string{}),
			AllowedMethods:   getCommaSeparatedEnv("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getCommaSeparatedEnv("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type"}),
			AllowCredentials: p.bool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           p.int("CORS_MAX_AGE", 3600),
		},
	}

	if p.err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", p.err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// envParser accumulates the first parse error encountered.
type envParser struct {
	err error
}

func (p *envParser) int(key string, defaultValue int) int {
	if p.err != nil {
		return defaultValue
	}
	value, err := getEnvAsIntOrError(key, defaultValue)
	if err != nil {
		p.err = err
	}
	return value
}

func (p *envParser) bool(key string, defaultValue bool) bool {
	if p.err != nil {
		return defaultValue
	}
	value, err := getEnvAsBoolOrError(key, defaultValue)
	if err != nil {
		p.err = err
	}
	return value
}

func (p *envParser) duration(key string, defaultValue time.Duration) time.Duration {
	if p.err != nil {
		return defaultValue
	}
	value, err := getEnvAsDurationOrError(key, defaultValue)
	if err != nil {
		p.err = err
	}
	return value
}

func (p *envParser) float(key string, defaultValue float64) float64 {
	if p.err != nil {
		return defaultValue
	}
	value, err := getEnvAsFloatOrError(key, defaultValue)
	if err != nil {
		p.err = err
	}
	return value
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Auth.JWTSecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: %d", c.Server.Port)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrError(key string, defaultValue int) (int, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue, fmt.Errorf("%s=%q is not a valid integer", key, valueStr)
	}
	return value, nil
}

func getEnvAsBoolOrError(key string, defaultValue bool) (bool, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue, fmt.Errorf("%s=%q is not a valid boolean", key, valueStr)
	}
	return value, nil
}

func getEnvAsDurationOrError(key string, defaultValue time.Duration) (time.Duration, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue, fmt.Errorf("%s=%q is not a valid duration", key, valueStr)
	}
	return value, nil
}

func getEnvAsFloatOrError(key string, defaultValue float64) (float64, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue, fmt.Errorf("%s=%q is not a valid float", key, valueStr)
	}
	return value, nil
}

func getCommaSeparatedEnv(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	parts := strings.Split(valueStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}
```

- [ ] **Step 6: Verify compiles**

```bash
go build ./cmd/api ./cmd/migrate
```

Expected: Two binaries created, no errors

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum cmd/ internal/config/
git commit -m "chore: add module definition and core infrastructure stub"
```

---

## Task 2: Database & Shared Infrastructure

**Files:**
- Create: internal/db/db.go
- Create: internal/db/dbutil/convert.go
- Create: internal/domain/user.go
- Create: internal/http/response.go
- Create: internal/logging/logger.go

- [ ] **Step 1: Create internal/db/db.go**

```go
// Package db provides database connection pooling.
package db

import (
	"context"
	"fmt"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool
type Pool struct {
	*pgxpool.Pool
}

// New creates a new database connection pool
func New(ctx context.Context, cfg config.DatabaseConfig) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close closes the database pool
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
```

- [ ] **Step 2: Create internal/db/dbutil/convert.go**

```go
// Package dbutil provides shared conversion helpers.
package dbutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// PgUUIDToPtr converts a pgtype.UUID to *uuid.UUID
func PgUUIDToPtr(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	u := uuid.UUID(p.Bytes)
	return &u
}

// UUIDToPgtype converts *uuid.UUID to pgtype.UUID
func UUIDToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// PgTimestamptzToPtr converts pgtype.Timestamptz to *time.Time
func PgTimestamptzToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// PgInt4ToPtr converts pgtype.Int4 to *int
func PgInt4ToPtr(p pgtype.Int4) *int {
	if !p.Valid {
		return nil
	}
	v := int(p.Int32)
	return &v
}

// IntToPgInt4 converts *int to pgtype.Int4
func IntToPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}
```

- [ ] **Step 3: Create internal/domain/user.go**

```go
// Package domain provides shared domain models.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID             uuid.UUID
	ApprovedUserID uuid.UUID
	HashedPassword string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Eager loaded
	ApprovedUser *ApprovedUser
	Roles        []Role
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// ApprovedUser represents an approved user who can create accounts
type ApprovedUser struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	CreatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Role represents a user role
type Role struct {
	ID          uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
}
```

- [ ] **Step 4: Create internal/http/response.go**

```go
// Package http provides HTTP response helpers.
package http

import (
	"encoding/json"
	"net/http"
)

// RespondJSON writes a JSON response
func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// RespondError writes a JSON error response
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"detail": message})
}
```

- [ ] **Step 5: Create internal/logging/logger.go**

```go
// Package logging provides structured logging utilities.
package logging

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new structured logger
func New(level, format string) (*zap.Logger, error) {
	zapLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var config zap.Config
	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build(
		zap.AddCallerSkip(0),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// WithTraceContext adds trace context to the logger
func WithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	spanContext := span.SpanContext()
	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}
```

- [ ] **Step 6: Verify compiles**

```bash
go build ./...
```

Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add internal/db/ internal/domain/ internal/http/ internal/logging/
git commit -m "feat: add database pool and shared infrastructure"
```

---

## Task 3: Middleware & Observability

**Files:**
- Create: internal/middleware/auth.go
- Create: internal/middleware/http.go
- Create: internal/observability/otel.go

- [ ] **Step 1: Create internal/middleware/http.go**

```go
// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/your-org/go-backend-template/internal/domain"
	"golang.org/x/time/rate"
)

type contextKey string

const (
	CurrentUserKey contextKey = "current_user"
	RequestIDKey   contextKey = "request_id"
)

// UserFromContext retrieves the current user from context
func UserFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(CurrentUserKey).(*domain.User)
	return u
}

// RequestIDFromContext retrieves the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// extractBearerToken extracts Bearer token from Authorization header
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}

// RateLimiter wraps a rate limiter
type RateLimiter struct {
	rate *rate.Limiter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		rate: rate.NewLimiter(rate.Every(duration), requests),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	return rl.rate.Allow()
}
```

- [ ] **Step 2: Create internal/middleware/auth.go**

```go
package middleware

import (
	"context"
	"net/http"

	"github.com/your-org/go-backend-template/internal/domain"
	http2 "github.com/your-org/go-backend-template/internal/http"
)

// AuthProvider defines the interface for auth service operations
type AuthProvider interface {
	GetUserFromToken(ctx context.Context, token string) (*domain.User, error)
}

// RequireAuth validates JWT Bearer token and injects user into context
func RequireAuth(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				http2.RespondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			user, err := authSvc.GetUserFromToken(r.Context(), token)
			if err != nil || user == nil {
				http2.RespondError(w, http.StatusUnauthorized, "could not validate credentials")
				return
			}

			ctx := context.WithValue(r.Context(), CurrentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin wraps RequireAuth and checks for admin role
func RequireAdmin(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RequireAuth(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http2.RespondError(w, http.StatusUnauthorized, "could not validate credentials")
				return
			}

			if !user.HasRole("admin") {
				http2.RespondError(w, http.StatusForbidden, "admin role required")
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// OptionalAuth extracts Bearer token if present but doesn't reject missing tokens
func OptionalAuth(authSvc AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authSvc.GetUserFromToken(r.Context(), token)
			if err != nil || user == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), CurrentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 3: Create internal/observability/otel.go**

```go
// Package observability provides OpenTelemetry integration.
package observability

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
)

// Setup initializes OpenTelemetry for distributed tracing
func Setup(ctx context.Context, serviceName string, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(getEnv("SERVICE_VERSION", "1.0.0")),
			semconv.DeploymentEnvironment(getEnv("ENVIRONMENT", "development")),
		),
	)
	if err != nil {
		logger.Warn("failed to create resource with auto-detection, using basic resource", zap.Error(err))
		res, err = resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceName(serviceName),
				semconv.ServiceVersion(getEnv("SERVICE_VERSION", "1.0.0")),
				semconv.DeploymentEnvironment(getEnv("ENVIRONMENT", "development")),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create basic resource: %w", err)
		}
	}

	exporterOptions := []otlptracehttp.Option{
		otlptracehttp.WithTimeout(10 * time.Second),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			MaxElapsedTime:  5 * time.Minute,
		}),
	}

	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		exporterOptions = append(exporterOptions, otlptracehttp.WithEndpoint(endpoint))
		logger.Info("using custom OTLP endpoint", zap.String("endpoint", endpoint))
	}

	exporter, err := otlptracehttp.New(ctx, exporterOptions...)
	if err != nil {
		logger.Warn("failed to create OTLP trace exporter, tracing will be disabled", zap.Error(err))
		return sdktrace.NewTracerProvider(), nil
	}

	samplingRatio := getSamplingRatio()
	logger.Info("trace sampling configured", zap.Float64("ratio", samplingRatio))

	batchProcessor := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithMaxQueueSize(2048),
		sdktrace.WithBatchTimeout(5*time.Second),
		sdktrace.WithMaxExportBatchSize(512),
		sdktrace.WithExportTimeout(30*time.Second),
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(batchProcessor),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(samplingRatio),
		)),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry tracing initialized successfully",
		zap.String("service_name", serviceName),
		zap.Float64("sampling_ratio", samplingRatio),
	)

	return tracerProvider, nil
}

func getSamplingRatio() float64 {
	defaultRatio := 0.1
	if env := os.Getenv("ENVIRONMENT"); env == "development" || env == "dev" || env == "" {
		defaultRatio = 1.0
	}

	ratio := float64(defaultRatio)
	if ratioStr := os.Getenv("OTEL_TRACE_SAMPLING_RATIO"); ratioStr != "" {
		if _, err := fmt.Sscanf(ratioStr, "%f", &ratio); err != nil {
			return float64(defaultRatio)
		}
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
	}

	return ratio
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

- [ ] **Step 4: Verify compiles**

```bash
go build ./...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/ internal/observability/
git commit -m "feat: add middleware and observability"
```

---

## Task 4: Router

**Files:**
- Create: internal/router/router.go

- [ ] **Step 1: Create internal/router/router.go**

```go
// Package router provides HTTP router setup.
package router

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/domain"
	http2 "github.com/your-org/go-backend-template/internal/http"
	"github.com/your-org/go-backend-template/internal/logging"
	"github.com/your-org/go-backend-template/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

// RouterConfig holds dependencies for router setup
type RouterConfig struct {
	AuthSvc     middleware.AuthProvider
	Logger      *zap.Logger
	AppConfig   *config.Config
	RateLimiter *middleware.RateLimiter
}

// New creates a new chi router with all routes configured
func New(cfg *RouterConfig) http.Handler {
	r := chi.NewMux()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware(cfg.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware(&cfg.AppConfig.CORS))

	// Root and health endpoints
	r.Get("/", rootHandler(cfg.AppConfig.Observability.ServiceVersion))
	r.Get("/health", healthHandler)

	// TODO: Auth routes
	// TODO: Admin routes
	// TODO: Todo routes

	return r
}

func loggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			ctx := r.Context()
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
			r = r.WithContext(ctx)

			tracedLogger := logging.WithTraceContext(ctx, logger)

			next.ServeHTTP(wrapped, r)

			tracedLogger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.status),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

func corsMiddleware(cfg *config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			allowed := false
			if len(cfg.AllowedOrigins) == 0 {
				allowed = true
			} else {
				for _, o := range cfg.AllowedOrigins {
					if o == origin || o == "*" {
						allowed = true
						break
					}
				}
			}

			if allowed {
				specificOrigins := len(cfg.AllowedOrigins) > 0
				if specificOrigins {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}

				if len(cfg.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				}

				if len(cfg.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
				}

				if cfg.AllowCredentials && specificOrigins {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func rootHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http2.RespondJSON(w, http.StatusOK, map[string]string{
			"message": "go-backend-template API",
			"version": version,
		})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	http2.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./...
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/router/
git commit -m "feat: add chi router with middleware stack"
```

---

## Task 5: Migrations

**Files:**
- Create: migrations/000001_create_extensions.{up,down}.sql
- Create: migrations/000002_create_enum_types.{up,down}.sql
- Create: migrations/000003_create_approved_users_table.{up,down}.sql
- Create: migrations/000004_create_users_table.{up,down}.sql
- Create: migrations/000005_create_roles_tables.{up,down}.sql
- Create: migrations/000006_create_todos_table.{up,down}.sql

- [ ] **Step 1: Create 000001_create_extensions.up.sql**

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

- [ ] **Step 2: Create 000001_create_extensions.down.sql**

```sql
DROP EXTENSION IF EXISTS pgcrypto;
```

- [ ] **Step 3: Create 000002_create_enum_types.up.sql**

```sql
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('admin', 'user');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;
```

- [ ] **Step 4: Create 000002_create_enum_types.down.sql**

```sql
DROP TYPE IF EXISTS user_role;
```

- [ ] **Step 5: Create 000003_create_approved_users_table.up.sql**

```sql
CREATE TABLE approved_users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT approved_users_email_key UNIQUE (email)
);
```

- [ ] **Step 6: Create 000003_create_approved_users_table.down.sql**

```sql
DROP TABLE IF EXISTS approved_users;
```

- [ ] **Step 7: Create 000004_create_users_table.up.sql**

```sql
CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    approved_user_id UUID        NOT NULL UNIQUE REFERENCES approved_users (id) ON DELETE RESTRICT,
    hashed_password  VARCHAR(255) NOT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_approved_user_id ON users (approved_user_id);

ALTER TABLE approved_users
    ADD COLUMN created_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL;

CREATE INDEX idx_approved_users_created_by ON approved_users (created_by_user_id);
```

- [ ] **Step 8: Create 000004_create_users_table.down.sql**

```sql
DROP INDEX IF EXISTS idx_approved_users_created_by;
ALTER TABLE approved_users DROP COLUMN IF EXISTS created_by_user_id;
DROP INDEX IF EXISTS idx_users_approved_user_id;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 9: Create 000005_create_roles_tables.up.sql**

```sql
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'Administrator with full access'),
    ('user', 'Regular user');

CREATE TABLE user_roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_roles_user_id_role_key UNIQUE (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);
```

- [ ] **Step 10: Create 000005_create_roles_tables.down.sql**

```sql
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
```

- [ ] **Step 11: Create 000006_create_todos_table.up.sql**

```sql
CREATE TABLE todos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      VARCHAR(255) NOT NULL,
    completed  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_todos_user_id ON todos (user_id);
```

- [ ] **Step 12: Create 000006_create_todos_table.down.sql**

```sql
DROP TABLE IF EXISTS todos;
```

- [ ] **Step 13: Commit**

```bash
git add migrations/
git commit -m "feat: add database migrations"
```

---

## Task 6: SQLC Queries

**Files:**
- Create: sqlc.yaml
- Create: sql/queries/auth.sql
- Create: sql/queries/todo.sql

- [ ] **Step 1: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations/"
    queries: "sql/queries/"
    gen:
      go:
        package: "sqlcdb"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_interface: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
```

- [ ] **Step 2: Create sql/queries/auth.sql**

```sql
-- name: IsEmailApproved :one
SELECT 1 FROM approved_users WHERE email = $1 LIMIT 1;

-- name: GetApprovedUserByEmail :one
SELECT id, email, first_name, created_by_user_id, created_at, updated_at
FROM approved_users
WHERE email = $1;

-- name: GetUserByApprovedUserID :one
SELECT id FROM users WHERE approved_user_id = $1;

-- name: InsertUser :one
INSERT INTO users (approved_user_id, hashed_password, is_active, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING id, approved_user_id, hashed_password, is_active, created_at, updated_at;

-- name: GetUserWithApprovedUserByEmail :one
SELECT
    u.id, u.approved_user_id, u.hashed_password, u.is_active, u.created_at, u.updated_at,
    au.email, au.first_name, au.created_by_user_id, au.created_at AS au_created_at, au.updated_at AS au_updated_at
FROM users u
JOIN approved_users au ON au.id = u.approved_user_id
WHERE au.email = $1;

-- name: GetUserWithApprovedUserByID :one
SELECT
    u.id, u.approved_user_id, u.hashed_password, u.is_active, u.created_at, u.updated_at,
    au.email, au.first_name, au.created_by_user_id, au.created_at AS au_created_at, au.updated_at AS au_updated_at
FROM users u
JOIN approved_users au ON au.id = u.approved_user_id
WHERE u.id = $1;

-- name: GetRolesByUserID :many
SELECT r.id, r.name, r.description, r.created_at
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET hashed_password = $1, updated_at = NOW()
WHERE id = $2;

-- name: InsertApprovedUser :one
INSERT INTO approved_users (email, first_name, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
RETURNING id, email, first_name, created_at, updated_at;

-- name: ListApprovedUsers :many
SELECT id, email, first_name, created_at, updated_at
FROM approved_users
ORDER BY created_at DESC;
```

- [ ] **Step 3: Create sql/queries/todo.sql**

```sql
-- name: ListTodosByUserID :many
SELECT id, user_id, title, completed, created_at, updated_at
FROM todos
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetTodoByID :one
SELECT id, user_id, title, completed, created_at, updated_at
FROM todos
WHERE id = $1 AND user_id = $2;

-- name: InsertTodo :one
INSERT INTO todos (user_id, title, completed, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING id, user_id, title, completed, created_at, updated_at;

-- name: UpdateTodo :one
UPDATE todos
SET title = $1, completed = $2, updated_at = NOW()
WHERE id = $3 AND user_id = $4
RETURNING id, user_id, title, completed, updated_at;

-- name: DeleteTodo :exec
DELETE FROM todos
WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 4: Run sqlc generate**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```

Expected: Generates internal/db/sqlc/*.go files

- [ ] **Step 5: Verify compiles**

```bash
go build ./...
```

Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add sqlc.yaml sql/ internal/db/sqlc/
git commit -m "feat: add sqlc queries and generate code"
```

---

## Task 7: Auth Feature

**Files:**
- Create: internal/auth/handler.go
- Create: internal/auth/service.go
- Create: internal/auth/repository.go
- Create: internal/auth/models.go
- Create: internal/auth/handler_test.go
- Create: internal/auth/service_test.go

- [ ] **Step 1: Create internal/auth/models.go**

```go
// Package auth provides authentication and authorization functionality.
package auth

import (
	"time"

	"github.com/your-org/go-backend-template/internal/domain"
)

// SignupRequest represents a user signup request
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// TokenResponse represents a JWT token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// UserResponse represents user information in API responses
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

// ApprovedUserRequest represents an approved user creation request
type ApprovedUserRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

// ApprovedUserResponse represents an approved user in API responses
type ApprovedUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserResponse(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}

	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	return &UserResponse{
		ID:        user.ID.String(),
		Email:     user.ApprovedUser.Email,
		FirstName: user.ApprovedUser.FirstName,
		IsActive:  user.IsActive,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
	}
}

func toApprovedUserResponse(au *domain.ApprovedUser) *ApprovedUserResponse {
	if au == nil {
		return nil
	}
	return &ApprovedUserResponse{
		ID:        au.ID.String(),
		Email:     au.Email,
		FirstName: au.FirstName,
		CreatedAt: au.CreatedAt,
	}
}

func toApprovedUserResponses(aus []*domain.ApprovedUser) []*ApprovedUserResponse {
	result := make([]*ApprovedUserResponse, 0, len(aus))
	for _, au := range aus {
		result = append(result, toApprovedUserResponse(au))
	}
	return result
}
```

- [ ] **Step 2: Create internal/auth/service.go**

```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ServiceInterface defines the interface for auth service operations
type ServiceInterface interface {
	Signup(ctx context.Context, input SignupInput) (*domain.User, error)
	Login(ctx context.Context, input LoginInput) (string, error)
	ChangePassword(ctx context.Context, input ChangePasswordInput) error
	GetUserFromToken(ctx context.Context, token string) (*domain.User, error)
	CreateApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error)
	ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error)
}

// RepositoryInterface defines the interface for auth repository operations
type RepositoryInterface interface {
	IsEmailApproved(ctx context.Context, email string) (bool, error)
	GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error)
	GetUserByApprovedUserID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	InsertUser(ctx context.Context, approvedUserID uuid.UUID, hashedPassword string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateUserPassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	InsertApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error)
	ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error)
}

// Service handles authentication business logic
type Service struct {
	repo   RepositoryInterface
	config config.AuthConfig
}

// NewService creates a new auth service
func NewService(repo RepositoryInterface, cfg config.AuthConfig) *Service {
	return &Service{
		repo:   repo,
		config: cfg,
	}
}

// Error types
var (
	ErrEmailNotApproved   = errors.New("email not approved")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidPassword    = errors.New("current password is incorrect")
	ErrInvalidToken       = errors.New("invalid token")
	ErrEmailAlreadyExists = errors.New("email already in approved list")
)

// SignupInput represents the input for a signup operation
type SignupInput struct {
	Email    string
	Password string
}

// Signup signs up a new user. Only users with approved emails can register.
func (s *Service) Signup(ctx context.Context, input SignupInput) (*domain.User, error) {
	input.Email = strings.ToLower(input.Email)

	// Check if email is approved
	approved, err := s.repo.IsEmailApproved(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email approval: %w", err)
	}
	if !approved {
		return nil, ErrEmailNotApproved
	}

	// Get approved user record
	approvedUser, err := s.repo.GetApprovedUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("get approved user: %w", err)
	}

	// Check if user already registered
	_, err = s.repo.GetUserByApprovedUserID(ctx, approvedUser.ID)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	user, err := s.repo.InsertUser(ctx, approvedUser.ID, string(hashedPassword))
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

// LoginInput represents the input for a login operation
type LoginInput struct {
	Email    string
	Password string
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, input LoginInput) (string, error) {
	input.Email = strings.ToLower(input.Email)

	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	if !user.IsActive {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.createAccessToken(input.Email)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return token, nil
}

func (s *Service) createAccessToken(email string) (string, error) {
	expiresAt := time.Now().Add(time.Duration(s.config.JWTExpireMinutes) * time.Minute)

	claims := jwt.MapClaims{
		"sub": email,
		"exp": expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecretKey))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// GetUserFromToken validates a JWT token and returns the user
func (s *Service) GetUserFromToken(ctx context.Context, token string) (*domain.User, error) {
	claims, err := s.validateToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	email, ok := claims["sub"].(string)
	if !ok || email == "" {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil || !user.IsActive {
		return nil, ErrInvalidToken
	}

	return user, nil
}

// ChangePasswordInput represents the input for a password change operation
type ChangePasswordInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.repo.GetUserByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.CurrentPassword)); err != nil {
		return ErrInvalidPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), s.config.BcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdateUserPassword(ctx, input.UserID, string(hashedPassword)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

// CreateApprovedUser creates a new approved user (admin only)
func (s *Service) CreateApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error) {
	email = strings.ToLower(email)

	// Check if already in approved list
	_, err := s.repo.GetApprovedUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}

	return s.repo.InsertApprovedUser(ctx, email, firstName)
}

// ListApprovedUsers lists all approved users (admin only)
func (s *Service) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	return s.repo.ListApprovedUsers(ctx)
}

func (s *Service) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
```

- [ ] **Step 3: Create internal/auth/repository.go**

```go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/your-org/go-backend-template/internal/db"
	"github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository handles database operations for authentication
type Repository struct {
	pool    *db.Pool
	queries *sqlc.Queries
}

// NewRepository creates a new auth repository
func NewRepository(database *db.Pool) *Repository {
	return &Repository{
		pool:    database,
		queries: sqlc.New(database),
	}
}

// IsEmailApproved checks if an email is in the approved users list
func (r *Repository) IsEmailApproved(ctx context.Context, email string) (bool, error) {
	_, err := r.queries.IsEmailApproved(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetApprovedUserByEmail gets an approved user by email
func (r *Repository) GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error) {
	row, err := r.queries.GetApprovedUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &domain.ApprovedUser{
		ID:        row.ID,
		Email:     row.Email,
		FirstName: row.FirstName,
		CreatedBy: uuidPtr(row.CreatedByUserID),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// GetUserByApprovedUserID gets a user by their approved_user_id
func (r *Repository) GetUserByApprovedUserID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.queries.GetUserByApprovedUserID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.User{ID: row.ID, ApprovedUserID: id}, nil
}

// InsertUser creates a new user
func (r *Repository) InsertUser(ctx context.Context, approvedUserID uuid.UUID, hashedPassword string) (*domain.User, error) {
	row, err := r.queries.InsertUser(ctx, sqlc.InsertUserParams{
		ApprovedUserID: approvedUserID,
		HashedPassword: hashedPassword,
		IsActive:       true,
	})
	if err != nil {
		return nil, err
	}
	return &domain.User{
		ID:             row.ID,
		ApprovedUserID: row.ApprovedUserID,
		HashedPassword: row.HashedPassword,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}, nil
}

// GetUserByEmail gets a user by email with their approved_user and roles
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserWithApprovedUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.userFromJoinRow(ctx, row)
}

// GetUserByID gets a user by ID with their approved_user and roles
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.queries.GetUserWithApprovedUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.userFromJoinRow(ctx, row)
}

func (r *Repository) userFromJoinRow(ctx context.Context, row sqlc.GetUserWithApprovedUserByIDRow) (*domain.User, error) {
	roles, err := r.fetchRoles(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:             row.ID,
		ApprovedUserID: row.ApprovedUserID,
		HashedPassword: row.HashedPassword,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		ApprovedUser: &domain.ApprovedUser{
			ID:        row.ApprovedUserID,
			Email:     row.Email,
			FirstName: row.FirstName,
			CreatedBy: uuidPtr(row.CreatedByUserID),
			CreatedAt: row.AuCreatedAt.Time,
			UpdatedAt: row.AuUpdatedAt.Time,
		},
		Roles: roles,
	}, nil
}

func (r *Repository) fetchRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.queries.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, domain.Role{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt.Time,
		})
	}
	return roles, nil
}

// UpdateUserPassword updates a user's password
func (r *Repository) UpdateUserPassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	return r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		HashedPassword: hashedPassword,
		ID:             id,
	})
}

// InsertApprovedUser creates a new approved user
func (r *Repository) InsertApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error) {
	row, err := r.queries.InsertApprovedUser(ctx, sqlc.InsertApprovedUserParams{
		Email:     email,
		FirstName: firstName,
	})
	if err != nil {
		return nil, err
	}
	return &domain.ApprovedUser{
		ID:        row.ID,
		Email:     row.Email,
		FirstName: row.FirstName,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// ListApprovedUsers lists all approved users
func (r *Repository) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	rows, err := r.queries.ListApprovedUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ApprovedUser, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domain.ApprovedUser{
			ID:        row.ID,
			Email:     row.Email,
			FirstName: row.FirstName,
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return result, nil
}

func uuidPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	ptr := uuid.UUID(u.Bytes)
	return &ptr
}
```

- [ ] **Step 4: Create internal/auth/handler.go**

```go
package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/your-org/go-backend-template/internal/http"
	"github.com/your-org/go-backend-template/internal/middleware"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for authentication and admin operations
type Handler struct {
	svc    ServiceInterface
	logger *zap.Logger
}

// NewHandler creates a new auth handler
func NewHandler(svc ServiceInterface, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// Signup handles POST /auth/signup
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		http.RespondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.svc.Signup(r.Context(), SignupInput(req))
	if err != nil {
		if errors.Is(err, ErrEmailNotApproved) {
			http.RespondError(w, http.StatusForbidden, "Email not approved. Please contact administrator.")
			return
		}
		if errors.Is(err, ErrUserAlreadyExists) {
			http.RespondError(w, http.StatusConflict, "User with this email already exists")
			return
		}

		h.logger.Error("signup failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusCreated, toUserResponse(user))
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		http.RespondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	token, err := h.svc.Login(r.Context(), LoginInput(req))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			http.RespondError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		h.logger.Error("login failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusOK, TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
	})
}

// LoginOAuth handles POST /auth/token (OAuth2 compatible)
func (h *Handler) LoginOAuth(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.RespondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email := r.FormValue("username")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.RespondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	token, err := h.svc.Login(r.Context(), LoginInput{
		Email:    email,
		Password: password,
	})

	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			http.RespondError(w, http.StatusUnauthorized, "Incorrect username or password")
			return
		}

		h.logger.Error("oauth login failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusOK, TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
	})
}

// GetMe handles GET /auth/me
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.RespondError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	http.RespondJSON(w, http.StatusOK, toUserResponse(user))
}

// ChangePassword handles POST /auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.RespondError(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.RespondError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	err := h.svc.ChangePassword(r.Context(), ChangePasswordInput{
		UserID:          user.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			http.RespondError(w, http.StatusBadRequest, "Current password is incorrect")
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			http.RespondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("change password failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Password changed successfully",
	})
}

// CreateApprovedUser handles POST /admin/approved-users
func (h *Handler) CreateApprovedUser(w http.ResponseWriter, r *http.Request) {
	var req ApprovedUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.FirstName == "" {
		http.RespondError(w, http.StatusBadRequest, "email and first_name are required")
		return
	}

	approvedUser, err := h.svc.CreateApprovedUser(r.Context(), req.Email, req.FirstName)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			http.RespondError(w, http.StatusConflict, "Email already in approved list")
			return
		}

		h.logger.Error("create approved user failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusCreated, toApprovedUserResponse(approvedUser))
}

// ListApprovedUsers handles GET /admin/approved-users
func (h *Handler) ListApprovedUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListApprovedUsers(r.Context())
	if err != nil {
		h.logger.Error("list approved users failed", zap.Error(err))
		http.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.RespondJSON(w, http.StatusOK, toApprovedUserResponses(users))
}
```

- [ ] **Step 5: Create internal/auth/handler_test.go**

```go
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSignupHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("valid signup", func(t *testing.T) {
		svc := &mockAuthService{}
		h := NewHandler(svc, logger)

		req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader([]byte(`{"email":"test@example.com","password":"password123"}`)))
		w := httptest.NewRecorder()

		h.Signup(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "email")
	})

	t.Run("missing email", func(t *testing.T) {
		svc := &mockAuthService{}
		h := NewHandler(svc, logger)

		req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader([]byte(`{"password":"password123"}`)))
		w := httptest.NewRecorder()

		h.Signup(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestLoginHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("valid login", func(t *testing.T) {
		svc := &mockAuthService{loginToken: "test-token"}
		h := NewHandler(svc, logger)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"test@example.com","password":"password123"}`)))
		w := httptest.NewRecorder()

		h.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "access_token")
	})

	t.Run("invalid credentials", func(t *testing.T) {
		svc := &mockAuthService{loginErr: ErrInvalidCredentials}
		h := NewHandler(svc, logger)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"test@example.com","password":"wrong"}`)))
		w := httptest.NewRecorder()

		h.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

type mockAuthService struct {
	loginToken string
	loginErr   error
}

func (m *mockAuthService) Signup(ctx context.Context, input SignupInput) (*domain.User, error) {
	return &domain.User{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
}

func (m *mockAuthService) Login(ctx context.Context, input LoginInput) (string, error) {
	return m.loginToken, m.loginErr
}

func (m *mockAuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	return nil
}

func (m *mockAuthService) GetUserFromToken(ctx context.Context, token string) (*domain.User, error) {
	return &domain.User{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
}

func (m *mockAuthService) CreateApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error) {
	return &domain.ApprovedUser{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Email: email, FirstName: firstName}, nil
}

func (m *mockAuthService) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	return []*domain.ApprovedUser{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Email: "test@example.com"}}, nil
}
```

- [ ] **Step 6: Create internal/auth/service_test.go**

```go
package auth

import (
	"context"
	"testing"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	ctx := context.Background()

	t.Run("email not approved", func(t *testing.T) {
		repo := &mockAuthRepository{approved: false}
		svc := NewService(repo, config.AuthConfig{BcryptCost: 4})

		_, err := svc.Signup(ctx, SignupInput{Email: "test@example.com", Password: "password123"})

		assert.ErrorIs(t, err, ErrEmailNotApproved)
	})
}

func TestLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid credentials", func(t *testing.T) {
		repo := &mockAuthRepository{user: nil}
		svc := NewService(repo, config.AuthConfig{BcryptCost: 4})

		_, err := svc.Login(ctx, LoginInput{Email: "test@example.com", Password: "wrong"})

		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

type mockAuthRepository struct {
	approved bool
	user     *domain.User
}

func (m *mockAuthRepository) IsEmailApproved(ctx context.Context, email string) (bool, error) {
	return m.approved, nil
}

func (m *mockAuthRepository) GetApprovedUserByEmail(ctx context.Context, email string) (*domain.ApprovedUser, error) {
	return &domain.ApprovedUser{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
}

func (m *mockAuthRepository) GetUserByApprovedUserID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, assert.AnError
}

func (m *mockAuthRepository) InsertUser(ctx context.Context, approvedUserID uuid.UUID, hashedPassword string) (*domain.User, error) {
	return &domain.User{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
}

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.user, nil
}

func (m *mockAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return m.user, nil
}

func (m *mockAuthRepository) UpdateUserPassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	return nil
}

func (m *mockAuthRepository) InsertApprovedUser(ctx context.Context, email, firstName string) (*domain.ApprovedUser, error) {
	return &domain.ApprovedUser{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
}

func (m *mockAuthRepository) ListApprovedUsers(ctx context.Context) ([]*domain.ApprovedUser, error) {
	return []*domain.ApprovedUser{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}}, nil
}
```

- [ ] **Step 7: Update router to wire auth handlers**

Modify internal/router/router.go:

```go
// Add to RouterConfig
type RouterConfig struct {
	AuthHandler  *auth.Handler
	AuthSvc      middleware.AuthProvider
	Logger       *zap.Logger
	AppConfig    *config.Config
	RateLimiter  *middleware.RateLimiter
}

// Add auth routes after health endpoints
r.Route("/auth", func(r chi.Router) {
	r.Use(RateLimitMiddleware(cfg.RateLimiter))

	r.Post("/signup", cfg.AuthHandler.Signup)
	r.Post("/login", cfg.AuthHandler.Login)
	r.Post("/token", cfg.AuthHandler.LoginOAuth)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.AuthSvc))
		r.Get("/me", cfg.AuthHandler.GetMe)
		r.Post("/change-password", cfg.AuthHandler.ChangePassword)
	})
})

r.Route("/admin", func(r chi.Router) {
	r.Use(middleware.RequireAdmin(cfg.AuthSvc))
	r.Post("/approved-users", cfg.AuthHandler.CreateApprovedUser)
	r.Get("/approved-users", cfg.AuthHandler.ListApprovedUsers)
})
```

- [ ] **Step 8: Verify compiles**

```bash
go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add internal/auth/
git commit -m "feat: add auth feature with signup/login/admin endpoints"
```

---

## Task 8: Todo Feature

(Similar structure - handler/service/repository/models + tests)

---

## Task 9: Integration Tests

---

## Task 10: Infrastructure Files

(.env.example, docker-compose.yml, Dockerfile, Makefile, .golangci.yml, README.md, etc.)

---

## Task 11: Wire Everything in main.go

---

**Plan complete. Two execution options:**

**1. Subagent-Driven (recommended)** - Fresh subagent per task, review between tasks

**2. Inline Execution** - Execute tasks in this session with checkpoints

**Which approach?**
