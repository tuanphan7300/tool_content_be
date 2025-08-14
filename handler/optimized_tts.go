package handler

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// OptimizedTTSRequest request cho optimized TTS
type OptimizedTTSRequest struct {
	Text             string  `json:"text" binding:"required"`
	TargetLanguage   string  `json:"target_language"`
	ServiceName      string  `json:"service_name"`
	SubtitleColor    string  `json:"subtitle_color"`
	SubtitleBgColor  string  `json:"subtitle_bgcolor"`
	BackgroundVolume float64 `json:"background_volume"`
	TTSVolume        float64 `json:"tts_volume"`
	SpeakingRate     float64 `json:"speaking_rate"`
	MaxConcurrent    int     `json:"max_concurrent"`
}

// OptimizedTTSResponse response từ optimized TTS
type OptimizedTTSResponse struct {
	Message       string                 `json:"message"`
	JobID         string                 `json:"job_id"`
	Status        string                 `json:"status"`
	Progress      map[string]interface{} `json:"progress"`
	EstimatedTime string                 `json:"estimated_time"`
	AudioPath     string                 `json:"audio_path,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// OptimizedTTSHandler xử lý TTS với concurrent processing
func OptimizedTTSHandler(c *gin.Context) {
	var req OptimizedTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Set default values
	if req.TargetLanguage == "" {
		req.TargetLanguage = "vi"
	}
	if req.ServiceName == "" {
		req.ServiceName = "gpt-4o-mini"
	}
	if req.SubtitleColor == "" {
		req.SubtitleColor = "#FFFFFF"
	}
	if req.SubtitleBgColor == "" {
		req.SubtitleBgColor = "#808080"
	}
	if req.BackgroundVolume == 0 {
		req.BackgroundVolume = 1.2
	}
	if req.TTSVolume == 0 {
		req.TTSVolume = 1.5
	}
	if req.SpeakingRate == 0 {
		req.SpeakingRate = 1.2
	}
	if req.MaxConcurrent == 0 {
		req.MaxConcurrent = 15 // Mặc định 15 workers
	}

	// Tạo job ID
	jobID := fmt.Sprintf("optimized_tts_%d_%d", userID, time.Now().UnixNano())

	// Tạo thư mục output
	outputDir := fmt.Sprintf("./storage/optimized_tts_%s", jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create output directory"})
		return
	}

	// Khởi tạo optimized TTS service
	ttsService, err := service.InitOptimizedTTSService("data/google_clound_tts_api.json", req.MaxConcurrent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize TTS service: " + err.Error()})
		return
	}

	// Tạo options
	options := service.TTSProcessingOptions{
		TargetLanguage:   req.TargetLanguage,
		ServiceName:      req.ServiceName,
		SubtitleColor:    req.SubtitleColor,
		SubtitleBgColor:  req.SubtitleBgColor,
		BackgroundVolume: req.BackgroundVolume,
		TTSVolume:        req.TTSVolume,
		SpeakingRate:     req.SpeakingRate,
		MaxConcurrent:    req.MaxConcurrent,
		UserID:           userID,
	}

	// Xử lý TTS với concurrent processing
	go func() {
		audioPath, err := ttsService.ProcessSRTConcurrent(req.Text, outputDir, options, jobID)
		if err != nil {
			logrus.Errorf("TTS processing failed for job %s: %v", jobID, err)
			return
		}
		logrus.Infof("TTS processing completed for job %s: %s", jobID, audioPath)
	}()

	// Trả về response ngay lập tức
	response := OptimizedTTSResponse{
		Message:       "TTS processing started with concurrent processing",
		JobID:         jobID,
		Status:        "processing",
		EstimatedTime: "2-5 minutes depending on text length",
	}

	c.JSON(http.StatusOK, response)
}

// GetOptimizedTTSProgress lấy tiến độ xử lý TTS
func GetOptimizedTTSProgress(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy TTS service
	ttsService := service.GetOptimizedTTSService()
	if ttsService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTS service not initialized"})
		return
	}

	// Lấy tiến độ xử lý
	progress := ttsService.GetProcessingStatus(jobID)
	if progress["error"] != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": progress["error"]})
		return
	}

	// Kiểm tra quyền truy cập (chỉ user tạo job mới được xem)
	// TODO: Implement proper access control

	c.JSON(http.StatusOK, gin.H{
		"job_id":    jobID,
		"progress":  progress,
		"timestamp": time.Now(),
	})
}

// GetOptimizedTTSResult lấy kết quả TTS đã hoàn thành
func GetOptimizedTTSResult(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy TTS service
	ttsService := service.GetOptimizedTTSService()
	if ttsService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTS service not initialized"})
		return
	}

	// Lấy tiến độ xử lý
	progress := ttsService.GetProcessingStatus(jobID)
	if progress["error"] != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": progress["error"]})
		return
	}

	// Kiểm tra xem job đã hoàn thành chưa
	status, ok := progress["status"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid progress status"})
		return
	}

	if status != "completed" {
		c.JSON(http.StatusOK, gin.H{
			"job_id":   jobID,
			"status":   status,
			"progress": progress,
			"message":  "Job is still processing",
		})
		return
	}

	// Job đã hoàn thành, trả về audio file
	audioPath := fmt.Sprintf("./storage/optimized_tts_%s/tts_output.mp3", jobID)

	// Kiểm tra file có tồn tại không
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audio file not found"})
		return
	}

	// Trả về file audio
	c.File(audioPath)
}

// GetOptimizedTTSStatistics lấy thống kê TTS (admin only)
func GetOptimizedTTSStatistics(c *gin.Context) {
	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Kiểm tra quyền admin

	// Lấy TTS service
	ttsService := service.GetOptimizedTTSService()
	if ttsService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTS service not initialized"})
		return
	}

	// Lấy thống kê
	stats := ttsService.GetServiceStatistics()

	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
		"timestamp":  time.Now(),
	})
}

// CancelOptimizedTTSJob hủy job TTS đang xử lý
func CancelOptimizedTTSJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Implement job cancellation logic
	// Cần thêm cancel channel và context cancellation

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancellation requested",
		"job_id":  jobID,
		"note":    "Job cancellation is not yet implemented",
	})
}
