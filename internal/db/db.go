// Package db provides database connection and utilities.
package db

import (
	"context"

	"github.com/your-org/go-backend-template/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool for database operations.
type Pool struct {
	*pgxpool.Pool
}

// New creates a new database connection pool.
func New(ctx context.Context, cfg config.DatabaseConfig) (*Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.URL)
	if err != nil {
		return nil, err
	}
	return &Pool{Pool: pool}, nil
}
