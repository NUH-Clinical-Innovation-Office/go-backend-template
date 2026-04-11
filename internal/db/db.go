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
