package handler

import (
	"creator-tool-backend/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// normalizeStoragePath đảm bảo path bắt đầu bằng /storage/ để nginx direct serving
func normalizeStoragePath(path string) string {
	if path == "" {
		return ""
	}
	// Nếu path đã bắt đầu bằng /storage/, return nguyên
	if strings.HasPrefix(path, "/storage/") {
		return path
	}
	// Nếu path bắt đầu bằng storage/, thêm /
	if strings.HasPrefix(path, "storage/") {
		return "/" + path
	}
	// Nếu không có storage/, trả về path gốc (có thể là relative path khác)
	return path
}

type SaveHistoryRequest struct {
	VideoFilename string          `json:"video_filename"`
	Transcript    string          `json:"transcript"`
	Suggestion    string          `json:"suggestion"`
	Segments      json.RawMessage `json:"segments"`
	Timestamps    []string        `json:"timestamps"`
}

func SaveHistory(c *gin.Context) {
	var request SaveHistoryRequest

	// Kiểm tra dữ liệu đầu vào
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Chuyển timestamps thành JSON
	timestampsJSON, err := json.Marshal(request.Timestamps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal timestamps"})
		return
	}

	// Lưu vào database
	result := config.Db.Create(&config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       request.VideoFilename,
		VideoFilenameOrigin: request.VideoFilename,
		Transcript:          request.Transcript,
		Suggestion:          request.Suggestion,
		Timestamps:          datatypes.JSON(timestampsJSON),
		Segments:            datatypes.JSON(request.Segments),
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History saved"})
}

func GetHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	processType := c.Query("process_type") // Lấy process_type từ query param

	// Debug logs
	fmt.Printf("GetHistory: userID=%d, processType=%s\n", userID, processType)

	var histories []config.CaptionHistory
	query := config.Db.Where("user_id = ? AND deleted_at IS NULL", userID)
	if processType != "" {
		query = query.Where("process_type = ?", processType)
	}
	if err := query.Order("created_at desc").Find(&histories).Error; err != nil {
		fmt.Printf("GetHistory: Error querying database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Debug logs
	fmt.Printf("GetHistory: Found %d records for userID=%d, processType=%s\n", len(histories), userID, processType)
	for i, h := range histories {
		fmt.Printf("GetHistory: Record %d - ID=%d, ProcessType=%s, CreatedAt=%v\n", i+1, h.ID, h.ProcessType, h.CreatedAt)
	}

	// Lấy process_status cho từng bản ghi
	type HistoryWithStatus struct {
		config.CaptionHistory
		ProcessStatus string `json:"process_status"`
	}
	var result []HistoryWithStatus
	for _, h := range histories {
		var processStatus config.UserProcessStatus
		status := ""

		// Thử tìm process_status bằng video_id
		if err := config.Db.Where("user_id = ? AND process_type = ? AND video_id = ?", h.UserID, h.ProcessType, h.ID).Order("created_at desc").First(&processStatus).Error; err == nil {
			status = processStatus.Status
		} else {
			// Nếu không tìm thấy process_status, thử tìm bằng process_type và user_id (cho các process không có video_id)
			if err := config.Db.Where("user_id = ? AND process_type = ?", h.UserID, h.ProcessType).Order("created_at desc").First(&processStatus).Error; err == nil {
				status = processStatus.Status
			} else {
				// Fallback: kiểm tra merged_video_file để xác định trạng thái
				if h.MergedVideoFile != "" {
					status = "completed"
				} else if h.ProcessType == "tiktok-optimize" && h.SuggestedCaption != "" {
					// TikTok Optimizer hoàn thành khi có suggested_caption
					status = "completed"
				} else {
					status = "processing"
				}
			}
		}

		result = append(result, HistoryWithStatus{
			CaptionHistory: h,
			ProcessStatus:  status,
		})
	}

	// Debug: Log the final result
	fmt.Printf("GetHistory: Returning %d records in result\n", len(result))

	// Ensure we always return an array, even if empty
	if result == nil {
		result = []HistoryWithStatus{}
	}

	c.JSON(http.StatusOK, result)
}

func GetHistoryByID(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var history config.CaptionHistory

	if err := config.Db.Where("id = ? AND user_id = ?", id, userID).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
		return
	}
	c.JSON(http.StatusOK, history)
}

func GetHistoryHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get history from database
	var histories []config.CaptionHistory
	if err := config.Db.Where("user_id = ?", userID).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
		return
	}

	// Format response to include only filenames without paths
	type HistoryResponse struct {
		ID                  uint      `json:"id"`
		VideoFilename       string    `json:"video_filename"`
		VideoFilenameOrigin string    `json:"video_filename_origin"`
		Transcript          string    `json:"transcript"`
		Segments            string    `json:"segments"`
		SegmentsVi          string    `json:"segments_vi"`
		BackgroundMusic     string    `json:"background_music"`
		SrtFile             string    `json:"srt_file"`
		OriginalSrtFile     string    `json:"original_srt_file"`
		TTSFile             string    `json:"tts_file"`
		MergedVideoFile     string    `json:"merged_video_file"`
		CreatedAt           time.Time `json:"created_at"`
		// TikTok Optimizer fields
		HookScore         int    `json:"hook_score"`
		ViralPotential    int    `json:"viral_potential"`
		TrendingHashtags  string `json:"trending_hashtags"`
		SuggestedCaption  string `json:"suggested_caption"`
		BestPostingTime   string `json:"best_posting_time"`
		OptimizationTips  string `json:"optimization_tips"`
		EngagementPrompts string `json:"engagement_prompts"`
		CallToAction      string `json:"call_to_action"`
	}

	var response []HistoryResponse
	for _, history := range histories {
		response = append(response, HistoryResponse{
			ID:                  history.ID,
			VideoFilename:       history.VideoFilename,
			VideoFilenameOrigin: history.VideoFilenameOrigin,
			Transcript:          history.Transcript,
			Segments:            string(history.Segments),
			SegmentsVi:          string(history.SegmentsVi),
			BackgroundMusic:     history.BackgroundMusic, // Nginx direct serving
			SrtFile:             history.SrtFile,         // Nginx direct serving
			OriginalSrtFile:     history.OriginalSrtFile, // Nginx direct serving
			TTSFile:             history.TTSFile,         // Nginx direct serving
			MergedVideoFile:     history.MergedVideoFile, // Nginx direct serving
			CreatedAt:           history.CreatedAt,
			// TikTok Optimizer fields
			HookScore:         history.HookScore,
			ViralPotential:    history.ViralPotential,
			TrendingHashtags:  string(history.TrendingHashtags),
			SuggestedCaption:  history.SuggestedCaption,
			BestPostingTime:   history.BestPostingTime,
			OptimizationTips:  string(history.OptimizationTips),
			EngagementPrompts: string(history.EngagementPrompts),
			CallToAction:      history.CallToAction,
		})
	}

	c.JSON(http.StatusOK, response)
}

// Xoá 1 history và tài nguyên liên quan
func DeleteHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var history config.CaptionHistory
	if err := config.Db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
		return
	}

	// Soft delete - chỉ đánh dấu deleted_at, không xóa thực sự
	now := time.Now()
	if err := config.Db.Model(&history).Update("deleted_at", &now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete history"})
		return
	}

	// Xoá toàn bộ thư mục chứa video và tài nguyên liên quan
	var videoDir string
	if history.ProcessType == "tiktok-optimize" && history.VideoFilename != "" {
		videoDir = filepath.Dir(history.VideoFilename)
	} else if history.VideoFilename != "" {
		videoDir = filepath.Dir(history.VideoFilename)
	} else if history.MergedVideoFile != "" {
		videoDir = filepath.Dir(history.MergedVideoFile)
	} else if history.SrtFile != "" {
		videoDir = filepath.Dir(history.SrtFile)
	}
	if videoDir != "" && videoDir != "." && videoDir != "/" {
		_ = os.RemoveAll(videoDir)
	}

	c.JSON(http.StatusOK, gin.H{"message": "History deleted successfully"})
}

// Xoá nhiều history
func DeleteHistories(c *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ids"})
		return
	}
	for _, id := range req.Ids {
		c.Params = append(c.Params[:0], gin.Param{Key: "id", Value: fmt.Sprint(id)})
		DeleteHistory(c)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Batch delete completed"})
}

// GetUserVideoCount trả về tổng số video của user
func GetUserVideoCount(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var count int64
	if err := config.Db.Model(&config.CaptionHistory{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get video count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_videos": count,
		"max_videos":   10,
		"can_upload":   count < 10,
	})
}

// GetUserVideoStats trả về thống kê chi tiết về video của user (bao gồm cả đã xóa)
func GetUserVideoStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Thống kê tổng quan
	var totalCount, activeCount, deletedCount int64
	var processTypeStats []struct {
		ProcessType string `json:"process_type"`
		Count       int64  `json:"count"`
	}

	// Đếm tổng số video (bao gồm cả đã xóa)
	config.Db.Model(&config.CaptionHistory{}).Where("user_id = ?", userID).Count(&totalCount)

	// Đếm video đang hoạt động
	config.Db.Model(&config.CaptionHistory{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&activeCount)

	// Đếm video đã xóa
	config.Db.Model(&config.CaptionHistory{}).Where("user_id = ? AND deleted_at IS NOT NULL", userID).Count(&deletedCount)

	// Thống kê theo process type
	config.Db.Model(&config.CaptionHistory{}).
		Select("process_type, count(*) as count").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Group("process_type").
		Scan(&processTypeStats)

	c.JSON(http.StatusOK, gin.H{
		"total_videos":       totalCount,
		"active_videos":      activeCount,
		"deleted_videos":     deletedCount,
		"process_type_stats": processTypeStats,
		"deletion_rate":      float64(deletedCount) / float64(totalCount) * 100, // Tỷ lệ xóa (%)
	})
}
