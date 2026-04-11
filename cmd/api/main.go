// Command api is the main entry point for the API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/go-backend-template/internal/auth"
	"github.com/your-org/go-backend-template/internal/config"
	"github.com/your-org/go-backend-template/internal/db"
	dbSQLC "github.com/your-org/go-backend-template/internal/db/sqlc"
	"github.com/your-org/go-backend-template/internal/logging"
	"github.com/your-org/go-backend-template/internal/observability"
	"github.com/your-org/go-backend-template/internal/router"
	"github.com/your-org/go-backend-template/internal/todo"
	"go.opentelemetry.io/otel/trace"
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

	// Initialize OpenTelemetry tracing
	tracerProvider, err := observability.Setup(
		context.Background(),
		cfg.Observability.ServiceName,
		logger,
	)
	if err != nil {
		logger.Warn("failed to initialize tracing", zap.Error(err))
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(ctx); err != nil {
				logger.Warn("tracer provider shutdown failed", zap.Error(err))
			}
		}()
		logger.Info("OpenTelemetry tracing initialized")
	}

	// Connect to database
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	logger.Info("database connection established")

	// Initialize SQLC queries
	queries := dbSQLC.New(pool.Pool)

	// Initialize auth repository and service
	authRepo := auth.NewRepository(queries)
	authService := auth.NewService(authRepo, cfg.Auth.JWTSecretKey, time.Duration(cfg.Auth.JWTExpireMinutes)*time.Minute)

	// Initialize todo repository and service
	todoRepo := todo.NewRepository(queries)
	todoService := todo.NewService(todoRepo)

	// Get tracer
	tracer := trace.NewNoopTracerProvider().Tracer(cfg.Observability.ServiceName)
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer(cfg.Observability.ServiceName)
	}

	// Build router with all dependencies
	routerConfig := router.RouterConfig{
		Logger:      logger,
		Tracer:      tracer,
		AuthSvc:     authService,
		TodoService: todoService,
	}
	mux := router.New(routerConfig)

	// Start HTTP server
	return startHTTPServer(cfg, mux, logger)
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
