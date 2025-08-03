package middleware

import (
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FileValidationMiddleware kiểm tra file size và duration trước khi tạo process status
func FileValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy file video
		videoFile, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file video"})
			c.Abort()
			return
		}

		// Kiểm tra kích thước file không quá 100MB
		if videoFile.Size > 100*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File quá lớn. Chỉ cho phép file dưới 100MB."})
			c.Abort()
			return
		}

		// Validate file type
		if !strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mp4") &&
			!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".avi") &&
			!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mov") &&
			!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mkv") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file video (mp4, avi, mov, mkv)"})
			c.Abort()
			return
		}

		// Tạo thư mục tạm để kiểm tra duration
		timestamp := time.Now().UnixNano()
		uniqueName := fmt.Sprintf("%d_%s", timestamp, strings.TrimSuffix(videoFile.Filename, filepath.Ext(videoFile.Filename)))
		tempDir := filepath.Join("storage", uniqueName)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo thư mục làm việc"})
			c.Abort()
			return
		}

		// Lưu file video tạm để kiểm tra duration
		tempVideoPath := filepath.Join(tempDir, videoFile.Filename)
		if err := c.SaveUploadedFile(videoFile, tempVideoPath); err != nil {
			util.CleanupDir(tempDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file video"})
			c.Abort()
			return
		}

		// Trích xuất audio tạm để kiểm tra duration
		tempAudioPath, err := service.ProcessVideoToAudio(tempVideoPath, tempDir)
		if err != nil {
			util.CleanupDir(tempDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể trích xuất audio: %v", err)})
			c.Abort()
			return
		}

		// Kiểm tra duration < 7 phút
		duration, _ := util.GetAudioDuration(tempAudioPath)
		if duration > 420 {
			util.CleanupDir(tempDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
			c.Abort()
			return
		}

		// Lưu thông tin file đã validate vào context
		c.Set("validated_file", videoFile)
		c.Set("temp_dir", tempDir)
		c.Set("temp_video_path", tempVideoPath)
		c.Set("temp_audio_path", tempAudioPath)
		c.Set("file_duration", duration)

		c.Next()
	}
}

// ProcessStatusMiddleware kiểm tra trạng thái process của user
func ProcessStatusMiddleware(processType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy user_id từ token
		userID := c.GetUint("user_id")
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Tạo service để kiểm tra trạng thái
		processService := service.NewProcessStatusService()

		// Kiểm tra xem user có process nào đang chạy không
		canStart, activeProcess, err := processService.CheckUserProcessStatus(userID, processType)
		if err != nil {
			logrus.WithError(err).Error("Error checking user process status")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		if !canStart {
			// User có process đang chạy
			timeSinceStart := time.Since(activeProcess.StartedAt)
			remainingTime := 10*time.Minute - timeSinceStart

			c.JSON(http.StatusConflict, gin.H{
				"error": "Bạn đang có một quá trình xử lý đang chạy. Vui lòng đợi quá trình hiện tại hoàn thành.",
				"details": gin.H{
					"process_id":     activeProcess.ID,
					"process_type":   activeProcess.ProcessType,
					"started_at":     activeProcess.StartedAt,
					"time_elapsed":   timeSinceStart.String(),
					"remaining_time": remainingTime.String(),
				},
			})
			c.Abort()
			return
		}

		// User có thể bắt đầu process mới
		// Tạo record mới cho process
		processStatus, err := processService.StartProcess(userID, processType)
		if err != nil {
			logrus.WithError(err).Error("Error starting process")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Lưu process_id vào context để handler có thể sử dụng
		c.Set("process_id", processStatus.ID)
		c.Set("process_status", processStatus)

		c.Next()
	}
}

// ProcessAnyStatusMiddleware kiểm tra nếu user có bất kỳ process nào đang chạy
func ProcessAnyStatusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		processService := service.NewProcessStatusService()
		activeProcesses, err := processService.GetUserActiveProcesses(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}
		if len(activeProcesses) > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error":            "Bạn đang có một quá trình xử lý khác đang chạy. Vui lòng đợi hoàn thành trước khi thực hiện thao tác mới.",
				"active_processes": activeProcesses,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
