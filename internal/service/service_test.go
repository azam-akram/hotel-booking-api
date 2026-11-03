package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/azam-akram/hotel-booking-api/internal/model"
)

// fakeStore is a Store whose FindHotels behaviour is set per test.
type fakeStore struct {
	findHotels func(ctx context.Context, name string) ([]model.Hotel, error)
}

func (f fakeStore) Ping(context.Context) error { return nil }

// The methods below are not exercised by the FindHotels tests; they only make
// fakeStore satisfy Store.

func (f fakeStore) GetHotel(context.Context, int64) (model.Hotel, error) {
	return model.Hotel{}, nil
}

func (f fakeStore) GetRoom(context.Context, int64, int64) (model.Room, error) {
	return model.Room{}, nil
}

func (f fakeStore) AvailableRooms(context.Context, int64, model.Date, model.Date, int) ([]model.Room, error) {
	return nil, nil
}

func (f fakeStore) FindHotels(ctx context.Context, name string) ([]model.Hotel, error) {
	return f.findHotels(ctx, name)
}

func TestFindHotels(t *testing.T) {
	grandPalace := model.Hotel{ID: 1, Name: "Grand Palace Hotel"}
	errStore := errors.New("connection refused")

	tests := []struct {
		name        string
		input       string
		storeHotels []model.Hotel
		storeErr    error
		wantCalls   int
		wantName    string // name the store should receive
		wantHotels  []model.Hotel
		wantErrIs   error // sentinel the returned error must wrap, if any
	}{
		{
			name:        "returns hotels from the store",
			input:       "Grand Palace",
			storeHotels: []model.Hotel{grandPalace},
			wantCalls:   1,
			wantName:    "Grand Palace",
			wantHotels:  []model.Hotel{grandPalace},
		},
		{
			name:        "trims surrounding whitespace before searching",
			input:       "  Grand Palace \t",
			storeHotels: []model.Hotel{grandPalace},
			wantCalls:   1,
			wantName:    "Grand Palace",
			wantHotels:  []model.Hotel{grandPalace},
		},
		{
			name:      "empty name is invalid and never reaches the store",
			input:     "",
			wantCalls: 0,
			wantErrIs: model.ErrInvalidInput,
		},
		{
			name:      "blank name is invalid and never reaches the store",
			input:     "   \t ",
			wantCalls: 0,
			wantErrIs: model.ErrInvalidInput,
		},
		{
			name:        "no matching hotel is not found",
			input:       "nowhere",
			storeHotels: []model.Hotel{},
			wantCalls:   1,
			wantName:    "nowhere",
			wantErrIs:   model.ErrNotFound,
		},
		{
			name:      "nil result from the store is not found",
			input:     "nowhere",
			wantCalls: 1,
			wantName:  "nowhere",
			wantErrIs: model.ErrNotFound,
		},
		{
			name:      "store error is returned unchanged",
			input:     "grand",
			storeErr:  errStore,
			wantCalls: 1,
			wantName:  "grand",
			wantErrIs: errStore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			var gotName string
			store := fakeStore{findHotels: func(_ context.Context, name string) ([]model.Hotel, error) {
				calls++
				gotName = name
				return tt.storeHotels, tt.storeErr
			}}
			svc := New(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

			got, err := svc.FindHotels(context.Background(), tt.input)

			if calls != tt.wantCalls {
				t.Fatalf("store called %d times, want %d", calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && gotName != tt.wantName {
				t.Errorf("store got name %q, want %q", gotName, tt.wantName)
			}

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error = %v, want one wrapping %v", err, tt.wantErrIs)
				}
				if got != nil {
					t.Errorf("hotels = %+v on error, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantHotels) {
				t.Errorf("hotels = %+v, want %+v", got, tt.wantHotels)
			}
		})
	}
}
