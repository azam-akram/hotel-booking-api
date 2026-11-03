package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

// fakeService is a BookingService whose FindHotels behaviour is set per test.
type fakeService struct {
	findHotels func(ctx context.Context, name string) ([]model.Hotel, error)
}

func (f fakeService) Ping(context.Context) error { return nil }

func (f fakeService) FindHotels(ctx context.Context, name string) ([]model.Hotel, error) {
	return f.findHotels(ctx, name)
}

func TestFindHotels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	grandPalace := model.Hotel{ID: 1, Name: "Grand Palace Hotel"}

	tests := []struct {
		name       string
		query      string
		svcHotels  []model.Hotel
		svcErr     error
		wantStatus int
		wantName   string // name the service should receive
		wantHotels []model.Hotel
		wantCode   string // error code expected in the body, if any
	}{
		{
			name:       "returns matching hotels",
			query:      "Grand Palace",
			svcHotels:  []model.Hotel{grandPalace},
			wantStatus: http.StatusOK,
			wantName:   "Grand Palace",
			wantHotels: []model.Hotel{grandPalace},
		},
		{
			name:       "passes query value through unchanged",
			query:      "  grand ",
			svcHotels:  []model.Hotel{grandPalace},
			wantStatus: http.StatusOK,
			wantName:   "  grand ",
			wantHotels: []model.Hotel{grandPalace},
		},
		{
			name:       "missing name is passed as empty and service rejects it",
			query:      "",
			svcErr:     model.Invalidf("name must not be blank"),
			wantStatus: http.StatusBadRequest,
			wantName:   "",
			wantCode:   model.CodeInvalidInput,
		},
		{
			name:       "no match maps to 404",
			query:      "nowhere",
			svcErr:     fmt.Errorf("%w: no hotel matches %q", model.ErrNotFound, "nowhere"),
			wantStatus: http.StatusNotFound,
			wantName:   "nowhere",
			wantCode:   model.CodeNotFound,
		},
		{
			name:       "unexpected error maps to 500 without leaking details",
			query:      "grand",
			svcErr:     errors.New("connection refused to 10.0.0.5"),
			wantStatus: http.StatusInternalServerError,
			wantName:   "grand",
			wantCode:   model.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotName string
			var calls int
			svc := fakeService{findHotels: func(_ context.Context, name string) ([]model.Hotel, error) {
				calls++
				gotName = name
				return tt.svcHotels, tt.svcErr
			}}
			h := New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

			r := gin.New()
			r.GET("/v1/hotels", h.FindHotels)

			target := "/v1/hotels"
			if tt.query != "" {
				target += "?name=" + url.QueryEscape(tt.query)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if calls != 1 {
				t.Fatalf("service called %d times, want 1", calls)
			}
			if gotName != tt.wantName {
				t.Errorf("service got name %q, want %q", gotName, tt.wantName)
			}

			if tt.wantCode != "" {
				var body model.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode error body: %v; body = %s", err, rec.Body)
				}
				if body.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
				if tt.wantStatus == http.StatusInternalServerError && body.Error.Message != "internal server error" {
					t.Errorf("500 message = %q, leaks internal detail", body.Error.Message)
				}
				return
			}

			var body model.HotelListResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v; body = %s", err, rec.Body)
			}
			if !reflect.DeepEqual(body.Hotels, tt.wantHotels) {
				t.Errorf("hotels = %+v, want %+v", body.Hotels, tt.wantHotels)
			}
		})
	}
}
