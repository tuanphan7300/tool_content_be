package util

import (
	"fmt"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Processfile(c *gin.Context, file *multipart.FileHeader) (video, audio, fileVideoPath string, audioPath string, err error) {
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

// GetAudioDuration trả về duration (giây) của file audio/video
func GetAudioDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	durStr := strings.TrimSpace(string(output))
	dur, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, err
	}
	return dur, nil
}

// ProcessfileToDir lưu file và tách audio vào thư mục videoDir
func ProcessfileToDir(c *gin.Context, file *multipart.FileHeader, videoDir string) (audioPath string, err error) {
	// Tạo folder videoDir
	err = os.MkdirAll(videoDir, os.ModePerm)
	if err != nil {
		log.WithError(err).Error("Could not create videoDir folder")
		return "", err
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
		log.WithError(err).Error("Failed to save file")
		return "", err
	}
	// Tách file audio từ video
	audioFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".mp3"
	audioPath = filepath.Join(videoDir, audioFilename)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-q:a", "0", "-map", "a", audioPath)
	err = cmd.Run()
	if err != nil {
		log.WithError(err).Error("Failed to extract audio")
		return "", err
	}
	return audioPath, nil
}

// CleanupDir xóa toàn bộ thư mục và file con, log lại khi xóa
func CleanupDir(dir string) error {
	if dir == "" {
		return nil
	}
	err := os.RemoveAll(dir)
	if err != nil {
		log.WithError(err).Errorf("[CLEANUP] Failed to remove dir: %s", dir)
		return err
	}
	log.Infof("[CLEANUP] Removed dir: %s", dir)
	return nil
}

// ParseSRTFile parses a .srt file and returns segments and transcript
func ParseSRTFile(srtPath string) ([]service.Segment, string, error) {
	segments, err := service.ParseSRTToSegments(srtPath)
	if err != nil {
		return nil, "", err
	}
	var transcriptBuilder strings.Builder
	for _, seg := range segments {
		transcriptBuilder.WriteString(seg.Text)
		transcriptBuilder.WriteString(" ")
	}
	return segments, strings.TrimSpace(transcriptBuilder.String()), nil
}
