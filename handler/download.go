package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DownloadFileHandler serves files for download with proper headers
func DownloadFileHandler(c *gin.Context) {
	// Get file path from URL parameter
	filePath := c.Param("filepath")

	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File path is required",
		})
		return
	}

	// Sanitize file path để tránh directory traversal attacks
	filePath = strings.TrimPrefix(filePath, "/")
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file path",
		})
		return
	}

	// Build full file path
	fullPath := filepath.Join("./storage", filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	}

	// Get file info
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get file info",
		})
		return
	}

	// Set appropriate headers for download
	filename := fileInfo.Name()

	// Set Content-Disposition header để force download
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Set Content-Type based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4":
		c.Header("Content-Type", "video/mp4")
	case ".avi":
		c.Header("Content-Type", "video/avi")
	case ".mov":
		c.Header("Content-Type", "video/quicktime")
	case ".mkv":
		c.Header("Content-Type", "video/x-matroska")
	case ".mp3":
		c.Header("Content-Type", "audio/mpeg")
	case ".wav":
		c.Header("Content-Type", "audio/wav")
	case ".srt":
		c.Header("Content-Type", "text/plain")
	case ".txt":
		c.Header("Content-Type", "text/plain")
	case ".json":
		c.Header("Content-Type", "application/json")
	default:
		c.Header("Content-Type", "application/octet-stream")
	}

	// Set cache headers
	c.Header("Cache-Control", "public, max-age=3600")

	// Serve the file
	c.File(fullPath)
}
