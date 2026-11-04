package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// All SQL queries are written here
const (
	queryFindHotels = `
		SELECT id, name
		FROM hotels
		WHERE strpos(lower(name), lower($1)) > 0
		ORDER BY lower(name) = lower($1) DESC, name`

	queryGetHotel = `
		SELECT id, name
		FROM hotels
		WHERE id = $1`

	queryGetRoom = `
		SELECT id, hotel_id, number, type, capacity
		FROM rooms
		WHERE id = $1 AND hotel_id = $2`

	// Rooms of a hotel that fit the guests and have no booking overlapping the stay.
	// Both ranges are half-open, so check-out day is still free.
	queryAvailableRooms = `
		SELECT r.id, r.hotel_id, r.number, r.type, r.capacity
		FROM rooms r
		WHERE r.hotel_id = $1
		  AND r.capacity >= $4
		  AND NOT EXISTS (
		      SELECT 1
		      FROM bookings b
		      WHERE b.room_id = r.id
		        AND daterange(b.check_in, b.check_out, '[)') && daterange($2::date, $3::date, '[)')
		  )
		ORDER BY r.id`

	queryInsertBooking = `
		INSERT INTO bookings
		    (reference, room_id, guest_name, guest_email, check_in, check_out, guests, created_at,
		     idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	queryGetBooking = `
		SELECT b.reference, r.id, r.hotel_id, r.number, r.type, r.capacity,
		       b.guest_name, b.guest_email, b.check_in, b.check_out, b.guests, b.created_at,
		       COALESCE(b.request_hash, '')
		FROM bookings b
		JOIN rooms r ON r.id = b.room_id
		WHERE b.reference = $1`

	queryGetBookingByKey = `
		SELECT b.reference, r.id, r.hotel_id, r.number, r.type, r.capacity,
		       b.guest_name, b.guest_email, b.check_in, b.check_out, b.guests, b.created_at,
		       COALESCE(b.request_hash, '')
		FROM bookings b
		JOIN rooms r ON r.id = b.room_id
		WHERE b.idempotency_key = $1`
)

// PostgreSQL error codes and constraint names (see db/migrations).
const (
	codeUniqueViolation    = "23505"
	codeExclusionViolation = "23P01"

	constraintReference = "bookings_reference_key"
	constraintIdemKey   = "bookings_idempotency_key_key"
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
	rows, err := p.pool.Query(ctx, queryFindHotels, name)
	if err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	hotels, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Hotel])
	if err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	return hotels, nil
}

func (p *Postgres) GetHotel(ctx context.Context, id int64) (model.Hotel, error) {
	rows, err := p.pool.Query(ctx, queryGetHotel, id)
	if err != nil {
		return model.Hotel{}, fmt.Errorf("get hotel: %w", err)
	}
	h, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[model.Hotel])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Hotel{}, fmt.Errorf("%w: hotel %d", model.ErrNotFound, id)
	}
	if err != nil {
		return model.Hotel{}, fmt.Errorf("get hotel: %w", err)
	}
	return h, nil
}

func (p *Postgres) GetRoom(ctx context.Context, hotelID, roomID int64) (model.Room, error) {
	rows, err := p.pool.Query(ctx, queryGetRoom, roomID, hotelID)
	if err != nil {
		return model.Room{}, fmt.Errorf("get room: %w", err)
	}
	r, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[model.Room])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Room{}, fmt.Errorf("%w: room %d in hotel %d", model.ErrNotFound, roomID, hotelID)
	}
	if err != nil {
		return model.Room{}, fmt.Errorf("get room: %w", err)
	}
	return r, nil
}

func (p *Postgres) AvailableRooms(ctx context.Context, hotelID int64, checkIn, checkOut model.Date, guests int) ([]model.Room, error) {
	rows, err := p.pool.Query(ctx, queryAvailableRooms, hotelID, checkIn.Time(), checkOut.Time(), guests)
	if err != nil {
		return nil, fmt.Errorf("available rooms: %w", err)
	}
	rooms, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Room])
	if err != nil {
		return nil, fmt.Errorf("available rooms: %w", err)
	}
	return rooms, nil
}

// CreateBooking inserts the booking and maps constraint violations to the
// model errors the service understands.
func (p *Postgres) CreateBooking(ctx context.Context, b model.Booking, idem model.Idempotency) error {
	_, err := p.pool.Exec(ctx, queryInsertBooking,
		b.Reference, b.Room.ID, b.GuestName, b.GuestEmail, b.CheckIn.Time(), b.CheckOut.Time(),
		b.Guests, b.CreatedAt, nullIfEmpty(idem.Key), nullIfEmpty(idem.RequestHash))
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == codeExclusionViolation:
			return model.ErrRoomUnavailable
		case pgErr.Code == codeUniqueViolation && pgErr.ConstraintName == constraintReference:
			return model.ErrDuplicateReference
		case pgErr.Code == codeUniqueViolation && pgErr.ConstraintName == constraintIdemKey:
			return model.ErrDuplicateKey
		}
	}
	return fmt.Errorf("insert booking: %w", err)
}

// BookingByKey finds the booking made with an idempotency key, together with
// the key and request hash saved with it.
func (p *Postgres) BookingByKey(ctx context.Context, key string) (model.Booking, model.Idempotency, error) {
	b, hash, err := scanBooking(p.pool.QueryRow(ctx, queryGetBookingByKey, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, model.Idempotency{}, fmt.Errorf("%w: idempotency key %q", model.ErrNotFound, key)
	}
	if err != nil {
		return model.Booking{}, model.Idempotency{}, fmt.Errorf("get booking by key: %w", err)
	}
	return b, model.Idempotency{Key: key, RequestHash: hash}, nil
}

func (p *Postgres) GetBooking(ctx context.Context, reference string) (model.Booking, error) {
	b, _, err := scanBooking(p.pool.QueryRow(ctx, queryGetBooking, reference))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, fmt.Errorf("%w: booking %q", model.ErrNotFound, reference)
	}
	if err != nil {
		return model.Booking{}, fmt.Errorf("get booking: %w", err)
	}
	return b, nil
}

func scanBooking(row pgx.Row) (model.Booking, string, error) {
	var (
		b                 model.Booking
		checkIn, checkOut time.Time
		hash              string
	)
	err := row.Scan(&b.Reference,
		&b.Room.ID, &b.Room.HotelID, &b.Room.Number, &b.Room.Type, &b.Room.Capacity,
		&b.GuestName, &b.GuestEmail, &checkIn, &checkOut, &b.Guests, &b.CreatedAt, &hash)
	if err != nil {
		return model.Booking{}, "", err
	}
	b.HotelID = b.Room.HotelID
	b.CheckIn, b.CheckOut = model.Date(checkIn), model.Date(checkOut)
	b.Nights = b.CheckIn.Nights(b.CheckOut)
	return b, hash, nil
}

// nullIfEmpty stores an empty string as SQL NULL.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
