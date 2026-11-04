package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

func notFound(c *gin.Context) {
	writeError(c, http.StatusNotFound, model.CodeNotFound, "no such route")
}

func methodNotAllowed(c *gin.Context) {
	writeError(c, http.StatusMethodNotAllowed, model.CodeMethodNotAllowed, "method not allowed")
}

// requestLogger writes one log line per request once it has finished.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		log.LogAttrs(c.Request.Context(), level, "request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("query", c.Request.URL.RawQuery),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
