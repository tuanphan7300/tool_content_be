package handler

import (
	"fmt"
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
	// Nhận file video trước để kiểm tra sớm
	videoFile, err := c.FormFile("file")
	if err != nil {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}

	// Kiểm tra kích thước file không quá 100MB
	if videoFile.Size > 100*1024*1024 {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileTooLarge, nil)
		return
	}

	// Nhận file sub
	subFile, err := c.FormFile("subtitle")
	if err != nil {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}
	if !(strings.HasSuffix(subFile.Filename, ".srt") || strings.HasSuffix(subFile.Filename, ".ass")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file phụ đề .srt hoặc .ass"})
		return
	}

	// Tạo thư mục tạm để kiểm tra duration
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(videoFile.Filename), filepath.Ext(videoFile.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(videoFile.Filename))
	tempDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video directory"})
		return
	}

	// Lưu file video tạm để kiểm tra duration
	safeVideoName := strings.ReplaceAll(videoFile.Filename, " ", "_")
	tempVideoPath := filepath.Join(tempDir, safeVideoName)
	if err := c.SaveUploadedFile(videoFile, tempVideoPath); err != nil {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}

	// Kiểm tra duration < 7 phút
	tempAudioPath, err := util.ProcessfileToDir(c, videoFile, tempDir)
	if err != nil {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}
	duration, _ := util.GetAudioDuration(tempAudioPath)
	if duration > 420 {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
		return
	}

	// Nếu pass tất cả kiểm tra, tiếp tục xử lý
	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		util.CleanupDir(tempDir)
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
	err = config.Db.Where("service_name = ? AND is_active = ?", "burn-sub", true).First(&burnSubPricing).Error
	if err != nil {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get burn-sub pricing"})
		return
	}

	// --- LOCK CREDIT TRƯỚC KHI XỬ LÝ ---
	_, err = creditService.LockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Lock credit for burn subtitle", nil)
	if err != nil {
		util.CleanupDir(tempDir)
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

	// Sử dụng thư mục tạm đã tạo
	videoDir := tempDir
	videoPath := tempVideoPath

	// Lưu file sub vào thư mục tạm
	safeSubName := strings.ReplaceAll(subFile.Filename, " ", "_")
	subPath := filepath.Join(videoDir, safeSubName)
	if err := c.SaveUploadedFile(subFile, subPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to subtitle save error", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subtitle file"})
		return
	}
	if _, err := os.Stat(videoPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to video file not found", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Video file not found after save"})
		return
	}

	if _, err := os.Stat(subPath); err != nil {
		creditService.UnlockCredits(userID, burnSubPricing.PricePerUnit, "burn-sub", "Unlock due to subtitle file not found", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Subtitle file not found after save"})
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
		subtitleBgColor = "#808080" // mặc định xám nhạt thay vì đen
	}

	// Tạo job burn-sub và enqueue vào queue
	jobID := fmt.Sprintf("burnsub_%d_%d", userID, timestamp)
	processID := c.GetUint("process_id") // Lấy process_id từ middleware
	job := &service.AudioProcessingJob{
		ID:              jobID,
		JobType:         "burn-sub",
		UserID:          userID,
		ProcessID:       processID,
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

	// Không tạo CaptionHistory ở đây - Worker Service sẽ tạo khi xử lý xong
	// Giới hạn 10 action gần nhất cho user
	processService := service.NewProcessStatusService()
	_ = processService.LimitUserCaptionHistories(userID)
	// Lấy tổng số action hiện tại
	var count int64
	config.Db.Model(&config.CaptionHistory{}).Where("user_id = ?", userID).Count(&count)
	deleteAt := time.Now().Add(24 * time.Hour)
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
