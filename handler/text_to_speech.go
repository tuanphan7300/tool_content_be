package handler

import (
	"context"
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
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

// GetAvailableVoicesHandler trả về danh sách giọng đọc có sẵn cho ngôn ngữ
func GetAvailableVoicesHandler(c *gin.Context) {
	language := c.Query("language")
	if language == "" {
		language = "vi" // default to Vietnamese
	}

	// Lấy voice cache service
	voiceCacheService := service.GetVoiceCacheService()
	if voiceCacheService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Voice cache service not initialized"})
		return
	}

	// Lấy tất cả voice samples
	allSamples := voiceCacheService.GetAllVoiceSamples()

	// Lọc theo ngôn ngữ
	var languageSamples []*service.VoiceSample
	for _, sample := range allSamples {
		if sample.LanguageCode[:2] == language { // So sánh 2 ký tự đầu (vi-VN -> vi)
			languageSamples = append(languageSamples, sample)
		}
	}

	if len(languageSamples) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"voices":   languageSamples,
			"language": language,
			"cached":   true,
		})
	} else {
		// Fallback về voice options nếu chưa có cached samples
		voices := service.GetAvailableVoices()
		if languageVoices, exists := voices[language]; exists {
			c.JSON(http.StatusOK, gin.H{
				"voices":   languageVoices,
				"language": language,
				"cached":   false,
			})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Language not supported"})
		}
	}
}

// VoicePreviewHandler tạo audio preview cho giọng đọc
func VoicePreviewHandler(c *gin.Context) {
	var req struct {
		Text         string `json:"text" binding:"required"`
		VoiceName    string `json:"voice_name" binding:"required"`
		LanguageCode string `json:"language_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Tạo thư mục output nếu chưa có
	outputDir := "./storage/voice_preview"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logrus.Errorf("Failed to create output directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create output directory"})
		return
	}

	// Tạo tên file unique
	timestamp := time.Now().UnixNano()
	filename := filepath.Join(outputDir, fmt.Sprintf("preview_%d_%d.mp3", userID, timestamp))

	// Khởi tạo Google TTS client
	ctx := context.Background()
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		credsPath = "data/google_clound_tts_api.json"
	}
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(credsPath))
	if err != nil {
		logrus.Errorf("Failed to create TTS client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TTS client"})
		return
	}
	defer client.Close()

	// Tạo request cho Google TTS
	ttsReq := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: req.Text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: req.LanguageCode,
			Name:         req.VoiceName,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
			SpeakingRate:    1.0, // Tốc độ bình thường cho preview
			SampleRateHertz: 44100,
		},
	}

	// Gọi Google TTS API
	resp, err := client.SynthesizeSpeech(ctx, ttsReq)
	if err != nil {
		// Friendly message for common billing errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "BILLING_DISABLED") || strings.Contains(errMsg, "PermissionDenied") {
			logrus.Errorf("Google TTS billing/permission error: %v", err)
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "Google TTS chưa bật thanh toán hoặc quyền bị hạn chế. Vui lòng bật billing cho dự án GCP dùng trong credentials và thử lại."})
			return
		}
		logrus.Errorf("Google TTS API call failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to synthesize speech"})
		return
	}

	// Lưu audio content vào file
	if err := os.WriteFile(filename, resp.AudioContent, 0644); err != nil {
		logrus.Errorf("Failed to save audio file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save audio file"})
		return
	}

	// Tạo URL để frontend có thể truy cập
	audioURL := fmt.Sprintf("/voice-preview/%d_%d.mp3", userID, timestamp)

	// Lưu history (optional)
	history := config.CaptionHistory{
		UserID:              userID,
		Transcript:          req.Text,
		VideoFilename:       filename,
		VideoFilenameOrigin: "voice_preview.mp3",
		ProcessType:         "voice-preview",
		CreatedAt:           time.Now(),
	}

	if err := config.Db.Create(&history).Error; err != nil {
		logrus.Errorf("Failed to save preview history: %v", err)
		// Không fail nếu không lưu được history
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Voice preview created successfully",
		"audio_url": audioURL,
		"filename":  filename,
	})
}

// RefreshVoiceSamplesHandler refresh tất cả voice samples (admin only)
func RefreshVoiceSamplesHandler(c *gin.Context) {
	// Lấy voice cache service
	voiceCacheService := service.GetVoiceCacheService()
	if voiceCacheService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Voice cache service not initialized"})
		return
	}

	// Refresh voice samples
	if err := voiceCacheService.RefreshVoiceSamples(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh voice samples"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Voice samples refresh started successfully",
		"status":  "refreshing",
	})
}

// TestVoiceCacheHandler test endpoint để kiểm tra voice cache (public)
func TestVoiceCacheHandler(c *gin.Context) {
	language := c.Query("language")
	if language == "" {
		language = "vi"
	}

	// Lấy voice cache service
	voiceCacheService := service.GetVoiceCacheService()
	if voiceCacheService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Voice cache service not initialized"})
		return
	}

	// Lấy tất cả voice samples
	allSamples := voiceCacheService.GetAllVoiceSamples()

	// Lọc theo ngôn ngữ
	var languageSamples []*service.VoiceSample
	for _, sample := range allSamples {
		if sample.LanguageCode[:2] == language {
			languageSamples = append(languageSamples, sample)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"voices":        languageSamples,
		"language":      language,
		"cached":        len(languageSamples) > 0,
		"total_samples": len(allSamples),
		"debug_info": gin.H{
			"cache_service_exists":   voiceCacheService != nil,
			"all_samples_count":      len(allSamples),
			"language_samples_count": len(languageSamples),
		},
	})
}
