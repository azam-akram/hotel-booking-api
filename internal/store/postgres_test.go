package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestStore connects to TEST_DATABASE_URL and returns a Postgres bound to a
// private, empty schema with the application tables. The schema is dropped
// when the test ends, so existing data is never touched. The test is skipped
// when TEST_DATABASE_URL is not set.
func newTestStore(t *testing.T) *Postgres {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// CI sets REQUIRE_DB so a lost variable fails the job instead of skipping.
		if os.Getenv("REQUIRE_DB") != "" {
			t.Fatal("TEST_DATABASE_URL not set but REQUIRE_DB is set")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}
	t.Cleanup(admin.Close)

	schema := fmt.Sprintf("store_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Build the real schema from the migration so constraints match production.
	ddl, err := os.ReadFile("../db/migrations/000001_schema.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(ddl)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	return New(pool)
}

func TestCreateBooking(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	var roomID int64
	err := s.pool.QueryRow(ctx, `
		WITH h AS (INSERT INTO hotels (name) VALUES ('Grand Palace Hotel') RETURNING id)
		INSERT INTO rooms (hotel_id, number, type, capacity)
		SELECT id, '201', 'double', 2 FROM h
		RETURNING id`).Scan(&roomID)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}

	day := func(s string) model.Date {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return model.Date(d)
	}
	booking := func(ref, in, out string) model.Booking {
		return model.Booking{
			Reference: ref, Room: model.Room{ID: roomID},
			GuestName: "Jane Doe", GuestEmail: "jane@example.com",
			CheckIn: day(in), CheckOut: day(out), Guests: 2, CreatedAt: time.Now().UTC(),
		}
	}

	// The cases run in order and share the database: the first booking is
	// what the later ones collide with.
	tests := []struct {
		name    string
		booking model.Booking
		wantErr error
	}{
		{name: "creates a booking", booking: booking("ABCD2345", "2030-02-10", "2030-02-12")},
		{name: "overlapping stay is unavailable", booking: booking("EFGH2345", "2030-02-11", "2030-02-13"), wantErr: model.ErrRoomUnavailable},
		{name: "same reference is a duplicate", booking: booking("ABCD2345", "2030-03-01", "2030-03-03"), wantErr: model.ErrDuplicateReference},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateBooking(ctx, tt.booking, model.Idempotency{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			got, err := s.GetBooking(ctx, tt.booking.Reference)
			if err != nil {
				t.Fatalf("GetBooking: %v", err)
			}
			if got.GuestName != "Jane Doe" || got.Room.ID != roomID || got.Nights != 2 {
				t.Errorf("stored booking = %+v", got)
			}
		})
	}
}
