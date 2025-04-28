package handler

import (
	"creator-tool-backend/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
)

var openAIKey = "" // <-- thay bằng key thật của bạn

func CaptionHandler(c *gin.Context) {
	//id := c.Param("id")
	audioPath := c.Param("audioPath")
	filePath := filepath.Join("storage", audioPath)

	text, segments, err := service.TranscribeWhisperOpenAI(filePath, openAIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transcript": text,
		"segments":   segments,
	})
}
