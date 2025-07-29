package middleware

import (
	"creator-tool-backend/config"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Load JWT secret key from config
		conf := config.InfaConfig{}
		conf.LoadConfig()

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(conf.JWTACCESSKEY), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token: " + err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Convert user_id to float64 first (since JWT numbers are stored as float64)
			if userID, ok := claims["user_id"].(float64); ok {
				// Convert to uint and set in context
				c.Set("user_id", uint(userID))
				c.Next()
			} else {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid user_id in token"})
			}
		} else {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token claims"})
		}
	}
}

// DatabaseMiddleware adds database connection to context
func DatabaseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get database connection from config
		db := config.Db
		c.Set("db", db)
		c.Next()
	}
}
