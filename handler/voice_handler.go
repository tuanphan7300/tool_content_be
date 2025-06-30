package handler

import (
	"creator-tool-backend/service"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func ProcessVoiceHandler(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("video")
	if err != nil {
		log.Printf("Error getting video file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No video file provided",
		})
		return
	}

	log.Printf("Received video file: %s, size: %d bytes", file.Filename, file.Size)

	// Tạo thư mục riêng cho video
	videoBase := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	videoDir := filepath.Join("./storage", videoBase)
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		log.Printf("Error creating video directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create video directory",
		})
		return
	}

	// Save the uploaded file
	videoPath := filepath.Join(videoDir, file.Filename)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		log.Printf("Error saving video file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save video file",
		})
		return
	}

	log.Printf("Video file saved to: %s", videoPath)

	// Process the video to extract audio
	audioPath, err := service.ProcessVideoToAudio(videoPath, videoDir)
	if err != nil {
		log.Printf("Error processing video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process video: %v", err),
		})
		return
	}

	// Extract vocals
	vocalsPath, err := service.ExtractVocals(audioPath, file.Filename, videoDir)
	if err != nil {
		log.Printf("Error extracting vocals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to extract vocals: %v", err),
		})
		return
	}

	log.Printf("Vocals extracted successfully to: %s", vocalsPath)

	// Return the vocals file for download
	c.File(vocalsPath)
}

func ProcessBackgroundMusicHandler(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("video")
	if err != nil {
		log.Printf("Error getting video file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No video file provided",
		})
		return
	}

	log.Printf("Received video file: %s, size: %d bytes", file.Filename, file.Size)

	// Tạo thư mục riêng cho video
	videoBase := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	videoDir := filepath.Join("./storage", videoBase)
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		log.Printf("Error creating video directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create video directory",
		})
		return
	}

	// Save the uploaded file
	videoPath := filepath.Join(videoDir, file.Filename)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		log.Printf("Error saving video file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save video file",
		})
		return
	}

	log.Printf("Video file saved to: %s", videoPath)

	// Process the video to extract audio
	audioPath, err := service.ProcessVideoToAudio(videoPath, videoDir)
	if err != nil {
		log.Printf("Error processing video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process video: %v", err),
		})
		return
	}

	// Extract background music
	backgroundPath, err := service.ExtractBackgroundMusic(audioPath, file.Filename, videoDir)
	if err != nil {
		log.Printf("Error extracting background music: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to extract background music: %v", err),
		})
		return
	}

	log.Printf("Background music extracted successfully to: %s", backgroundPath)

	// Return the background music file for download
	c.File(backgroundPath)
}
