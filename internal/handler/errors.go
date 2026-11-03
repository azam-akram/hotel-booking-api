package handler

import (
	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

// writeError sends the standard error body and stops the handler.
func writeError(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, model.ErrorResponse{Error: model.ErrorBody{Code: code, Message: msg}})
}
