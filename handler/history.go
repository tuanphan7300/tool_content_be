package handler

import (
	"creator-tool-backend/config"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type SaveHistoryRequest struct {
	VideoFilename string          `json:"video_filename"`
	Transcript    string          `json:"transcript"`
	Suggestion    string          `json:"suggestion"`
	Segments      json.RawMessage `json:"segments"`
	Timestamps    []string        `json:"timestamps"`
}

func SaveHistory(c *gin.Context) {
	var request SaveHistoryRequest

	// Kiểm tra dữ liệu đầu vào
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Chuyển timestamps thành JSON
	timestampsJSON, err := json.Marshal(request.Timestamps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal timestamps"})
		return
	}

	// Lưu vào database
	result := config.Db.Create(&config.CaptionHistory{
		UserID:        userID,
		VideoFilename: request.VideoFilename,
		Transcript:    request.Transcript,
		Suggestion:    request.Suggestion,
		Timestamps:    datatypes.JSON(timestampsJSON),
		Segments:      datatypes.JSON(request.Segments),
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History saved"})
}

func GetHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	var histories []config.CaptionHistory
	if err := config.Db.Where("user_id = ?", userID).Order("created_at desc").Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, histories)
}

func GetHistoryByID(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var history config.CaptionHistory

	if err := config.Db.Where("id = ? AND user_id = ?", id, userID).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
		return
	}
	c.JSON(http.StatusOK, history)
}

func GetHistoryHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get history from database
	var histories []config.CaptionHistory
	if err := config.Db.Where("user_id = ?", userID).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
		return
	}

	// Format response to include only filenames without paths
	type HistoryResponse struct {
		ID              uint      `json:"id"`
		VideoFilename   string    `json:"video_filename"`
		Transcript      string    `json:"transcript"`
		Segments        string    `json:"segments"`
		SegmentsVi      string    `json:"segments_vi"`
		BackgroundMusic string    `json:"background_music"`
		SrtFile         string    `json:"srt_file"`
		OriginalSrtFile string    `json:"original_srt_file"`
		TTSFile         string    `json:"tts_file"`
		MergedVideoFile string    `json:"merged_video_file"`
		CreatedAt       time.Time `json:"created_at"`
	}

	var response []HistoryResponse
	for _, history := range histories {
		response = append(response, HistoryResponse{
			ID:              history.ID,
			VideoFilename:   history.VideoFilename,
			Transcript:      history.Transcript,
			Segments:        string(history.Segments),
			SegmentsVi:      string(history.SegmentsVi),
			BackgroundMusic: filepath.Base(history.BackgroundMusic),
			SrtFile:         filepath.Base(history.SrtFile),
			OriginalSrtFile: filepath.Base(history.OriginalSrtFile),
			TTSFile:         filepath.Base(history.TTSFile),
			MergedVideoFile: filepath.Base(history.MergedVideoFile),
			CreatedAt:       history.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}
