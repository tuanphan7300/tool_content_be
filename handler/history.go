package handler

import (
	"creator-tool-backend/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

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
	var histories []config.CaptionHistory
	query := config.Db.Where("user_id = ?", userID)
	if processType != "" {
		query = query.Where("process_type = ?", processType)
	}
	if err := query.Order("created_at desc").Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
		if err := config.Db.Where("user_id = ? AND process_type = ? AND video_id = ?", h.UserID, h.ProcessType, h.ID).Order("created_at desc").First(&processStatus).Error; err == nil {
			status = processStatus.Status
		}
		result = append(result, HistoryWithStatus{
			CaptionHistory: h,
			ProcessStatus:  status,
		})
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
			BackgroundMusic:     filepath.Base(history.BackgroundMusic),
			SrtFile:             filepath.Base(history.SrtFile),
			OriginalSrtFile:     filepath.Base(history.OriginalSrtFile),
			TTSFile:             filepath.Base(history.TTSFile),
			MergedVideoFile:     filepath.Base(history.MergedVideoFile),
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
	if err := config.Db.Where("id = ? AND user_id = ?", id, userID).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "History not found"})
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
	if err := config.Db.Delete(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "History and storage deleted"})
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
