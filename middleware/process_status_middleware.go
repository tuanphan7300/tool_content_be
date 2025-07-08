package middleware

import (
	"creator-tool-backend/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

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
