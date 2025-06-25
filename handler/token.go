package handler

import (
	"creator-tool-backend/config"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Lấy số dư token hiện tại
func GetTokenBalance(c *gin.Context) {
	userID := c.GetUint("user_id")
	var userToken config.UserTokens
	if err := config.Db.Where("user_id = ?", userID).First(&userToken).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"total_tokens": 0, "used_tokens": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total_tokens": userToken.TotalTokens, "used_tokens": userToken.UsedTokens})
}

// Lấy lịch sử giao dịch token
func GetTokenHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	var txs []config.TokenTransaction
	if err := config.Db.Where("user_id = ?", userID).Order("created_at desc").Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

// Nạp token (dùng cho test/dev, thực tế sẽ tích hợp payment)
func AddToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Amount int `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}
	var userToken config.UserTokens
	if err := config.Db.Where("user_id = ?", userID).First(&userToken).Error; err != nil {
		userToken = config.UserTokens{UserID: userID, TotalTokens: req.Amount, UsedTokens: 0}
		config.Db.Create(&userToken)
	} else {
		userToken.TotalTokens += req.Amount
		config.Db.Save(&userToken)
	}
	// Ghi log giao dịch
	tx := config.TokenTransaction{
		UserID:      userID,
		Type:        "add",
		Amount:      req.Amount,
		Description: "Nạp token",
		Service:     "topup",
		CreatedAt:   time.Now(),
	}
	config.Db.Create(&tx)
	c.JSON(http.StatusOK, gin.H{"message": "Nạp token thành công", "total_tokens": userToken.TotalTokens})
}

// Trừ token khi sử dụng dịch vụ
func DeductToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Amount      int    `json:"amount"`
		Service     string `json:"service"`
		Description string `json:"description"`
		VideoID     *uint  `json:"video_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var userToken config.UserTokens
	if err := config.Db.Where("user_id = ?", userID).First(&userToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User chưa có token"})
		return
	}
	if userToken.TotalTokens < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ token"})
		return
	}
	userToken.TotalTokens -= req.Amount
	userToken.UsedTokens += req.Amount
	config.Db.Save(&userToken)
	// Ghi log giao dịch
	tx := config.TokenTransaction{
		UserID:      userID,
		Type:        "deduct",
		Amount:      req.Amount,
		Description: req.Description,
		Service:     req.Service,
		VideoID:     req.VideoID,
		CreatedAt:   time.Now(),
	}
	config.Db.Create(&tx)
	c.JSON(http.StatusOK, gin.H{"message": "Đã trừ token", "total_tokens": userToken.TotalTokens})
}

// Hàm tiện ích trừ token cho handler khác dùng
func DeductUserToken(userID uint, amount int, service, description string, videoID *uint) error {
	var userToken config.UserTokens
	if err := config.Db.Where("user_id = ?", userID).First(&userToken).Error; err != nil {
		return err
	}
	if userToken.TotalTokens < amount {
		return fmt.Errorf("Không đủ token")
	}
	userToken.TotalTokens -= amount
	userToken.UsedTokens += amount
	config.Db.Save(&userToken)
	tx := config.TokenTransaction{
		UserID:      userID,
		Type:        "deduct",
		Amount:      amount,
		Description: description,
		Service:     service,
		VideoID:     videoID,
		CreatedAt:   time.Now(),
	}
	config.Db.Create(&tx)
	return nil
}
