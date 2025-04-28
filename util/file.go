package util

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Processfile(c *gin.Context, file *multipart.FileHeader) (video, audio, fileVideoPath string, audioPath string, err error) {
	if !isValidFile(file.Filename, file.Size) {
		log.Error("Invalid file format or size")
		return "", "", "", "", err
	}

	// Tạo folder storage
	err = os.MkdirAll("storage", os.ModePerm)
	if err != nil {
		log.WithError(err).Error("Could not create storage folder")
		return "", "", "", "", err
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
		return "", "", "", "", err
	}
	// Tách file audio từ video
	audioFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".mp3"
	audioPath = filepath.Join("storage", audioFilename)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-q:a", "0", "-map", "a", audioPath)
	err = cmd.Run()
	if err != nil {
		log.WithError(err).Error("Failed to extract audio")
		return
	}
	return filename, audioFilename, filePath, audioPath, nil

}

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
