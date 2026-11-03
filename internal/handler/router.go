package handler

import (
	"log/slog"
	"net/http"

	"github.com/azam-akram/hotel-booking-api/internal/docs"
	"github.com/gin-gonic/gin"
)

const apiPrefix = "/v1"

func NewRouter(h *BookingHandler, log *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.NoRoute(notFound)
	r.NoMethod(methodNotAllowed)

	r.GET("/healthcheck", h.Health)

	// Swagger documentation
	r.GET("/openapi.yml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", docs.Spec)
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.UIPage))
	})

	v1 := r.Group(apiPrefix)
	v1.GET("/hotels", h.FindHotels)

	return r
}
