package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// errorResponse registers the error in gin's error chain (so the logger middleware
// can capture it) and writes the JSON error response.
func errorResponse(c *gin.Context, status int, msg string) {
	_ = c.Error(fmt.Errorf("%s", msg))
	c.JSON(status, gin.H{
		"success": false,
		"error":   msg,
	})
}
