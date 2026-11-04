package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

// fakeService is a BookingService whose CreateBooking behaviour is set per test.
type fakeService struct {
	createBooking func(ctx context.Context, req model.BookingRequest, key string) (model.BookingResponse, error)
}

func (f fakeService) Ping(context.Context) error { return nil }

func (f fakeService) AvailableRooms(context.Context, int64, model.AvailabilityQuery) (model.AvailableRoomsResponse, error) {
	return model.AvailableRoomsResponse{}, nil
}

func (f fakeService) FindHotels(context.Context, string) ([]model.Hotel, error) { return nil, nil }

func (f fakeService) GetBooking(context.Context, string) (model.Booking, error) {
	return model.Booking{}, nil
}

func (f fakeService) CreateBooking(ctx context.Context, req model.BookingRequest, key string) (model.BookingResponse, error) {
	return f.createBooking(ctx, req, key)
}

func TestCreateBooking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const body = `{
		"hotel_id": 1, "room_id": 3, "guest_name": "Jane Doe", "guest_email": "jane@example.com",
		"check_in": "2030-02-10", "check_out": "2030-02-12", "guests": 2
	}`

	tests := []struct {
		name         string
		svcErr       error
		wantStatus   int
		wantLocation string
		wantCode     string // error code in the body, if any
	}{
		{
			name:         "creates a booking",
			wantStatus:   http.StatusCreated,
			wantLocation: "/v1/bookings/ABCD2345",
		},
		{
			name:       "room already booked",
			svcErr:     model.ErrRoomUnavailable,
			wantStatus: http.StatusConflict,
			wantCode:   model.CodeRoomUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeService{createBooking: func(context.Context, model.BookingRequest, string) (model.BookingResponse, error) {
				if tt.svcErr != nil {
					return model.BookingResponse{}, tt.svcErr
				}
				return model.BookingResponse{Booking: model.Booking{Reference: "ABCD2345"}}, nil
			}}
			h := New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

			r := gin.New()
			r.POST("/v1/bookings", h.CreateBooking)

			req := httptest.NewRequest(http.MethodPost, "/v1/bookings", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}

			if tt.wantCode != "" {
				var e model.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
					t.Fatalf("decode error body: %v; body = %s", err, rec.Body)
				}
				if e.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", e.Error.Code, tt.wantCode)
				}
				return
			}

			var got model.Booking
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode booking: %v; body = %s", err, rec.Body)
			}
			if got.Reference != "ABCD2345" {
				t.Errorf("reference = %q, want ABCD2345", got.Reference)
			}
		})
	}
}
