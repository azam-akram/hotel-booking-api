package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const dbStartupTimeout = 30 * time.Second

// openDatabase creates a connection pool and verifies the database is reachable.
func openDatabase(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbStartupTimeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info("Database connected")

	return pool, nil
}
