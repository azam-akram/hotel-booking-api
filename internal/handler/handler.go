package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

type BookingService interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
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

func (h *BookingHandler) FindHotels(c *gin.Context) {
	hotels, err := h.svc.FindHotels(c.Request.Context(), c.Query("name"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.HotelListResponse{Hotels: hotels})
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
