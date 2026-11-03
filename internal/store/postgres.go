package store

import (
	"context"

	"github.com/azam-akram/hotel-booking-api/internal/model"
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
	return []model.Hotel{}, nil
}
