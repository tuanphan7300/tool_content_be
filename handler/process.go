package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func ProcessHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}
	// Kiểm tra kích thước file không quá 100MB
	if file.Size > 100*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file tối đa 100MB."})
		return
	}

	// Tạo thư mục riêng cho video process
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video directory"})
		return
	}
	videoPath := filepath.Join(videoDir, uniqueName)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}
	// Tách audio từ video vào đúng thư mục
	audioPath, err := util.ProcessfileToDir(c, file, videoDir)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}
	// Kiểm tra duration < 7 phút
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 420 {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
		return
	}
	// Gọi Whisper để lấy transcript và usage thực tế
	transcript, segments, whisperUsage, err := service.TranscribeWhisperOpenAI(audioPath, apiKey)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}
	// Trừ token đúng usage thực tế
	whisperTokens := 0
	if whisperUsage != nil && whisperUsage.TotalTokens > 0 {
		whisperTokens = whisperUsage.TotalTokens
	} else {
		duration, _ := util.GetAudioDuration(audioPath)
		whisperTokens = int(duration/60.0*6 + 0.5)
		if whisperTokens < 6 {
			whisperTokens = 6
		}
	}
	// Lưu history trước để lấy video_id
	jsonData, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       videoPath,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            jsonData,
		ProcessType:         "process",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
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
	//Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(transcript, apiKey)
	if err != nil {
		util.CleanupDir(videoDir)
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
		"delete_at":  deleteAt,
		"warning":    warning,
	})
}

func ProcessVideoHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey

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

	// Get target language parameter (default to Vietnamese if not provided)
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi" // Default to Vietnamese
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No video file provided",
		})
		return
	}
	// Kiểm tra kích thước file không quá 100MB
	if file.Size > 100*1024*1024 {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file tối đa 100MB."})
		return
	}

	// Tạo thư mục riêng cho video
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
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
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save video file",
		})
		return
	}

	// Tách audio từ video
	audioPath, err := util.ProcessfileToDir(c, file, videoDir)
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	// Kiểm tra duration < 7 phút
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 420 {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
		return
	}

	// Check for custom SRT file
	customSrtFile, err := c.FormFile("custom_srt")
	var transcript string
	var segments []service.Segment
	var hasCustomSrt bool = false
	if err == nil && customSrtFile != nil {
		// User uploaded custom SRT
		hasCustomSrt = true
		customSrtPath := filepath.Join(videoDir, "custom.srt")
		if err := c.SaveUploadedFile(customSrtFile, customSrtPath); err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save custom SRT file"})
			return
		}
		// Parse SRT file to segments
		segments, transcript, err = util.ParseSRTFile(customSrtPath)
		if err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đọc file phụ đề .srt"})
			return
		}
	} else {
		// Không có custom SRT, dùng Whisper như cũ
		transcript, segments, _, err = service.TranscribeWhisperOpenAI(audioPath, apiKey)
		if err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transcribe vocals: %v", err)})
			return
		}
	}

	// Tính chi phí Whisper theo thời gian audio thực tế
	durationMinutes := duration / 60.0

	pricingService := service.NewPricingService()
	whisperCost, err := pricingService.CalculateWhisperCost(durationMinutes)
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate cost"})
		return
	}

	// Ước tính tổng chi phí với markup và lock credit
	estimatedCostWithMarkup, err := pricingService.EstimateProcessVideoCostWithMarkup(durationMinutes, len(transcript), len("estimated_text"), userID)
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to estimate cost"})
		return
	}

	estimatedCost := estimatedCostWithMarkup["total"]

	// Lock credit trước khi xử lý
	_, err = creditService.LockCredits(userID, estimatedCost, "process-video", "Lock credit for video processing", nil)
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit để xử lý video",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
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
		VideoFilename:       videoPath,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            segmentsJSON,
		ProcessType:         "process-video",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to database error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save to database"})
		return
	}

	// Trừ credit cho Whisper theo chi phí chính xác
	if err := creditService.DeductCredits(userID, whisperCost, "whisper", "Whisper transcribe", &captionHistory.ID, "per_minute", durationMinutes); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to Whisper error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho Whisper",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Create original SRT file from Whisper segments first
	originalSRTPath := filepath.Join(videoDir, strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_original.srt")
	originalSRTContent := createSRT(segments)
	if err := os.WriteFile(originalSRTPath, []byte(originalSRTContent), 0644); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to SRT error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create original SRT file"})
		return
	}

	var translatedSRTContent string
	var translatedSRTPath string
	var geminiCost float64 = 0
	var geminiTokens int = 0

	if !hasCustomSrt {
		// Lấy service_name và model_api_name cho nghiệp vụ dịch SRT từ bảng service_config
		serviceName, srtModelAPIName, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to service config error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active SRT translation service"})
			return
		}

		// Translate the original SRT file using Gemini to target language
		translatedSRTContent, err = service.TranslateSRTFileWithModelAndLanguage(originalSRTPath, geminiKey, srtModelAPIName, targetLanguage)
		if err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to translation error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to translate SRT: %v", err)})
			return
		}

		// Save translated SRT file
		translatedSRTPath = filepath.Join(videoDir, strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_vi.srt")
		if err := os.WriteFile(translatedSRTPath, []byte(translatedSRTContent), 0644); err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to SRT save error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save translated SRT file"})
			return
		}

		// Tính chi phí Gemini theo số ký tự thực tế - sử dụng serviceName, không phải model_api_name
		geminiCost, geminiTokens, _, err = pricingService.CalculateGeminiCost(originalSRTContent, serviceName)
		if err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to Gemini cost error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate Gemini cost"})
			return
		}

		if err := creditService.DeductCredits(userID, geminiCost, serviceName, "Gemini dịch SRT", &captionHistory.ID, "per_token", float64(geminiTokens)); err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost, "process-video", "Unlock remaining credits due to Gemini deduction error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Không đủ credit cho Gemini",
				"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
			})
			return
		}
	} else {
		// User uploaded custom SRT, use it directly
		translatedSRTPath = filepath.Join(videoDir, "custom.srt")
		originalSRTPath = translatedSRTPath // Custom SRT is both original and translated
		log.Printf("Using custom SRT file: %s", translatedSRTPath)
	}

	// Use original segments for database storage (no need to parse SRT back to segments)
	segmentsViJSON, _ := json.Marshal(segments)
	// Lấy các tham số tuỳ chỉnh từ form-data
	backgroundVolume := 1.2
	ttsVolume := 1.5 // Tăng default TTS volume để voice rõ ràng hơn
	speakingRate := 1.2

	// Log raw form values
	if v := c.PostForm("background_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			backgroundVolume = f
		}
	}
	if v := c.PostForm("tts_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ttsVolume = f
		}
	}
	if v := c.PostForm("speaking_rate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			speakingRate = f
		}
	}

	// Read translated SRT content for TTS
	log.Printf("Reading SRT file for TTS: %s", translatedSRTPath)
	srtContentBytes, err := os.ReadFile(translatedSRTPath)
	if err != nil {
		log.Printf("Failed to read SRT file: %v", err)
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost, "process-video", "Unlock remaining credits due to SRT read error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read SRT file: %v", err)})
		return
	}
	log.Printf("Successfully read SRT file, size: %d bytes", len(srtContentBytes))

	// Tính chi phí TTS theo số ký tự thực tế (sử dụng Wavenet cho chất lượng tốt)
	ttsCost, err := pricingService.CalculateTTSCost(string(srtContentBytes), true)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost, "process-video", "Unlock remaining credits due to TTS cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate TTS cost"})
		return
	}

	if err := creditService.DeductCredits(userID, ttsCost, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len(srtContentBytes))); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost-ttsCost, "process-video", "Unlock remaining credits due to TTS deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho TTS",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Convert translated SRT to speech with target language
	log.Printf("Starting TTS conversion with language: %s, speaking rate: %f", targetLanguage, speakingRate)
	ttsPath, err := service.ConvertSRTToSpeechWithLanguage(string(srtContentBytes), videoDir, speakingRate, targetLanguage)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-geminiCost-ttsCost, "process-video", "Unlock remaining credits due to TTS conversion error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to convert SRT to speech: %v", err)})
		return
	}

	// Merge video with background music and TTS audio
	backgroundPath, err := service.ExtractBackgroundMusicAsync(audioPath, uniqueName, videoDir)
	if err != nil {
		backgroundPath, err = service.FallbackSeparateAudio(audioPath, uniqueName, "no_vocals", videoDir)
		if err != nil {
			backgroundPath = audioPath
		}
	}

	mergedVideoPath, err := service.MergeVideoWithAudio(videoPath, backgroundPath, ttsPath, videoDir, backgroundVolume, ttsVolume)
	if err != nil {
		mergedVideoPath = ""
	}

	// Burn subtitle vào video với background đen
	finalVideoPath := mergedVideoPath
	if mergedVideoPath != "" && translatedSRTPath != "" {
		burnedVideoPath, err := service.BurnSubtitleWithBackground(mergedVideoPath, translatedSRTPath, videoDir)
		if err != nil {
			log.Printf("Failed to burn subtitle: %v", err)
			// Nếu burn subtitle thất bại, vẫn dùng video đã merge
			finalVideoPath = mergedVideoPath
		} else {
			finalVideoPath = burnedVideoPath
		}
	}

	// Update history
	captionHistory.SegmentsVi = segmentsViJSON
	captionHistory.SrtFile = translatedSRTPath
	captionHistory.OriginalSrtFile = originalSRTPath
	captionHistory.TTSFile = ttsPath
	captionHistory.MergedVideoFile = finalVideoPath
	captionHistory.BackgroundMusic = backgroundPath
	config.Db.Save(&captionHistory)

	// Cập nhật trạng thái process thành completed và video_id
	if processID > 0 {
		processService.UpdateProcessStatus(processID, "completed")
		processService.UpdateProcessVideoID(processID, captionHistory.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Video processed successfully",
		"background_music":  backgroundPath,
		"srt_file":          translatedSRTPath, // Trả về file phụ đề đã dịch (khớp với audio TTS)
		"original_srt_file": originalSRTPath,   // Thêm file phụ đề gốc nếu cần
		"tts_file":          ttsPath,
		"merged_video":      finalVideoPath,
		"transcript":        transcript,
		"segments":          segments,
		"segments_vi":       segments,
		"id":                captionHistory.ID,
		"process_id":        processID,
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

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	creditService := service.NewCreditService()
	pricingService := service.NewPricingService()

	// Kiểm tra trạng thái process TikTok Optimizer đang chạy
	var existingProcess config.UserProcessStatus
	err := config.Db.Where("user_id = ? AND process_type = ? AND status = ?", userID, "tiktok-optimize", "processing").First(&existingProcess).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Bạn đang có một quá trình TikTok Optimizer đang chạy. Vui lòng chờ hoàn thành trước khi upload mới."})
		return
	}

	// Tạo trạng thái process TikTok Optimizer (processing)
	processStatus := config.UserProcessStatus{
		UserID:      userID,
		Status:      "processing",
		ProcessType: "tiktok-optimize",
		StartedAt:   time.Now(),
	}
	if err := config.Db.Create(&processStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo trạng thái process"})
		return
	}
	defer func() {
		// Nếu có panic hoặc return sớm, cập nhật trạng thái failed
		if r := recover(); r != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			panic(r)
		}
	}()

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}

	// Get target language parameter (default to Vietnamese if not provided)
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi" // Default to Vietnamese
	}

	// Lấy thông tin bổ sung
	currentCaption := c.PostForm("current_caption")
	targetAudience := c.PostForm("target_audience")
	if targetAudience == "" {
		targetAudience = "general"
	}

	// Tạo thư mục riêng cho video TikTok Optimizer
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video directory"})
		return
	}
	videoPath := filepath.Join(videoDir, uniqueName)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}
	// Tách audio từ video vào đúng thư mục
	audioPath, err := util.ProcessfileToDir(c, file, videoDir)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}
	// Kiểm tra duration < 7 phút
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 420 {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
		return
	}

	// --- TÍNH PHÍ WHISPER ---
	durationMinutes := duration / 60.0
	whisperCost, err := pricingService.CalculateWhisperCost(durationMinutes)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate Whisper cost"})
		return
	}

	// --- TẠO PROMPT GPT ---
	transcript, segments, _, err := service.TranscribeWhisperOpenAI(audioPath, apiKey)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}
	prompt := fmt.Sprintf(`Phân tích video TikTok và đưa ra gợi ý tối ưu. TRẢ VỀ KẾT QUẢ BẰNG TIẾNG VIỆT:

Video transcript: %s
Thời lượng: %.1f giây
Caption hiện tại: %s
Target audience: %s

Hãy phân tích và đưa ra (TẤT CẢ BẰNG TIẾNG VIỆT):

1. HOOK SCORE (0-100): Đánh giá độ mạnh của hook trong 3 giây đầu
2. OPTIMIZATION TIPS: 3-5 tips để tối ưu video (bằng tiếng Việt)
3. TRENDING HASHTAGS: 10 hashtags trending phù hợp
4. SUGGESTED CAPTION: Caption tối ưu cho TikTok (bằng tiếng Việt)
5. BEST POSTING TIME: Thời gian đăng tốt nhất
6. VIRAL POTENTIAL: Điểm viral tiềm năng (0-100)
7. ENGAGEMENT PROMPTS: 3 câu hỏi để tăng engagement (bằng tiếng Việt)
8. CALL TO ACTION: Gợi ý CTA hiệu quả (bằng tiếng Việt)

LƯU Ý: Tất cả nội dung phải bằng tiếng Việt, chỉ có hashtags có thể có tiếng Anh.

QUAN TRỌNG: Chỉ trả về JSON thuần túy, không có text giải thích thêm.

{
  "hook_score": 85,
  "optimization_tips": ["Gợi ý 1 bằng tiếng Việt", "Gợi ý 2 bằng tiếng Việt", "Gợi ý 3 bằng tiếng Việt"],
  "trending_hashtags": ["#hashtag1", "#hashtag2"],
  "suggested_caption": "Caption tối ưu bằng tiếng Việt...",
  "best_posting_time": "19:00-21:00",
  "viral_potential": 75,
  "engagement_prompts": ["Câu hỏi 1 bằng tiếng Việt?", "Câu hỏi 2 bằng tiếng Việt?"],
  "call_to_action": "Follow để xem thêm!"
}`, transcript, duration, currentCaption, targetAudience)

	// --- TÍNH PHÍ GPT ---
	gptTokens := len([]rune(prompt)) / 4
	if gptTokens < 1 {
		gptTokens = 1
	}
	gptCost, _, err := pricingService.CalculateGPTCost(prompt)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate GPT cost"})
		return
	}

	// --- LOCK CREDIT TRƯỚC KHI XỬ LÝ ---
	totalCost := whisperCost + gptCost
	_, err = creditService.LockCredits(userID, totalCost, "tiktok-optimize", "Lock credit for TikTok Optimizer", nil)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit để tối ưu TikTok",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}
	// Đảm bảo unlock nếu có lỗi
	defer func() {
		if r := recover(); r != nil {
			creditService.UnlockCredits(userID, totalCost, "tiktok-optimize", "Unlock due to panic", nil)
			panic(r)
		}
	}()

	// --- GỌI GPT ĐỂ PHÂN TÍCH ---
	analysisRaw, err := service.GenerateTikTokOptimization(prompt, apiKey)
	if err != nil {
		creditService.UnlockCredits(userID, totalCost, "tiktok-optimize", "Unlock due to GPT error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}

	// --- TIẾP TỤC XỬ LÝ VÀ TRẢ KẾT QUẢ NHƯ CŨ ---
	var analysis interface{} = analysisRaw
	var result map[string]interface{}
	if s, ok := analysis.(string); ok {
		// Tìm JSON trong response (có thể có text thêm)
		jsonStart := strings.Index(s, "{")
		jsonEnd := strings.LastIndex(s, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			jsonStr := s[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				util.CleanupDir(videoDir)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT response parse error"})
				return
			}
		} else {
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No JSON found in GPT response"})
			return
		}
	} else if m, ok := analysis.(map[string]interface{}); ok {
		result = m
	} else {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT response type error"})
		return
	}

	// Lưu history
	jsonData, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       videoPath,
		VideoFilenameOrigin: file.Filename,
		Transcript:          transcript,
		Segments:            jsonData,
		ProcessType:         "tiktok-optimize",
		CreatedAt:           time.Now(),
	}
	// Gán các trường TikTok Optimizer nếu có
	if v, ok := result["hook_score"].(float64); ok {
		captionHistory.HookScore = int(v)
	}
	if v, ok := result["viral_potential"].(float64); ok {
		captionHistory.ViralPotential = int(v)
	}
	if v, ok := result["trending_hashtags"].([]interface{}); ok {
		b, _ := json.Marshal(v)
		captionHistory.TrendingHashtags = datatypes.JSON(b)
	}
	if v, ok := result["suggested_caption"].(string); ok {
		captionHistory.SuggestedCaption = v
	}
	if v, ok := result["best_posting_time"].(string); ok {
		captionHistory.BestPostingTime = v
	}
	if v, ok := result["optimization_tips"].([]interface{}); ok {
		b, _ := json.Marshal(v)
		captionHistory.OptimizationTips = datatypes.JSON(b)
	}
	if v, ok := result["engagement_prompts"].([]interface{}); ok {
		b, _ := json.Marshal(v)
		captionHistory.EngagementPrompts = datatypes.JSON(b)
	}
	if v, ok := result["call_to_action"].(string); ok {
		captionHistory.CallToAction = v
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}

	// --- TRỪ CREDIT SAU KHI TẠO HISTORY (để có video_id) ---
	err = creditService.DeductCredits(userID, whisperCost, "whisper", "Whisper transcribe (TikTok Optimizer)", &captionHistory.ID, "per_minute", durationMinutes)
	if err != nil {
		creditService.UnlockCredits(userID, gptCost, "tiktok-optimize", "Unlock GPT credit after Whisper deduction error", &captionHistory.ID)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho Whisper",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}
	err = creditService.DeductCredits(userID, gptCost, "gpt_3.5_turbo", "GPT TikTok Optimization", &captionHistory.ID, "per_token", float64(gptTokens))
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho GPT",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Trả về đúng các trường từ GPT
	result["transcript"] = transcript
	result["segments"] = segments
	result["id"] = captionHistory.ID
	result["process_id"] = processStatus.ID
	config.Db.Model(processStatus).Updates(map[string]interface{}{
		"status":      "completed",
		"CompletedAt": time.Now(),
	})
	c.JSON(http.StatusOK, result)
}

// CreateSubtitleHandler tạo file SRT từ video
func CreateSubtitleHandler(c *gin.Context) {
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	creditService := service.NewCreditService()
	pricingService := service.NewPricingService()

	// Lấy process status từ middleware
	processStatusInterface, exists := c.Get("process_status")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy trạng thái process"})
		return
	}
	processStatus := processStatusInterface.(*config.UserProcessStatus)
	processID := processStatus.ID
	defer func() {
		// Nếu có panic hoặc return sớm, cập nhật trạng thái failed
		if r := recover(); r != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			panic(r)
		}
	}()

	// Lấy file video
	videoFile, err := c.FormFile("file")
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file video"})
		return
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mp4") &&
		!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".avi") &&
		!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mov") &&
		!strings.HasSuffix(strings.ToLower(videoFile.Filename), ".mkv") {
		config.Db.Model(processStatus).Update("status", "failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file video (mp4, avi, mov, mkv)"})
		return
	}

	// Lấy các tham số
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi" // Default to Vietnamese
	}

	isBilingual := c.PostForm("is_bilingual") == "true"

	// Tạo thư mục làm việc
	uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), strings.TrimSuffix(videoFile.Filename, filepath.Ext(videoFile.Filename)))
	videoDir := filepath.Join("storage", uniqueName)
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo thư mục làm việc"})
		return
	}
	// Không cleanup ngay, chỉ cleanup khi có lỗi

	// Lưu file video
	videoPath := filepath.Join(videoDir, videoFile.Filename)
	if err := c.SaveUploadedFile(videoFile, videoPath); err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file video"})
		return
	}

	// Trích xuất audio từ video
	audioPath, err := service.ProcessVideoToAudio(videoPath, videoDir)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể trích xuất audio: %v", err)})
		return
	}

	// Tính toán chi phí Whisper
	whisperCost, err := pricingService.CalculateWhisperCost(1.0) // Ước tính 1 phút
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí Whisper"})
		return
	}

	// Tính toán chi phí Gemini (nếu song ngữ)
	var geminiCost float64 = 0
	if isBilingual {
		// Ước tính chi phí Gemini dựa trên độ dài video
		serviceName, _, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ Gemini"})
			return
		}
		geminiCost, _, _, err = pricingService.CalculateGeminiCost("sample text", serviceName)
		if err != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí Gemini"})
			return
		}
	}

	totalCost := whisperCost + geminiCost

	// Kiểm tra credit
	creditBalanceMap, err := creditService.GetUserCreditBalance(userID)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra số dư"})
		return
	}
	creditBalance := creditBalanceMap["available_credits"]

	if creditBalance < totalCost {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Lock credit
	_, err = creditService.LockCredits(userID, totalCost, "create-subtitle", "Lock credit for create subtitle", nil)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa credit"})
		return
	}

	// Transcribe với Whisper
	transcript, segments, _, err := service.TranscribeWhisperOpenAI(audioPath, apiKey)
	if err != nil {
		creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to transcription error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể transcribe: %v", err)})
		return
	}

	// Tạo file SRT gốc
	originalSRTContent := createSRT(segments)
	originalSRTPath := filepath.Join(videoDir, uniqueName+"_original.srt")
	if err := os.WriteFile(originalSRTPath, []byte(originalSRTContent), 0644); err != nil {
		creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to SRT save error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file SRT gốc"})
		return
	}

	// Tạo caption history
	segmentsJSON, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       videoFile.Filename,
		VideoFilenameOrigin: videoFile.Filename,
		Transcript:          transcript,
		Segments:            datatypes.JSON(segmentsJSON),
		SrtFile:             originalSRTPath, // Sẽ được update nếu có dịch
		OriginalSrtFile:     originalSRTPath, // Luôn là SRT gốc
		ProcessType:         "create-subtitle",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Nếu song ngữ, dịch SRT
	var translatedSRTPath string
	if isBilingual {
		// Lấy service config cho Gemini
		_, srtModelAPIName, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to service config error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ Gemini"})
			return
		}

		// Dịch SRT
		translatedSRTContent, err := service.TranslateSRTFileWithModelAndLanguage(originalSRTPath, geminiKey, srtModelAPIName, targetLanguage)
		if err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to translation error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể dịch SRT: %v", err)})
			return
		}

		// Lưu file SRT đã dịch
		translatedSRTPath = filepath.Join(videoDir, uniqueName+"_"+targetLanguage+".srt")
		if err := os.WriteFile(translatedSRTPath, []byte(translatedSRTContent), 0644); err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to translated SRT save error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file SRT đã dịch"})
			return
		}

		// Parse segments đã dịch
		translatedSegments, _, err := util.ParseSRTFile(translatedSRTPath)
		if err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to translated SRT parse error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể parse file SRT đã dịch"})
			return
		}

		translatedSegmentsJSON, _ := json.Marshal(translatedSegments)
		captionHistory.SegmentsVi = datatypes.JSON(translatedSegmentsJSON)
		captionHistory.SrtFile = translatedSRTPath // SRT đã dịch
		// OriginalSrtFile đã được set là originalSRTPath ở trên
	}

	// Lưu caption history
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to database error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu lịch sử"})
		return
	}

	// Trừ credit cho Whisper
	if err := creditService.DeductCredits(userID, whisperCost, "whisper", "Whisper transcribe", &captionHistory.ID, "per_minute", 1.0); err != nil {
		creditService.UnlockCredits(userID, totalCost-whisperCost, "create-subtitle", "Unlock remaining credits due to Whisper deduction error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho Whisper",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Trừ credit cho Gemini (nếu song ngữ)
	if isBilingual {
		serviceName, _, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			creditService.UnlockCredits(userID, totalCost-whisperCost, "create-subtitle", "Unlock remaining credits due to Gemini service config error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ Gemini"})
			return
		}

		if err := creditService.DeductCredits(userID, geminiCost, serviceName, "Gemini dịch SRT", &captionHistory.ID, "per_token", 1.0); err != nil {
			creditService.UnlockCredits(userID, totalCost-whisperCost-geminiCost, "create-subtitle", "Unlock remaining credits due to Gemini deduction error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Không đủ credit cho Gemini",
				"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
			})
			return
		}
	}

	// Cập nhật trạng thái process thành completed
	if processID > 0 {
		psService := service.NewProcessStatusService()
		psService.UpdateProcessStatus(processID, "completed")
		psService.UpdateProcessVideoID(processID, captionHistory.ID)
	}

	// Trả về kết quả
	response := gin.H{
		"message":      "Tạo phụ đề thành công",
		"original_srt": originalSRTPath,
		"transcript":   transcript,
		"segments":     segments,
		"id":           captionHistory.ID,
		"process_id":   processID,
	}

	if isBilingual {
		response["translated_srt"] = translatedSRTPath
		response["target_language"] = targetLanguage
	}

	c.JSON(http.StatusOK, response)

	// Cleanup files sau 1 giờ để user có thời gian tải về
	go func() {
		time.Sleep(1 * time.Hour)
		util.CleanupDir(videoDir)
	}()
}
