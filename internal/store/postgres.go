package store

import (
	"context"
	"fmt"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{
		pool: pool,
	}
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) FindHotels(ctx context.Context, name string) ([]model.Hotel, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, name FROM hotels
		WHERE strpos(lower(name), lower($1)) > 0
		ORDER BY lower(name) = lower($1) DESC, name`, name)
	if err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	hotels, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Hotel])
	if err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	return hotels, nil
}
