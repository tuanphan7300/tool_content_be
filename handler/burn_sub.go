package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"creator-tool-backend/util"

	"github.com/gin-gonic/gin"
)

// BurnSubHandler nhận video + sub, lưu file, trả về process_id
func BurnSubHandler(c *gin.Context) {
	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get target language parameter (default to Vietnamese if not provided)
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi" // Default to Vietnamese
	}

	creditService := service.NewCreditService()

	// --- TÍNH PHÍ burn-sub ---
	// Lấy pricing từ database
	var burnSubPricing config.ServicePricing
	err := config.Db.Where("service_name = ? AND is_active = ?", "burn-sub", true).First(&burnSubPricing).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get burn-sub pricing"})
		return
	}

	// --- LOCK CREDIT TRƯỚC KHI XỬ LÝ ---
	_, err = creditService.LockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Lock credit for burn subtitle", nil)
	if err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit để burn subtitle",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}
	// Đảm bảo unlock nếu có lỗi
	defer func() {
		if r := recover(); r != nil {
			creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to panic", nil)
			panic(r)
		}
	}()

	// Nhận file video
	videoFile, err := c.FormFile("video")
	if err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to missing video file", nil)
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}
	if videoFile.Size > 100*1024*1024 {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to file size limit", nil)
		util.HandleError(c, http.StatusBadRequest, util.ErrFileTooLarge, nil)
		return
	}

	// Nhận file sub
	subFile, err := c.FormFile("subtitle")
	if err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to missing subtitle file", nil)
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}
	if !(strings.HasSuffix(subFile.Filename, ".srt") || strings.HasSuffix(subFile.Filename, ".ass")) {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to invalid subtitle format", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file phụ đề .srt hoặc .ass"})
		return
	}

	// Tạo thư mục lưu file
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(videoFile.Filename), filepath.Ext(videoFile.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(videoFile.Filename))
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to directory creation error", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video directory"})
		return
	}

	// Chuẩn hóa tên file video và sub
	safeVideoName := strings.ReplaceAll(videoFile.Filename, " ", "_")
	videoPath := filepath.Join(videoDir, safeVideoName)
	if err := c.SaveUploadedFile(videoFile, videoPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to video save error", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}
	if _, err := os.Stat(videoPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to video file not found", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Video file not found after save"})
		return
	}

	safeSubName := strings.ReplaceAll(subFile.Filename, " ", "_")
	subPath := filepath.Join(videoDir, safeSubName)
	if err := c.SaveUploadedFile(subFile, subPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to subtitle save error", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subtitle file"})
		return
	}
	if _, err := os.Stat(subPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to subtitle file not found", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Subtitle file not found after save"})
		return
	}

	subtitleColor := c.PostForm("subtitle_color")
	if subtitleColor == "" {
		subtitleColor = "#FFFFFF" // mặc định trắng
	}
	subtitleBgColor := c.PostForm("subtitle_bgcolor")
	if subtitleBgColor == "" {
		subtitleBgColor = "#000000" // mặc định đen
	}

	// Tạo job burn-sub và enqueue vào queue
	jobID := fmt.Sprintf("burnsub_%d_%d", userID, timestamp)
	job := &service.AudioProcessingJob{
		ID:              jobID,
		JobType:         "burn-sub",
		UserID:          userID,
		FileName:        safeVideoName,
		VideoDir:        videoDir,
		SubtitlePath:    subPath,
		Priority:        5,
		MaxDuration:     600, // 10 phút
		SubtitleColor:   subtitleColor,
		SubtitleBgColor: subtitleBgColor,
	}
	queueService := service.GetQueueService()
	if queueService == nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to queue service not initialized", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Queue service not initialized"})
		return
	}
	if err := queueService.EnqueueJob(job); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to enqueue job error", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	// --- TRỪ CREDIT SAU KHI ENQUEUE THÀNH CÔNG ---
	err = creditService.DeductCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Burn subtitle job", nil, "per_job", 1.0)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho burn subtitle",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Tạo CaptionHistory cho burn-sub
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       videoPath,
		VideoFilenameOrigin: videoFile.Filename,
		SrtFile:             subPath,
		ProcessType:         "burn-sub",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("Failed to save burn-sub history: %v", err)
	} else {
		log.Printf("Successfully saved burn-sub history: ID=%d, UserID=%d, ProcessType=%s",
			captionHistory.ID, captionHistory.UserID, captionHistory.ProcessType)
	}
	// Giới hạn 10 action gần nhất cho user
	processService := service.NewProcessStatusService()
	_ = processService.LimitUserCaptionHistories(userID)
	// Lấy tổng số action hiện tại
	var count int64
	config.Db.Model(&config.CaptionHistory{}).Where("user_id = ?", userID).Count(&count)
	deleteAt := captionHistory.CreatedAt.Add(24 * time.Hour)
	var warning string
	if count >= 9 {
		warning = "Bạn chỉ được lưu tối đa 10 kết quả, kết quả cũ nhất sẽ bị xóa khi tạo mới."
	}
	if time.Until(deleteAt) < time.Hour {
		warning = "Kết quả này sẽ bị xóa sau chưa đầy 1 giờ, hãy tải về nếu cần giữ lại."
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "Đã nhận video và phụ đề, đang xử lý...",
		"process_id": jobID,
		"delete_at":  deleteAt,
		"warning":    warning,
	})
}
