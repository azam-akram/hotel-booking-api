package handler

import (
	"net/http"

	"github.com/azam-akram/hotel-booking-api/internal/model"
	"github.com/gin-gonic/gin"
)

func notFound(c *gin.Context) {
	writeError(c, http.StatusNotFound, model.CodeNotFound, "no such route")
}

func methodNotAllowed(c *gin.Context) {
	writeError(c, http.StatusMethodNotAllowed, model.CodeMethodNotAllowed, "method not allowed")
}
