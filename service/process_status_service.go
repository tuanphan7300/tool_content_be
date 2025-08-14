package service

import (
	"creator-tool-backend/config"
	"path/filepath"
	"time"

	"os"

	"gorm.io/gorm"
)

// ProcessStatusService quản lý trạng thái process của user
type ProcessStatusService struct{}

// NewProcessStatusService tạo instance mới của ProcessStatusService
func NewProcessStatusService() *ProcessStatusService {
	return &ProcessStatusService{}
}

// CheckUserProcessStatus kiểm tra xem user có đang có process nào đang chạy không
// Trả về true nếu user có thể bắt đầu process mới, false nếu đang có process đang chạy
func (s *ProcessStatusService) CheckUserProcessStatus(userID uint, processType string) (bool, *config.UserProcessStatus, error) {
	var processStatus config.UserProcessStatus

	// Tìm process đang chạy của user
	err := config.Db.Where("user_id = ? AND status = ? AND process_type = ?",
		userID, "processing", processType).
		Order("started_at DESC").
		First(&processStatus).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Không có process nào đang chạy
			return true, nil, nil
		}
		return false, nil, err
	}

	// Kiểm tra xem process có bị treo quá 10 phút không
	timeSinceStart := time.Since(processStatus.StartedAt)
	if timeSinceStart > 10*time.Minute {
		// Process bị treo quá 10 phút, cho phép user bắt đầu process mới
		// Cập nhật trạng thái process cũ thành failed
		s.UpdateProcessStatus(processStatus.ID, "failed")
		return true, nil, nil
	}

	// User có process đang chạy và chưa quá 10 phút
	return false, &processStatus, nil
}

// StartProcess tạo record mới cho process đang bắt đầu
func (s *ProcessStatusService) StartProcess(userID uint, processType string) (*config.UserProcessStatus, error) {
	processStatus := config.UserProcessStatus{
		UserID:      userID,
		Status:      "processing",
		ProcessType: processType,
		StartedAt:   time.Now(),
	}

	err := config.Db.Create(&processStatus).Error
	if err != nil {
		return nil, err
	}

	return &processStatus, nil
}

// UpdateProcessStatus cập nhật trạng thái của process
func (s *ProcessStatusService) UpdateProcessStatus(processID uint, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "completed" || status == "failed" || status == "cancelled" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	return config.Db.Model(&config.UserProcessStatus{}).
		Where("id = ?", processID).
		Updates(updates).Error
}

// UpdateProcessVideoID cập nhật video_id cho process
func (s *ProcessStatusService) UpdateProcessVideoID(processID uint, videoID uint) error {
	return config.Db.Model(&config.UserProcessStatus{}).
		Where("id = ?", processID).
		Update("video_id", videoID).Error
}

// CleanupStaleProcesses dọn dẹp các process bị treo quá 10 phút
func (s *ProcessStatusService) CleanupStaleProcesses() error {
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)

	return config.Db.Model(&config.UserProcessStatus{}).
		Where("status = ? AND started_at < ?", "processing", tenMinutesAgo).
		Updates(map[string]interface{}{
			"status":       "failed",
			"completed_at": time.Now(),
		}).Error
}

// CheckAndCleanupStaleProcesses kiểm tra và cleanup các process bị treo quá 10 phút
func (s *ProcessStatusService) CheckAndCleanupStaleProcesses(userID uint) error {
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)

	// Tìm và cập nhật các process bị treo của user này
	return config.Db.Model(&config.UserProcessStatus{}).
		Where("user_id = ? AND status = ? AND started_at < ?", userID, "processing", tenMinutesAgo).
		Updates(map[string]interface{}{
			"status":       "failed",
			"completed_at": time.Now(),
		}).Error
}

// GetUserActiveProcesses lấy danh sách process đang chạy của user
func (s *ProcessStatusService) GetUserActiveProcesses(userID uint) ([]config.UserProcessStatus, error) {
	// Trước tiên, cleanup các process bị treo quá 10 phút
	if err := s.CheckAndCleanupStaleProcesses(userID); err != nil {
		return nil, err
	}

	var processes []config.UserProcessStatus

	err := config.Db.Where("user_id = ? AND status = ?", userID, "processing").
		Order("started_at DESC").
		Find(&processes).Error

	return processes, err
}

// DeleteCaptionHistoryAndFiles xóa record CaptionHistory và tất cả file vật lý liên quan
func (s *ProcessStatusService) DeleteCaptionHistoryAndFiles(history *config.CaptionHistory) error {
	//deleteFile := func(path string) {
	//	if path != "" {
	//		_ = os.Remove(path)
	//	}
	//}
	//deleteFile(history.VideoFilename)
	//deleteFile(history.SrtFile)
	//deleteFile(history.OriginalSrtFile)
	//deleteFile(history.TTSFile)
	//deleteFile(history.MergedVideoFile)
	//deleteFile(history.BackgroundMusic)
	//// Có thể xóa thêm các file khác nếu cần
	//return config.Db.Delete(history).Error
	var videoDir string
	if history.VideoFilename != "" {
		videoDir = filepath.Dir(history.VideoFilename)
	} else if history.MergedVideoFile != "" {
		videoDir = filepath.Dir(history.MergedVideoFile)
	} else if history.SrtFile != "" {
		videoDir = filepath.Dir(history.SrtFile)
	}

	// Xóa toàn bộ thư mục
	if videoDir != "" && videoDir != "." && videoDir != "/" {
		_ = os.RemoveAll(videoDir)
	}

	return config.Db.Delete(history).Error
}

// CleanupOldCaptionHistories xóa các CaptionHistory quá 24h và file liên quan
func (s *ProcessStatusService) CleanupOldCaptionHistories() error {
	cutoff := time.Now().Add(-24 * time.Hour)
	var oldHistories []config.CaptionHistory
	if err := config.Db.Where("created_at < ?", cutoff).Find(&oldHistories).Error; err != nil {
		return err
	}
	for _, history := range oldHistories {
		s.DeleteCaptionHistoryAndFiles(&history)
	}
	return nil
}

// LimitUserCaptionHistories giữ tối đa 10 action gần nhất cho mỗi user
func (s *ProcessStatusService) LimitUserCaptionHistories(userID uint) error {
	var histories []config.CaptionHistory
	if err := config.Db.Where("user_id = ? AND deleted_at IS NULL", userID).Order("created_at desc").Find(&histories).Error; err != nil {
		return err
	}
	if len(histories) <= 10 {
		return nil
	}
	// Xóa các action cũ nhất (bắt đầu từ index 10)
	for i := 10; i < len(histories); i++ {
		s.DeleteCaptionHistoryAndFiles(&histories[i])
	}
	return nil
}
