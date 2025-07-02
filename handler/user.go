package handler

import (
	"creator-tool-backend/config"

	"github.com/gin-gonic/gin"
)

// GetUserProfileHandler returns the current user's profile
func GetUserProfileHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := GetUserByID(c, userID.(uint))
	if err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	// Don't return sensitive information
	user.PasswordHash = ""

	c.JSON(200, gin.H{
		"user": user,
	})
}

// GetUserByID retrieves user by ID
func GetUserByID(c *gin.Context, userID uint) (config.Users, error) {
	var user config.Users
	if err := config.Db.First(&user, userID).Error; err != nil {
		return user, err
	}
	return user, nil
}
