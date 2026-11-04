package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

const maxNights = 30
const (
	referenceAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I
	referenceLength   = 8
	referenceAttempts = 3
)

type Store interface {
	Ping(ctx context.Context) error
	Seed(ctx context.Context) error
	Reset(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
	GetHotel(ctx context.Context, id int64) (model.Hotel, error)
	GetRoom(ctx context.Context, hotelID, roomID int64) (model.Room, error)
	AvailableRooms(ctx context.Context, hotelID int64, checkIn, checkOut model.Date, guests int) ([]model.Room, error)
	CreateBooking(ctx context.Context, b model.Booking, idem model.Idempotency) error
	BookingByKey(ctx context.Context, key string) (model.Booking, model.Idempotency, error)
	GetBooking(ctx context.Context, reference string) (model.Booking, error)
}

type Service struct {
	store  Store
	logger *slog.Logger
	now    func() time.Time
}

func New(store Store, log *slog.Logger) *Service {
	return &Service{
		store:  store,
		logger: log,
		now:    time.Now,
	}
}

// Ping checks that the database is reachable.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

// FindHotels returns the hotels whose name contains the given text.
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

// AvailableRooms returns the rooms of a hotel that are free for the whole stay
// and have capacity for the guests.
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

func (s *Service) CreateBooking(ctx context.Context, req model.BookingRequest, idempotencyKey string) (model.BookingResponse, error) {
	var idem model.Idempotency
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		if err := validateKey(key); err != nil {
			return model.BookingResponse{}, err
		}
		hash, err := hashRequest(req)
		if err != nil {
			return model.BookingResponse{}, err
		}
		idem = model.Idempotency{Key: key, RequestHash: hash}

		res, err := s.replay(ctx, idem)
		if !errors.Is(err, model.ErrNotFound) {
			return res, err
		}
	}

	if err := s.validateBooking(&req); err != nil {
		return model.BookingResponse{}, err
	}
	room, err := s.store.GetRoom(ctx, req.HotelID, req.RoomID)
	if err != nil {
		return model.BookingResponse{}, err
	}
	if req.Guests > room.Capacity {
		return model.BookingResponse{}, fmt.Errorf("%w: room %s holds %d, requested %d",
			model.ErrOverCapacity, room.Number, room.Capacity, req.Guests)
	}

	b := model.Booking{
		HotelID:    req.HotelID,
		Room:       room,
		GuestName:  req.GuestName,
		GuestEmail: req.GuestEmail,
		CheckIn:    req.CheckIn,
		CheckOut:   req.CheckOut,
		Nights:     req.CheckIn.Nights(req.CheckOut),
		Guests:     req.Guests,
		CreatedAt:  s.now().UTC(),
	}

	for range referenceAttempts {
		ref, err := newReference()
		if err != nil {
			return model.BookingResponse{}, err
		}
		b.Reference = ref
		err = s.store.CreateBooking(ctx, b, idem)
		if errors.Is(err, model.ErrDuplicateReference) {
			continue
		}
		if errors.Is(err, model.ErrDuplicateKey) {
			// A concurrent request with the same key won the race.
			return s.replay(ctx, idem)
		}
		if err != nil {
			return model.BookingResponse{}, err
		}

		return model.BookingResponse{Booking: b}, nil
	}
	return model.BookingResponse{}, errors.New("could not generate a unique booking reference")
}

// Seed inserts the test data.
func (s *Service) Seed(ctx context.Context) error {
	return s.store.Seed(ctx)
}

// Reset removes all data.
func (s *Service) Reset(ctx context.Context) error {
	return s.store.Reset(ctx)
}

// GetBooking finds a booking by reference. References are stored in upper
// case, so the lookup ignores case and surrounding spaces.
func (s *Service) GetBooking(ctx context.Context, reference string) (model.Booking, error) {
	return s.store.GetBooking(ctx, strings.ToUpper(strings.TrimSpace(reference)))
}

// replay returns the booking created earlier with idem.Key. It returns
// model.ErrNotFound if the key is unused, and model.ErrIdempotencyMismatch if
// it was used for a different request.
func (s *Service) replay(ctx context.Context, idem model.Idempotency) (model.BookingResponse, error) {
	b, stored, err := s.store.BookingByKey(ctx, idem.Key)
	if err != nil {
		return model.BookingResponse{}, err
	}
	if stored.RequestHash != idem.RequestHash {
		return model.BookingResponse{}, model.ErrIdempotencyMismatch
	}
	return model.BookingResponse{Booking: b, Replayed: true}, nil
}

// newReference returns a random, non-sequential booking reference.
func newReference() (string, error) {
	buf := make([]byte, referenceLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate reference: %w", err)
	}
	for i, v := range buf {
		buf[i] = referenceAlphabet[int(v)%len(referenceAlphabet)] // 256 % 32 == 0, so no modulo bias
	}
	return string(buf), nil
}
