package handler

import (
	"errors"
	"net/http"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

// writeError sends the standard error body and stops the handler.
func writeError(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, model.ErrorResponse{Error: model.ErrorBody{Code: code, Message: msg}})
}

func (h *BookingHandler) respondError(c *gin.Context, err error) {
	status, code, msg := http.StatusInternalServerError, model.CodeInternal, "internal server error"

	switch {
	case errors.Is(err, model.ErrInvalidInput):
		status, code, msg = http.StatusBadRequest, model.CodeInvalidInput, err.Error()
	case errors.Is(err, model.ErrPayloadTooLarge):
		status, code, msg = http.StatusRequestEntityTooLarge, model.CodePayloadTooLarge, err.Error()
	case errors.Is(err, model.ErrNotFound):
		status, code, msg = http.StatusNotFound, model.CodeNotFound, err.Error()
	case errors.Is(err, model.ErrRoomUnavailable):
		status, code, msg = http.StatusConflict, model.CodeRoomUnavailable, err.Error()
	case errors.Is(err, model.ErrOverCapacity):
		status, code, msg = http.StatusUnprocessableEntity, model.CodeOverCapacity, err.Error()
	case errors.Is(err, model.ErrIdempotencyMismatch):
		status, code, msg = http.StatusUnprocessableEntity, model.CodeIdempotencyMismatch, err.Error()
	default:
	}

	writeError(c, status, code, msg)
}
