package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetCreditBalance lấy số dư credit của user
func GetCreditBalance(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	creditService := service.NewCreditService()
	balance, err := creditService.GetUserCreditBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get credit balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":  balance,
		"currency": "USD",
	})
}

// GetCreditHistory lấy lịch sử giao dịch credit (theo video)
func GetCreditHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy limit từ query parameter
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	creditService := service.NewCreditService()
	// Lấy danh sách transaction type = 'deduct' có video_id
	transactions, err := creditService.GetTransactionHistory(userID, 1000) // lấy nhiều để group
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction history"})
		return
	}

	// Group theo video_id, chỉ lấy transaction có video_id != nil
	type VideoCost struct {
		VideoID       uint      `json:"video_id"`
		VideoFilename string    `json:"video_filename"`
		CreatedAt     time.Time `json:"created_at"`
		TotalCost     float64   `json:"total_cost"`
	}
	videoMap := map[uint]*VideoCost{}
	// Lấy map video_id -> video_filename từ CaptionHistory
	var histories []config.CaptionHistory
	config.Db.Where("user_id = ?", userID).Find(&histories)
	videoNameMap := map[uint]string{}
	createdAtMap := map[uint]time.Time{}
	for _, h := range histories {
		videoNameMap[h.ID] = h.VideoFilenameOrigin
		createdAtMap[h.ID] = h.CreatedAt
	}
	for _, tx := range transactions {
		if tx.TransactionType != "deduct" || tx.VideoID == nil {
			continue
		}
		vid := *tx.VideoID
		if _, ok := videoMap[vid]; !ok {
			videoMap[vid] = &VideoCost{
				VideoID:       vid,
				VideoFilename: videoNameMap[vid],
				CreatedAt:     createdAtMap[vid],
				TotalCost:     0,
			}
		}
		videoMap[vid].TotalCost += tx.Amount
	}
	// Chuyển sang slice và sort theo created_at desc
	var result []VideoCost
	for _, v := range videoMap {
		result = append(result, *v)
	}
	// Sort
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": result,
		"count":  len(result),
	})
}

// AddCredits thêm credit cho user (dùng cho test/dev)
func AddCredits(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Description string  `json:"description"`
		ReferenceID string  `json:"reference_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validation
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	if req.Amount > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount cannot exceed $1000"})
		return
	}

	creditService := service.NewCreditService()
	err := creditService.AddCredits(userID, req.Amount, req.Description, req.ReferenceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add credits"})
		return
	}

	// Lấy balance mới
	balance, err := creditService.GetUserCreditBalance(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Credits added successfully",
			"amount":  req.Amount,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Credits added successfully",
		"amount":      req.Amount,
		"new_balance": balance,
	})
}

// EstimateCost ước tính chi phí cho process-video
func EstimateCost(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		DurationMinutes  float64 `json:"duration_minutes" binding:"required,gt=0"`
		TranscriptLength int     `json:"transcript_length"`
		SrtLength        int     `json:"srt_length"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validation
	if req.DurationMinutes <= 0 || req.DurationMinutes > 600 { // Max 10 hours
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duration must be between 0 and 600 minutes"})
		return
	}

	// Ước tính transcript và SRT length nếu không được cung cấp
	if req.TranscriptLength == 0 {
		// Ước tính: 150 từ/phút, mỗi từ 5 ký tự
		req.TranscriptLength = int(req.DurationMinutes * 150 * 5)
	}
	if req.SrtLength == 0 {
		// Ước tính: SRT dài hơn transcript 20% do format
		req.SrtLength = int(float64(req.TranscriptLength) * 1.2)
	}

	creditService := service.NewCreditService()
	estimates, err := creditService.EstimateTotalCost(req.DurationMinutes, req.TranscriptLength, req.SrtLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to estimate cost"})
		return
	}

	// Lấy credit balance của user
	balance, err := creditService.GetUserCreditBalance(userID)
	if err != nil {
		balance = map[string]float64{"available_credits": 0}
	}

	c.JSON(http.StatusOK, gin.H{
		"estimates":          estimates,
		"user_balance":       balance,
		"sufficient_credits": balance["available_credits"] >= estimates["total"],
		"currency":           "USD",
	})
}
