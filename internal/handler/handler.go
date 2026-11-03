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

// FindHotels handles GET /hotels?name=
func (h *BookingHandler) FindHotels(c *gin.Context) {
	c.JSON(http.StatusOK, model.HotelListResponse{Hotels: []model.Hotel{}})
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
