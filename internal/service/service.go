package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

type Store interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
}

type Service struct {
	store  Store
	logger *slog.Logger
}

func New(s Store, l *slog.Logger) *Service {
	return &Service{
		store:  s,
		logger: l,
	}
}

// Ping checks that the database is reachable.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func (s *Service) FindHotels(ctx context.Context, name string) ([]model.Hotel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.Invalidf("name must not be blank")
	}
	hotels, err := s.store.FindHotels(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(hotels) == 0 {
		return nil, fmt.Errorf("%w: no hotel matches %q", model.ErrNotFound, name)
	}
	return hotels, nil
}
