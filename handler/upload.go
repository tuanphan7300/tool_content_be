package handler

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func UploadHandler(c *gin.Context) {
	// Lấy file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing video file"})
		return
	}

	// Kiểm tra định dạng và kích thước
	if !isValidFile(file.Filename, file.Size) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ .mp4/.mov, tối đa 100MB"})
		return
	}

	// Tạo thư mục riêng cho video
	videoBase := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	videoDir := filepath.Join("storage", videoBase)
	if err := os.MkdirAll(videoDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create videoDir folder"})
		return
	}
	// Tạo tên file an toàn
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
	filename = strings.ReplaceAll(filename, " ", "_")
	if len(filename) > 20 {
		filename = filename[:20]
	}
	filePath := filepath.Join(videoDir, filename)
	// Lưu file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	// Tách file audio từ video
	audioFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".mp3"
	audioPath := filepath.Join(videoDir, audioFilename)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-q:a", "0", "-map", "a", audioPath)
	err = cmd.Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}
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
