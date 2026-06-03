package handlers

import (
	"errors"
	"net/http"

	"multi-platform-distribution/internal/domain"

	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
