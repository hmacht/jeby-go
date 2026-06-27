package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(validKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")

		if key == "" || key != validKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or missing API key",
			})
			return
		}

		c.Next()
	}
}
