package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type TextToSpeechRequest struct {
	Text      string  `json:"text" binding:"required"`
	VoiceName string  `json:"voice_name" binding:"required"`
	Speed     float64 `json:"speed" binding:"required"`
	Pitch     float64 `json:"pitch" binding:"required"`
}

func TextToSpeechHandler(c *gin.Context) {
	var req TextToSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create output directory if it doesn't exist
	outputDir := "./storage/tts"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logrus.Errorf("Failed to create output directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create output directory"})
		return
	}

	// Generate output filename
	filename := filepath.Join(outputDir, "output.mp3")

	// Convert text to speech
	options := service.TTSOptions{
		Text:      req.Text,
		VoiceName: req.VoiceName,
		Speed:     req.Speed,
		Pitch:     req.Pitch,
	}

	audioContent, err := service.TextToSpeech(req.Text, options)
	if err != nil {
		logrus.Errorf("Failed to convert text to speech: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert text to speech"})
		return
	}

	// Save audio file
	if err := os.WriteFile(filename, []byte(audioContent), 0644); err != nil {
		logrus.Errorf("Failed to save audio file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save audio file"})
		return
	}

	// Save to database
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lưu history trước để lấy video_id
	history := config.CaptionHistory{
		UserID:              userID,
		Transcript:          req.Text,
		VideoFilename:       filename,
		VideoFilenameOrigin: "text_to_speech.mp3",
		ProcessType:         "text-to-speech",
		CreatedAt:           time.Now(),
	}

	if err := config.Db.Create(&history).Error; err != nil {
		logrus.Errorf("Failed to save history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}

	videoID := &history.ID
	// Tính chi phí TTS theo số ký tự
	pricingService := service.NewPricingService()
	creditService := service.NewCreditService()
	useWavenet := false // Đổi thành true nếu sử dụng Wavenet voices

	baseCost, err := pricingService.CalculateTTSCost(req.Text, useWavenet)
	if err != nil {
		logrus.Errorf("Không tính được chi phí TTS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tính được chi phí TTS"})
		return
	}

	err = creditService.DeductCredits(
		userID,
		baseCost,
		"tts",
		"Google TTS",
		videoID,
		"character",
		float64(len([]rune(req.Text))),
	)
	if err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit để sử dụng TTS", "warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Text converted to speech successfully",
		"file":    filename,
	})
}
