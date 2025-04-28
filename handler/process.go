package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/limit"
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func ProcessHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey
	log := logrus.WithFields(logrus.Fields{
		"ip": c.ClientIP(),
	})

	// Kiểm tra giới hạn Free (5 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 3 video/ngày. Nâng cấp Pro!"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}

	// Lưu file tạm
	_, _, _, saveAudioPath, _ := util.Processfile(c, file)
	// Gọi Whisper để lấy transcript
	transcript, segments, err := service.TranscribeWhisperOpenAI(saveAudioPath, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}

	//Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(transcript, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	jsonData, err := json.Marshal(segments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	translatedSegments, err := service.TranslateSegmentsWithGemini(string(jsonData), geminiKey)
	if err != nil {
		fmt.Println("Error translating segments:", err)
		return
	}
	jsonDataVi, err := json.Marshal(translatedSegments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	//// Lưu history
	result := config.Db.Create(&config.CaptionHistory{
		VideoFilename: file.Filename,
		Transcript:    transcript,
		Suggestion:    captionsAndHashtag,
		Segments:      jsonData,
		SegmentsVi:    jsonDataVi,
		CreatedAt:     time.Now(),
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transcript":         transcript,
		"captionsAndHashtag": captionsAndHashtag,
	})
}
