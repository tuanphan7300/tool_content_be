package middleware

import (
	"net/http"
	"strings"

	"creator-tool-backend/config"

	"github.com/gin-gonic/gin"
)

// SepayAuthMiddleware xác thực API Key từ Sepay
func SepayAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		infaConfig := config.InfaConfig{}
		infaConfig.LoadConfig()
		// Lấy API Key từ header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing Authorization header",
			})
			c.Abort()
			return
		}

		// Kiểm tra format "Apikey API_KEY"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Apikey" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Authorization header format. Expected: 'Apikey API_KEY'",
			})
			c.Abort()
			return
		}

		apiKey := parts[1]
		expectedApiKey := infaConfig.SepayApiKey

		// Kiểm tra API Key có hợp lệ không
		if expectedApiKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Sepay API Key not configured",
			})
			c.Abort()
			return
		}

		if apiKey != expectedApiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API Key",
			})
			c.Abort()
			return
		}

		// API Key hợp lệ, tiếp tục
		c.Next()
	}
}
