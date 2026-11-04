package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

// maxBodyBytes caps the size of a request body the API will read.
const maxBodyBytes = 1 << 20 // 1 MiB

type BookingService interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
	AvailableRooms(ctx context.Context, hotelID int64, q model.AvailabilityQuery) (model.AvailableRoomsResponse, error)
	CreateBooking(ctx context.Context, req model.BookingRequest, idempotencyKey string) (model.BookingResponse, error)
	GetBooking(ctx context.Context, reference string) (model.Booking, error)
}

type BookingHandler struct {
	svc    BookingService
	logger *slog.Logger
}

func New(svc BookingService, log *slog.Logger) *BookingHandler {
	return &BookingHandler{
		svc:    svc,
		logger: log,
	}
}

// Health handles GET /healthcheck
func (h *BookingHandler) Health(c *gin.Context) {
	if err := h.svc.Ping(c.Request.Context()); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// FindHotels handles GET /v1/hotels?name=
func (h *BookingHandler) FindHotels(c *gin.Context) {
	hotels, err := h.svc.FindHotels(c.Request.Context(), c.Query("name"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.HotelListResponse{Hotels: hotels})
}

// AvailableRooms handles GET /v1/hotels/:hotelId/rooms/available
func (h *BookingHandler) AvailableRooms(c *gin.Context) {
	hotelID, err := strconv.ParseInt(c.Param("hotelId"), 10, 64)
	if err != nil || hotelID <= 0 {
		h.respondError(c, model.Invalidf("hotelId must be a positive integer"))
		return
	}

	var q model.AvailabilityQuery
	if err = c.ShouldBindQuery(&q); err != nil {
		h.logger.DebugContext(c.Request.Context(), "invalid availability query", "error", err, "query", c.Request.URL.RawQuery)
		h.respondError(c, model.Invalidf("check_in and check_out must be dates as YYYY-MM-DD, and guests must be between 1 and 10"))
		return
	}

	res, err := h.svc.AvailableRooms(c.Request.Context(), hotelID, q)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreateBooking handles POST /v1/bookings
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	var req model.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.respondError(c, model.ErrPayloadTooLarge)
			return
		}
		h.logger.DebugContext(c.Request.Context(), "invalid booking request", "error", err)
		h.respondError(c, model.Invalidf("request body must be valid JSON with hotel_id, room_id, guest_name, guest_email, check_in, check_out and guests"))
		return
	}

	res, err := h.svc.CreateBooking(c.Request.Context(), req, c.GetHeader("Idempotency-Key"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	status := http.StatusCreated
	if res.Replayed {
		status = http.StatusOK
	} else {
		c.Header("Location", "/v1/bookings/"+res.Booking.Reference)
	}
	c.JSON(status, res.Booking)
}

// GetBooking handles GET /v1/bookings/:reference
func (h *BookingHandler) GetBooking(c *gin.Context) {
	b, err := h.svc.GetBooking(c.Request.Context(), c.Param("reference"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}
