package handler

import (
	"log/slog"

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

	v1 := r.Group(apiPrefix)
	v1.GET("/hotels", h.FindHotels)

	return r
}
