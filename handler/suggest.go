package handler

import (
	"net/http"
	"path/filepath"

	"creator-tool-backend/service"
	"github.com/gin-gonic/gin"
)

func SuggestHandler(c *gin.Context) {
	id := c.Param("id")
	filePath := filepath.Join("storage", id)

	// Gọi lại Whisper để lấy transcript (bây giờ nhận về text + segments)
	text, _, err := service.TranscribeWhisperOpenAI(filePath, openAIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper failed: " + err.Error()})
		return
	}

	// Gọi GPT để gợi ý caption
	suggestion, err := service.GenerateSuggestion(text, openAIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transcript": text,
		"suggestion": suggestion,
	})
}
