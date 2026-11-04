package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const roomColumns = "id, hotel_id, number, type, capacity"

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

func (p *Postgres) GetHotel(ctx context.Context, id int64) (model.Hotel, error) {
	rows, _ := p.pool.Query(ctx, "SELECT id, name FROM hotels WHERE id = $1", id)
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
	rows, _ := p.pool.Query(ctx,
		"SELECT "+roomColumns+" FROM rooms WHERE id = $1 AND hotel_id = $2", roomID, hotelID)
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
	rows, err := p.pool.Query(ctx, `
		SELECT `+roomColumns+` FROM rooms r
		WHERE r.hotel_id = $1
		  AND r.capacity >= $4
		  AND NOT EXISTS (
		      SELECT 1 FROM bookings b
		      WHERE b.room_id = r.id
		        AND daterange(b.check_in, b.check_out, '[)') && daterange($2::date, $3::date, '[)'))
		ORDER BY r.id`, hotelID, checkIn.Time(), checkOut.Time(), guests)
	if err != nil {
		return nil, fmt.Errorf("available rooms: %w", err)
	}
	rooms, err := pgx.CollectRows(rows, pgx.RowToStructByPos[model.Room])
	if err != nil {
		return nil, fmt.Errorf("available rooms: %w", err)
	}
	return rooms, nil
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

const bookingSelect = `
	SELECT b.reference, r.id, r.hotel_id, r.number, r.type, r.capacity,
	       b.guest_name, b.guest_email, b.check_in, b.check_out, b.guests, b.created_at,
	       COALESCE(b.request_hash, '')
	FROM bookings b JOIN rooms r ON r.id = b.room_id`

func (p *Postgres) GetBooking(ctx context.Context, reference string) (model.Booking, error) {
	b, _, err := scanBooking(p.pool.QueryRow(ctx, bookingSelect+" WHERE b.reference = $1", reference))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, fmt.Errorf("%w: booking %q", model.ErrNotFound, reference)
	}
	if err != nil {
		return model.Booking{}, fmt.Errorf("get booking: %w", err)
	}
	return b, nil
}
