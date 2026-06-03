package middleware

import (
	"net/http"
	"strings"

	"multi-platform-distribution/internal/auth"

	"github.com/gin-gonic/gin"
)

func AdminAuth(apiKey string, tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey != "" && c.GetHeader("X-API-Key") == apiKey {
			c.Next()
			return
		}

		header := strings.TrimSpace(c.GetHeader("Authorization"))
		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if header == tokenString || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin authentication required"})
			return
		}

		claims, err := tokens.Verify(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}

		c.Set("admin_id", claims.AdminID.String())
		c.Set("admin_email", claims.Email)
		c.Next()
	}
}
