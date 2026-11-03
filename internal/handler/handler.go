package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

type BookingService interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
	AvailableRooms(ctx context.Context, hotelID int64, q model.AvailabilityQuery) (model.AvailableRoomsResponse, error)
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

// Health handles GET /healthcheck.
func (h *BookingHandler) Health(c *gin.Context) {
	if err := h.svc.Ping(c.Request.Context()); err != nil {
		h.logger.Error("health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// FindHotels handles GET /hotels?name=
func (h *BookingHandler) FindHotels(c *gin.Context) {
	hotels, err := h.svc.FindHotels(c.Request.Context(), c.Query("name"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.HotelListResponse{Hotels: hotels})
}

// AvailableRooms handles GET /hotels/:hotelId/rooms/available
func (h *BookingHandler) AvailableRooms(c *gin.Context) {
	hotelID, err := strconv.ParseInt(c.Param("hotelId"), 10, 64)
	if err != nil || hotelID <= 0 {
		h.respondError(c, model.Invalidf("hotelId must be a positive integer"))
		return
	}
	var q model.AvailabilityQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		h.respondError(c, err)
		return
	}
	res, err := h.svc.AvailableRooms(c.Request.Context(), hotelID, q)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
