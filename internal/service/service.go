package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

const maxNights = 30

type Store interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
	GetHotel(ctx context.Context, id int64) (model.Hotel, error)
	GetRoom(ctx context.Context, hotelID, roomID int64) (model.Room, error)
	AvailableRooms(ctx context.Context, hotelID int64, checkIn, checkOut model.Date, guests int) ([]model.Room, error)
}

type Service struct {
	store  Store
	logger *slog.Logger
	now    func() time.Time
}

func New(s Store, l *slog.Logger) *Service {
	return &Service{
		store:  s,
		logger: l,
		now:    time.Now,
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
	s.logger.DebugContext(ctx, "hotel search", "name", name, "matches", len(hotels))
	if len(hotels) == 0 {
		return nil, fmt.Errorf("%w: no hotel matches %q", model.ErrNotFound, name)
	}
	return hotels, nil
}

func (s *Service) AvailableRooms(ctx context.Context, hotelID int64, q model.AvailabilityQuery) (model.AvailableRoomsResponse, error) {
	if err := s.validateStay(q.CheckIn, q.CheckOut); err != nil {
		return model.AvailableRoomsResponse{}, err
	}
	if _, err := s.store.GetHotel(ctx, hotelID); err != nil {
		return model.AvailableRoomsResponse{}, err
	}
	rooms, err := s.store.AvailableRooms(ctx, hotelID, q.CheckIn, q.CheckOut, q.Guests)
	if err != nil {
		return model.AvailableRoomsResponse{}, err
	}
	s.logger.DebugContext(ctx, "availability search", "hotel_id", hotelID,
		"check_in", q.CheckIn.String(), "check_out", q.CheckOut.String(), "guests", q.Guests, "rooms", len(rooms))
	if rooms == nil {
		rooms = []model.Room{} // encode as [] rather than null
	}
	return model.AvailableRoomsResponse{
		HotelID:  hotelID,
		CheckIn:  q.CheckIn,
		CheckOut: q.CheckOut,
		Nights:   q.CheckIn.Nights(q.CheckOut),
		Guests:   q.Guests,
		Rooms:    rooms,
	}, nil
}

func (s *Service) validateStay(checkIn, checkOut model.Date) error {
	if checkIn.IsZero() || checkOut.IsZero() {
		return model.Invalidf("check_in and check_out are required")
	}
	if !checkOut.Time().After(checkIn.Time()) {
		return model.Invalidf("check_out must be after check_in")
	}
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if checkIn.Time().Before(today) {
		return model.Invalidf("check_in must not be in the past")
	}
	if n := checkIn.Nights(checkOut); n > maxNights {
		return model.Invalidf("stay of %d nights exceeds the maximum of %d", n, maxNights)
	}
	return nil
}
