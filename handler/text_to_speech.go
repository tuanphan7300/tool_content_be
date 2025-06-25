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

	// Trừ token cho Google TTS
	tokenPerChar := 1.0 / 62.5
	tokens := int(float64(len(req.Text))*tokenPerChar + 0.9999) // làm tròn lên
	if tokens < 1 {
		tokens = 1
	}
	if err := DeductUserToken(userID, tokens, "tts", "Google TTS", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho TTS"})
		return
	}

	history := config.CaptionHistory{
		UserID:        userID,
		Transcript:    req.Text,
		VideoFilename: filename,
		CreatedAt:     time.Now(),
	}

	if err := config.Db.Create(&history).Error; err != nil {
		logrus.Errorf("Failed to save history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Text converted to speech successfully",
		"file":    filename,
	})
}
