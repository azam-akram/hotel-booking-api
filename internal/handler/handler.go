package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

type BookingService interface {
	FindHotels(ctx context.Context, name string) ([]model.Hotel, error)
}

type BookingHandler struct {
	logger *slog.Logger
}

func NewBookHandler(log *slog.Logger) *BookingHandler {
	return &BookingHandler{
		logger: log,
	}
}

// FindHotels handles GET /hotels?name=
func (h *BookingHandler) FindHotels(c *gin.Context) {
	c.JSON(http.StatusOK, model.HotelListResponse{Hotels: []model.Hotel{}})
}

// Health handles GET /healthcheck.
func (h *BookingHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
