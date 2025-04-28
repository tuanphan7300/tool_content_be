package handler

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"creator-tool-backend/limit"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func UploadHandler(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"ip": c.ClientIP(),
	})

	// Kiểm tra giới hạn Free (5 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 5 video/ngày. Nâng cấp Pro!"})
		return
	}

	// Lấy file
	file, err := c.FormFile("file")
	if err != nil {
		log.WithError(err).Error("Missing video file")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing video file"})
		return
	}

	// Kiểm tra định dạng và kích thước
	if !isValidFile(file.Filename, file.Size) {
		log.Error("Invalid file format or size")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ .mp4/.mov, tối đa 100MB"})
		return
	}

	// Tạo folder storage
	err = os.MkdirAll("storage", os.ModePerm)
	if err != nil {
		log.WithError(err).Error("Could not create storage folder")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create storage folder"})
		return
	}

	// Tạo tên file an toàn
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
	filename = strings.ReplaceAll(filename, " ", "_")
	if len(filename) > 20 {
		filename = filename[:20]
	}
	filePath := filepath.Join("storage", filename)

	// Lưu file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		log.WithError(err).Error("Failed to save file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	// Tách file audio từ video
	audioFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".mp3"
	audioPath := filepath.Join("storage", audioFilename)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-q:a", "0", "-map", "a", audioPath)
	err = cmd.Run()
	if err != nil {
		log.WithError(err).Error("Failed to extract audio")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}

	log.Info("File uploaded and audio extracted successfully")
	c.JSON(http.StatusOK, gin.H{
		"message":       "File uploaded and audio extracted successfully",
		"filePath":      filePath,
		"filename":      filename,
		"audioPath":     audioPath,
		"audioFilename": audioFilename,
	})
}

// isValidFile kiểm tra định dạng và kích thước file
func isValidFile(filename string, size int64) bool {
	ext := filepath.Ext(filename)
	if ext != ".mp4" && ext != ".mov" {
		return false
	}
	if size > 100*1024*1024 { // 100MB
		return false
	}
	return true
}
