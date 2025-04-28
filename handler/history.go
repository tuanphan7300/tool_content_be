package handler

import (
	"creator-tool-backend/config"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"net/http"
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

	// Chuyển timestamps thành JSON
	timestampsJSON, err := json.Marshal(request.Timestamps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal timestamps"})
		return
	}

	// Lưu vào database
	result := config.Db.Create(&config.CaptionHistory{
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
	var histories []config.CaptionHistory
	if err := config.Db.Order("created_at desc").Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, histories)
}

func GetHistoryByID(c *gin.Context) {
	id := c.Param("id")
	var history config.CaptionHistory

	if err := config.Db.First(&history, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
		return
	}
	c.JSON(http.StatusOK, history)
}
