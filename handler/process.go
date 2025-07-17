package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/limit"
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ProcessHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	log := logrus.WithFields(logrus.Fields{
		"ip": c.ClientIP(),
	})

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Kiểm tra giới hạn Free (5 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 3 video/ngày. Nâng cấp Pro!"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}

	// Lưu file tạm
	_, _, _, saveAudioPath, _ := util.Processfile(c, file)
	// Kiểm tra duration < 10 phút
	duration, _ := util.GetAudioDuration(saveAudioPath)
	if duration > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 10 phút."})
		return
	}

	// Gọi Whisper để lấy transcript và usage thực tế
	transcript, segments, whisperUsage, err := service.TranscribeWhisperOpenAI(saveAudioPath, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}
	// Trừ token đúng usage thực tế
	whisperTokens := 0
	if whisperUsage != nil && whisperUsage.TotalTokens > 0 {
		whisperTokens = whisperUsage.TotalTokens
	} else {
		duration, _ := util.GetAudioDuration(saveAudioPath)
		whisperTokens = int(duration/60.0*6 + 0.5)
		if whisperTokens < 6 {
			whisperTokens = 6
		}
	}
	// Lưu history trước để lấy video_id
	jsonData, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       file.Filename,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            jsonData,
		ProcessType:         "process",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}

	//Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(transcript, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	// Trừ token cho Gemini dịch (tính theo ký tự gửi lên)
	jsonData, _ = json.Marshal(segments)
	geminiTokens := int(float64(len(jsonData))/62.5 + 0.9999)
	if geminiTokens < 1 {
		geminiTokens = 1
	}

	// Update history
	captionHistory.Suggestion = captionsAndHashtag
	config.Db.Save(&captionHistory)
	c.JSON(http.StatusOK, gin.H{
		"transcript": transcript,
		"suggestion": captionsAndHashtag,
		"segments":   segments,
		"id":         captionHistory.ID,
	})
}

func ProcessVideoHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey
	log := logrus.WithFields(logrus.Fields{
		"ip": c.ClientIP(),
	})

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy process_id từ middleware
	processID := c.GetUint("process_id")
	processService := service.NewProcessStatusService()
	creditService := service.NewCreditService()

	// Đảm bảo cập nhật trạng thái process khi hoàn thành hoặc lỗi
	defer func() {
		if processID > 0 {
			if r := recover(); r != nil {
				// Có panic, cập nhật trạng thái failed
				processService.UpdateProcessStatus(processID, "failed")
				panic(r) // Re-panic để gin có thể xử lý
			}
		}
	}()

	// Kiểm tra giới hạn Free (3 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 3 video/ngày. Nâng cấp Pro!"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("Error getting video file: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No video file provided",
		})
		return
	}

	log.Printf("Received video file: %s, size: %d bytes", file.Filename, file.Size)

	// Tạo tên file duy nhất
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))

	// Tạo thư mục riêng cho video
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		log.Printf("Error creating video directory: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create video directory",
		})
		return
	}

	// Lưu video vào thư mục riêng
	videoPath := filepath.Join(videoDir, uniqueName)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		log.Printf("Error saving video file: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save video file",
		})
		return
	}

	log.Printf("Video file saved to: %s", videoPath)

	// Tách audio từ video
	audioPath, err := util.ProcessfileToDir(c, file, videoDir)
	if err != nil {
		log.Printf("Error processing file: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}

	// Kiểm tra duration < 10 phút
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 600 {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 10 phút."})
		return
	}

	// Gọi Whisper để lấy transcript và usage thực tế
	transcript, segments, _, err := service.TranscribeWhisperOpenAI(audioPath, apiKey)
	if err != nil {
		log.Printf("Error transcribing vocals: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transcribe vocals: %v", err)})
		return
	}

	// Tính chi phí Whisper theo thời gian audio thực tế
	durationMinutes := duration / 60.0

	pricingService := service.NewPricingService()
	whisperCost, err := pricingService.CalculateWhisperCost(durationMinutes)
	if err != nil {
		log.Printf("Error calculating Whisper cost: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate cost"})
		return
	}

	log.Printf("Whisper cost: $%.6f for %.2f minutes", whisperCost, durationMinutes)

	// Ước tính tổng chi phí với markup và lock credit
	estimatedCostWithMarkup, err := pricingService.EstimateProcessVideoCostWithMarkup(durationMinutes, len(transcript), len("estimated_text"), userID)
	if err != nil {
		log.Printf("Error estimating cost with markup: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to estimate cost"})
		return
	}

	estimatedCost := estimatedCostWithMarkup["total"]

	// Lock credit trước khi xử lý
	_, err = creditService.LockCredits(userID, estimatedCost, "process-video", "Lock credit for video processing", nil)
	if err != nil {
		log.Printf("Error locking credits: %v", err)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit để xử lý video"})
		return
	}

	// Đảm bảo unlock credit nếu có lỗi
	defer func() {
		if r := recover(); r != nil {
			creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to panic", nil)
			panic(r)
		}
	}()

	// Save to database trước để lấy video_id
	segmentsJSON, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       uniqueName,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            segmentsJSON,
		ProcessType:         "process-video",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("Error saving to database: %v", err)
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to database error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save to database"})
		return
	}

	// Trừ credit cho Whisper theo chi phí chính xác
	if err := creditService.DeductCredits(userID, whisperCost, "whisper", "Whisper transcribe", &captionHistory.ID, "per_minute", durationMinutes); err != nil {
		log.Printf("[BUG] DeductCredits whisper: userID=%d", userID)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to Whisper error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho Whisper"})
		return
	}

	// Create original SRT file from Whisper segments first
	originalSRTPath := filepath.Join(videoDir, strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_original.srt")
	originalSRTContent := createSRT(segments)
	if err := os.WriteFile(originalSRTPath, []byte(originalSRTContent), 0644); err != nil {
		log.Printf("Error creating original SRT file: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to SRT error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create original SRT file"})
		return
	}

	// Lấy service_name và model_api_name cho nghiệp vụ dịch SRT từ bảng service_config
	serviceName, srtModelAPIName, err := pricingService.GetActiveServiceForType("srt_translation")
	if err != nil {
		log.Printf("Error getting active SRT translation service: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to service config error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active SRT translation service"})
		return
	}

	// Translate the original SRT file using Gemini
	translatedSRTContent, err := service.TranslateSRTFileWithModel(originalSRTPath, geminiKey, srtModelAPIName)
	if err != nil {
		log.Printf("Error translating SRT file: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to translation error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to translate SRT: %v", err)})
		return
	}

	// Save translated SRT file
	translatedSRTPath := filepath.Join(videoDir, strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_vi.srt")
	if err := os.WriteFile(translatedSRTPath, []byte(translatedSRTContent), 0644); err != nil {
		log.Printf("Error saving translated SRT file: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to SRT save error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save translated SRT file"})
		return
	}

	// Tính chi phí Gemini theo số ký tự thực tế - sử dụng serviceName, không phải model_api_name
	geminiCost, geminiTokens, _, err := pricingService.CalculateGeminiCost(originalSRTContent, serviceName)
	if err != nil {
		log.Printf("Error calculating Gemini cost: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to Gemini cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate Gemini cost"})
		return
	}

	log.Printf("Gemini cost: $%.6f for %d tokens", geminiCost, geminiTokens)

	if err := creditService.DeductCredits(userID, geminiCost, serviceName, "Gemini dịch SRT", &captionHistory.ID, "per_token", float64(geminiTokens)); err != nil {
		log.Printf("[BUG] DeductCredits gemini: userID=%d", userID)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost, "process-video", "Unlock remaining credits due to Gemini deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho Gemini"})
		return
	}

	// Use original segments for database storage (no need to parse SRT back to segments)
	segmentsViJSON, _ := json.Marshal(segments)
	// Lấy các tham số tuỳ chỉnh từ form-data
	backgroundVolume := 1.2
	ttsVolume := 1.5 // Tăng default TTS volume để voice rõ ràng hơn
	speakingRate := 1.2

	// Log raw form values
	log.Printf("Raw form values - background_volume: %s, tts_volume: %s, speaking_rate: %s",
		c.PostForm("background_volume"), c.PostForm("tts_volume"), c.PostForm("speaking_rate"))

	if v := c.PostForm("background_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			backgroundVolume = f
			log.Printf("Parsed background_volume: %.2f", backgroundVolume)
		} else {
			log.Printf("Failed to parse background_volume: %s, error: %v", v, err)
		}
	}
	if v := c.PostForm("tts_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ttsVolume = f
			log.Printf("Parsed tts_volume: %.2f", ttsVolume)
		} else {
			log.Printf("Failed to parse tts_volume: %s, error: %v", v, err)
		}
	}
	if v := c.PostForm("speaking_rate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			speakingRate = f
			log.Printf("Parsed speaking_rate: %.2f", speakingRate)
		} else {
			log.Printf("Failed to parse speaking_rate: %s, error: %v", v, err)
		}
	}

	log.Printf("Final volume settings - background: %.2f, tts: %.2f, speaking_rate: %.2f",
		backgroundVolume, ttsVolume, speakingRate)
	// Read translated SRT content for TTS
	srtContentBytes, _ := os.ReadFile(translatedSRTPath)

	// Tính chi phí TTS theo số ký tự thực tế (sử dụng Wavenet cho chất lượng tốt)
	ttsCost, err := pricingService.CalculateTTSCost(string(srtContentBytes), true)
	if err != nil {
		log.Printf("Error calculating TTS cost: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost, "process-video", "Unlock remaining credits due to TTS cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate TTS cost"})
		return
	}

	log.Printf("TTS cost: $%.6f for %d characters", ttsCost, len(srtContentBytes))

	if err := creditService.DeductCredits(userID, ttsCost, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len(srtContentBytes))); err != nil {
		log.Printf("[BUG] DeductCredits tts: userID=%d", userID)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost-ttsCost, "process-video", "Unlock remaining credits due to TTS deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho TTS"})
		return
	}

	// Convert translated SRT to speech
	ttsPath, err := service.ConvertSRTToSpeech(string(srtContentBytes), videoDir, speakingRate)
	if err != nil {
		log.Printf("Error converting SRT to speech: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost-ttsCost, "process-video", "Unlock remaining credits due to TTS conversion error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to convert SRT to speech: %v", err)})
		return
	}

	// Merge video with background music and TTS audio
	backgroundPath, err := service.ExtractBackgroundMusicAsync(audioPath, uniqueName, videoDir)
	if err != nil {
		log.Printf("Error extracting background music: %v", err)
		log.Printf("Trying FFmpeg fallback...")
		backgroundPath, err = service.FallbackSeparateAudio(audioPath, uniqueName, "no_vocals", videoDir)
		if err != nil {
			log.Printf("FFmpeg fallback also failed: %v", err)
			// Sử dụng audio gốc nếu tách thất bại
			backgroundPath = audioPath
		}
	}

	mergedVideoPath, err := service.MergeVideoWithAudio(videoPath, backgroundPath, ttsPath, videoDir, backgroundVolume, ttsVolume)
	if err != nil {
		log.Printf("Error merging video: %v", err)
		mergedVideoPath = ""
	}

	// Update history
	captionHistory.SegmentsVi = segmentsViJSON
	captionHistory.SrtFile = translatedSRTPath
	captionHistory.OriginalSrtFile = originalSRTPath
	captionHistory.TTSFile = ttsPath
	captionHistory.MergedVideoFile = mergedVideoPath
	captionHistory.BackgroundMusic = backgroundPath
	config.Db.Save(&captionHistory)

	// Cập nhật trạng thái process thành completed và video_id
	if processID > 0 {
		processService.UpdateProcessStatus(processID, "completed")
		processService.UpdateProcessVideoID(processID, captionHistory.ID)
	}

	log.Printf("Total cost: $%.6f", estimatedCost)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Video processed successfully",
		"background_music": backgroundPath,
		"srt_file":         translatedSRTPath,
		"tts_file":         ttsPath,
		"merged_video":     mergedVideoPath,
		"transcript":       transcript,
		"segments":         segments,
		"segments_vi":      segments,
		"id":               captionHistory.ID,
	})
}

// EstimateProcessVideoCostHandler ước tính chi phí cho process-video
func EstimateProcessVideoCostHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		DurationMinutes  float64 `json:"duration_minutes" binding:"required"`
		TranscriptLength int     `json:"transcript_length"`
		SrtLength        int     `json:"srt_length"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Ước tính transcript và SRT length nếu không được cung cấp
	if req.TranscriptLength == 0 {
		// Ước tính: 150 từ/phút, mỗi từ 5 ký tự
		req.TranscriptLength = int(req.DurationMinutes * 150 * 5)
	}
	if req.SrtLength == 0 {
		// Ước tính: SRT dài hơn transcript 20% do format
		req.SrtLength = int(float64(req.TranscriptLength) * 1.2)
	}

	pricingService := service.NewPricingService()
	estimates, err := pricingService.EstimateProcessVideoCostWithMarkup(req.DurationMinutes, req.TranscriptLength, req.SrtLength, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to estimate cost"})
		return
	}

	// Lấy credit balance của user
	creditService := service.NewCreditService()
	var creditBalance float64
	creditBalanceMap, err := creditService.GetUserCreditBalance(userID)
	if err != nil {
		creditBalance = 0
	} else {
		creditBalance = creditBalanceMap["available_credits"]
	}

	// Lấy thông tin tier của user
	userTier, err := pricingService.GetUserTier(userID)
	tierName := "free"
	if err == nil {
		tierName = userTier.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"estimates":           estimates,
		"user_credit_balance": creditBalance,
		"sufficient_credits":  creditBalance >= estimates["total"],
		"currency":            "USD",
		"user_tier":           tierName,
		"markup_info": gin.H{
			"markup_amount":     estimates["markup_amount"],
			"markup_percentage": estimates["markup_percentage"],
			"base_cost":         estimates["total_base"],
			"final_cost":        estimates["total"],
		},
	})
}

// Helper function to create SRT content from segments
func createSRT(segments []service.Segment) string {
	var srtBuilder strings.Builder
	for i, segment := range segments {
		// SRT format: index, start --> end, text
		srtBuilder.WriteString(fmt.Sprintf("%d\n", i+1))
		srtBuilder.WriteString(fmt.Sprintf("%s --> %s\n",
			formatTime(segment.Start),
			formatTime(segment.End)))
		srtBuilder.WriteString(segment.Text + "\n\n")
	}
	return srtBuilder.String()
}

// Helper function to format time in SRT format (HH:MM:SS,mmm)
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// GenerateCaptionHandler tạo caption mới từ transcript
func GenerateCaptionHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Transcript string `json:"transcript" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(req.Transcript, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestion": captionsAndHashtag,
	})
}

// TikTokOptimizerHandler phân tích và tối ưu video cho TikTok
func TikTokOptimizerHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	log := logrus.WithFields(logrus.Fields{
		"ip": c.ClientIP(),
	})

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Kiểm tra giới hạn Free (5 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 3 video/ngày. Nâng cấp Pro!"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}

	// Lấy thông tin bổ sung
	currentCaption := c.PostForm("current_caption")
	targetAudience := c.PostForm("target_audience")
	if targetAudience == "" {
		targetAudience = "general"
	}

	// Lưu file tạm
	_, _, _, saveAudioPath, _ := util.Processfile(c, file)

	// Kiểm tra duration < 10 phút
	duration, _ := util.GetAudioDuration(saveAudioPath)
	if duration > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 10 phút."})
		return
	}

	// Gọi Whisper để lấy transcript
	transcript, segments, _, err := service.TranscribeWhisperOpenAI(saveAudioPath, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}

	// Tạo prompt cho TikTok optimization
	prompt := fmt.Sprintf(`Phân tích video TikTok và đưa ra gợi ý tối ưu:

Video transcript: %s
Thời lượng: %.1f giây
Caption hiện tại: %s
Target audience: %s

Hãy phân tích và đưa ra:

1. HOOK SCORE (0-100): Đánh giá độ mạnh của hook trong 3 giây đầu
2. OPTIMIZATION TIPS: 3-5 tips để tối ưu video
3. TRENDING HASHTAGS: 10 hashtags trending phù hợp
4. SUGGESTED CAPTION: Caption tối ưu cho TikTok
5. BEST POSTING TIME: Thời gian đăng tốt nhất
6. VIRAL POTENTIAL: Điểm viral tiềm năng (0-100)
7. ENGAGEMENT PROMPTS: 3 câu hỏi để tăng engagement
8. CALL TO ACTION: Gợi ý CTA hiệu quả

Trả về dưới dạng JSON:
{
  "hook_score": 85,
  "optimization_tips": ["tip1", "tip2", "tip3"],
  "trending_hashtags": ["#hashtag1", "#hashtag2"],
  "suggested_caption": "Caption tối ưu...",
  "best_posting_time": "19:00-21:00",
  "viral_potential": 75,
  "engagement_prompts": ["Câu hỏi 1?", "Câu hỏi 2?"],
  "call_to_action": "Follow để xem thêm!"
}`, transcript, duration, currentCaption, targetAudience)

	// Gọi GPT để phân tích
	analysisRaw, err := service.GenerateSuggestion(prompt, apiKey)
	log.Printf("[TikTokOptimizer] GPT response: %v", analysisRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	var analysis interface{} = analysisRaw
	// Parse analysis thành map[string]interface{}
	var result map[string]interface{}
	if s, ok := analysis.(string); ok {
		if err := json.Unmarshal([]byte(s), &result); err != nil {
			log.Printf("[TikTokOptimizer] Failed to parse GPT JSON: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT response parse error"})
			return
		}
	} else if m, ok := analysis.(map[string]interface{}); ok {
		result = m
	} else {
		log.Printf("[TikTokOptimizer] Unexpected GPT response type: %T", analysis)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT response type error"})
		return
	}

	// Lưu history
	jsonData, _ := json.Marshal(segments)
	var suggestionStr string
	if s, ok := analysis.(string); ok {
		suggestionStr = s
	} else if m, ok := analysis.(map[string]interface{}); ok {
		b, _ := json.Marshal(m)
		suggestionStr = string(b)
	}
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       file.Filename,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            jsonData,
		Suggestion:          suggestionStr, // Lưu lại response dạng string
		ProcessType:         "tiktok-optimize",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}

	// Trả về đúng các trường từ GPT
	result["transcript"] = transcript
	result["segments"] = segments
	result["id"] = captionHistory.ID
	c.JSON(http.StatusOK, result)
}
