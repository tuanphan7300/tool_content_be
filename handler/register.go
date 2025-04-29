package handler

import (
	"creator-tool-backend/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(c *gin.Context) {
	var req SaveRegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	user, err := GetUserByEmail(c, req.Email)
	if err == nil && user.ID != 0 {
		c.JSON(400, gin.H{"error": "email đã tồn tại"})
		return
	}
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "server error"})
		return
	}
	req.Password = string(hash)
	result := CreateUser(c, req)
	if !result {
		c.JSON(500, gin.H{"error": "Tạo user thất bại"})
		return
	}
	c.JSON(200, gin.H{"message": "register success"})
}

type SaveRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func CreateUser(c *gin.Context, request SaveRegisterRequest) bool {
	result := config.Db.Create(&config.Users{
		Email:        request.Email,
		PasswordHash: request.Password,
	})

	if result.Error != nil {
		return false
	}
	return true
}

func GetUserByEmail(c *gin.Context, email string) (config.Users, error) {
	var user config.Users
	if err := config.Db.Where("email = ?", email).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}
