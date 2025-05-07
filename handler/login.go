package handler

import (
	"creator-tool-backend/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"time"
)

func LoginHandler(c *gin.Context) {
	conf := config.InfaConfig{}
	conf.LoadConfig()

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	user, err := GetUserByEmail(c, req.Email)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(401, gin.H{"error": "invalid email or password"})
		return
	}

	// Tạo token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(conf.JWTACCESSKEY))
	if err != nil {
		c.JSON(500, gin.H{"error": "token creation failed"})
		return
	}

	c.JSON(200, gin.H{"token": tokenString})
}

func PingPongHandler(c *gin.Context) {
	print("pong")
	c.JSON(200, gin.H{"ping": "pong"})
}
