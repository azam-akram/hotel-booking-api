package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

// fakeStore is a Store with just enough behaviour for the CreateBooking test.
type fakeStore struct {
	room      model.Room
	createErr error // returned by CreateBooking
	created   []model.Booking
}

func (f *fakeStore) Ping(context.Context) error { return nil }

func (f *fakeStore) GetHotel(context.Context, int64) (model.Hotel, error) {
	return model.Hotel{}, nil
}

func (f *fakeStore) GetRoom(context.Context, int64, int64) (model.Room, error) {
	return f.room, nil
}

func (f *fakeStore) AvailableRooms(context.Context, int64, model.Date, model.Date, int) ([]model.Room, error) {
	return nil, nil
}

func (f *fakeStore) FindHotels(context.Context, string) ([]model.Hotel, error) { return nil, nil }

func (f *fakeStore) CreateBooking(_ context.Context, b model.Booking, _ model.Idempotency) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, b)
	return nil
}

func (f *fakeStore) BookingByKey(context.Context, string) (model.Booking, model.Idempotency, error) {
	return model.Booking{}, model.Idempotency{}, fmt.Errorf("%w: idempotency key", model.ErrNotFound)
}

func (f *fakeStore) GetBooking(context.Context, string) (model.Booking, error) {
	return model.Booking{}, nil
}

func (f *fakeStore) Seed(ctx context.Context) error {
	return nil
}

func (f *fakeStore) Reset(ctx context.Context) error {
	return nil
}

func TestCreateBooking(t *testing.T) {
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	checkIn, _ := time.Parse("2006-01-02", "2030-02-10")
	checkOut, _ := time.Parse("2006-01-02", "2030-02-12")

	validReq := model.BookingRequest{
		HotelID: 1, RoomID: 3, GuestName: " Jane Doe ", GuestEmail: "jane@example.com",
		CheckIn: model.Date(checkIn), CheckOut: model.Date(checkOut), Guests: 2,
	}
	tooManyGuests := validReq
	tooManyGuests.Guests = 3

	tests := []struct {
		name      string
		req       model.BookingRequest
		now       time.Time
		createErr error
		wantErr   error
	}{
		{name: "creates a booking", req: validReq, now: now},
		{name: "too many guests", req: tooManyGuests, now: now, wantErr: model.ErrOverCapacity},
		{name: "check-in in the past", req: validReq, now: checkIn.AddDate(0, 0, 1), wantErr: model.ErrInvalidInput},
		{name: "room already booked", req: validReq, now: now, createErr: model.ErrRoomUnavailable, wantErr: model.ErrRoomUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{
				room:      model.Room{ID: 3, HotelID: 1, Number: "201", Capacity: 2},
				createErr: tt.createErr,
			}
			svc := New(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
			svc.now = func() time.Time { return tt.now }

			res, err := svc.CreateBooking(context.Background(), tt.req, "")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(store.created) != 1 {
				t.Fatalf("store saved %d bookings, want 1", len(store.created))
			}
			if res.Booking.Nights != 2 {
				t.Errorf("nights = %d, want 2", res.Booking.Nights)
			}
			if res.Booking.GuestName != "Jane Doe" {
				t.Errorf("guest name = %q, want %q", res.Booking.GuestName, "Jane Doe")
			}
			if store.created[0].Reference != res.Booking.Reference {
				t.Errorf("saved reference %q, returned %q", store.created[0].Reference, res.Booking.Reference)
			}
		})
	}
}
