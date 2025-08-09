package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
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
	// Lấy file trước để kiểm tra sớm
	file, err := c.FormFile("file")
	if err != nil {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}

	// Kiểm tra kích thước file không quá 100MB
	if file.Size > 100*1024*1024 {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileTooLarge, nil)
		return
	}

	// Tạo thư mục tạm để kiểm tra duration
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))
	tempDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		util.HandleError(c, http.StatusInternalServerError, util.ErrDirectoryCreation, err)
		return
	}

	// Lưu file tạm để kiểm tra duration
	tempVideoPath := filepath.Join(tempDir, uniqueName)
	if err := c.SaveUploadedFile(file, tempVideoPath); err != nil {
		util.CleanupDir(tempDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrFileUploadFailed, err)
		return
	}

	// Tách audio tạm để kiểm tra duration
	tempAudioPath, err := util.ProcessfileToDir(c, file, tempDir)
	if err != nil {
		util.CleanupDir(tempDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrProcessingFailed, err)
		return
	}

	// Kiểm tra duration < 7 phút
	duration, _ := util.GetAudioDuration(tempAudioPath)
	if duration > 420 {
		util.CleanupDir(tempDir)
		util.HandleError(c, http.StatusBadRequest, util.ErrFileDurationTooLong, nil)
		return
	}

	// Nếu pass tất cả kiểm tra, tiếp tục xử lý
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey

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

	// Sử dụng thư mục tạm đã tạo
	videoDir := tempDir
	videoPath := tempVideoPath
	audioPath := tempAudioPath

	// Khởi tạo services
	pricingService := service.NewPricingService()

	// Lấy service config cho speech_to_text (Whisper)
	whisperServiceName, whisperModelAPIName, err := pricingService.GetActiveServiceForType("speech_to_text")
	if err != nil {
		util.CleanupDir(videoDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrServiceUnavailable, err)
		return
	}

	// Lấy service config cho caption_generation (GPT)
	gptServiceName, gptModelAPIName, err := pricingService.GetActiveServiceForType("caption_generation")
	if err != nil {
		util.CleanupDir(videoDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrServiceUnavailable, err)
		return
	}

	// Gọi Whisper để lấy transcript
	transcript, segments, _, err := service.TranscribeWithService(audioPath, apiKey, whisperServiceName, whisperModelAPIName)
	if err != nil {
		util.CleanupDir(videoDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrProcessingFailed, err)
		return
	}

	// Token Whisper được dùng nội bộ cho phân tích, không còn cần lưu; bỏ tính để tránh cảnh báo linter

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
		util.HandleError(c, http.StatusInternalServerError, util.ErrDatabaseOperation, err)
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

	// Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateCaptionWithService(transcript, apiKey, gptServiceName, gptModelAPIName, targetLanguage)
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
	pricingService := service.NewPricingService()

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

	// Get subtitle color parameters
	subtitleColor := c.PostForm("subtitle_color")
	if subtitleColor == "" {
		subtitleColor = "#FFFFFF" // Default to white (same as burn-sub)
	}
	subtitleBgColor := c.PostForm("subtitle_bgcolor")
	if subtitleBgColor == "" {
		subtitleBgColor = "#808080" // Default to gray (same as burn-sub)
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}
	// Kiểm tra kích thước file không quá 100MB
	if file.Size > 100*1024*1024 {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.HandleError(c, http.StatusBadRequest, util.ErrFileTooLarge, nil)
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
		parsedSegments, parsedTranscript, err := util.ParseSRTFile(customSrtPath)
		if err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đọc file phụ đề .srt"})
			return
		}
		segments = parsedSegments
		transcript = parsedTranscript
	} else {
		// Không có custom SRT, dùng Whisper thông qua service_config
		whisperServiceName, whisperModelAPIName, err := pricingService.GetActiveServiceForType("speech_to_text")
		if err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active speech-to-text service"})
			return
		}

		transcript, segments, _, err = service.TranscribeWithService(audioPath, apiKey, whisperServiceName, whisperModelAPIName)
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
	var translationCost float64 = 0

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

		// Translate the original SRT file using the configured service (Gemini or GPT)
		if strings.Contains(serviceName, "gpt") {
			// Use GPT for translation
			translatedSRTContent, err = service.TranslateSRTFileWithGPT(originalSRTPath, apiKey, srtModelAPIName, targetLanguage)
		} else {
			// Use Gemini for translation (default)
			translatedSRTContent, err = service.TranslateSRTFileWithModelAndLanguage(originalSRTPath, geminiKey, srtModelAPIName, targetLanguage)
		}
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

		// Tính chi phí translation theo service được chọn (tách input/output nếu có)
		var translationTokens int
		var serviceDescription string

		if strings.Contains(serviceName, "gpt") {
			// GPT: cố gắng tính input + output riêng, fallback nếu thiếu
			inCost, outCost, inTok, outTok, _, splitErr := pricingService.CalculateLLMCostSplit(originalSRTContent, translatedSRTContent, serviceName)
			if splitErr == nil {
				translationCost = inCost + outCost
				translationTokens = inTok + outTok
				serviceDescription = "GPT dịch SRT (input+output)"
			} else {
				translationCost, translationTokens, err = pricingService.CalculateGPTCost(originalSRTContent)
				serviceDescription = "GPT dịch SRT (input only)"
			}
		} else {
			// Gemini: cố gắng tính input + output riêng, fallback nếu thiếu
			inCost, outCost, inTok, outTok, _, splitErr := pricingService.CalculateLLMCostSplit(originalSRTContent, translatedSRTContent, serviceName)
			if splitErr == nil {
				translationCost = inCost + outCost
				translationTokens = inTok + outTok
				serviceDescription = "Gemini dịch SRT (input+output)"
			} else {
				translationCost, translationTokens, _, err = pricingService.CalculateLLMCost(originalSRTContent, serviceName)
				serviceDescription = "Gemini dịch SRT (input only)"
			}
		}

		if err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost, "process-video", "Unlock remaining credits due to translation cost error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate translation cost"})
			return
		}

		if err := creditService.DeductCredits(userID, translationCost, serviceName, serviceDescription, &captionHistory.ID, "per_token", float64(translationTokens)); err != nil {
			creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost, "process-video", "Unlock remaining credits due to translation deduction error", nil)
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Không đủ credit cho translation",
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
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost, "process-video", "Unlock remaining credits due to SRT read error", nil)
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
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost, "process-video", "Unlock remaining credits due to TTS cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate TTS cost"})
		return
	}

	if err := creditService.DeductCredits(userID, ttsCost, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len(srtContentBytes))); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost-ttsCost, "process-video", "Unlock remaining credits due to TTS deduction error", nil)
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

	// Convert translated SRT to speech with target language using service_config
	log.Printf("Starting TTS conversion with language: %s, speaking rate: %f", targetLanguage, speakingRate)

	// Lấy service config cho TTS
	ttsServiceName, ttsModelAPIName, err := pricingService.GetActiveServiceForType("text_to_speech")
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost-ttsCost, "process-video", "Unlock remaining credits due to TTS service config error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active text-to-speech service"})
		return
	}

	ttsPath, err := service.ConvertSRTToSpeechWithService(string(srtContentBytes), videoDir, speakingRate, targetLanguage, ttsServiceName, ttsModelAPIName)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperCost-translationCost-ttsCost, "process-video", "Unlock remaining credits due to TTS conversion error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrProcessingFailed, err)
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

	// Burn subtitle vào video với solid background box
	finalVideoPath := mergedVideoPath
	if mergedVideoPath != "" && translatedSRTPath != "" {
		log.Printf("Attempting to burn subtitle: %s", translatedSRTPath)

		// Check if SRT file exists and has content
		if srtContent, err := os.ReadFile(translatedSRTPath); err != nil {
			log.Printf("Failed to read SRT file: %v", err)
			finalVideoPath = mergedVideoPath
		} else if len(strings.TrimSpace(string(srtContent))) == 0 {
			log.Printf("SRT file is empty: %s", translatedSRTPath)
			finalVideoPath = mergedVideoPath
		} else {
			// Try SRT method first
			burnedVideoPath, err := service.BurnSubtitleWithBackground(mergedVideoPath, translatedSRTPath, videoDir, subtitleColor, subtitleBgColor)
			if err != nil {
				log.Printf("SRT method failed: %v", err)

				// Try ASS method as fallback
				log.Printf("Trying ASS method as fallback...")
				burnedVideoPath, err = service.BurnSubtitleWithASS(mergedVideoPath, translatedSRTPath, videoDir, subtitleColor, subtitleBgColor)
				if err != nil {
					log.Printf("ASS method also failed: %v", err)
					// Nếu cả hai method đều thất bại, vẫn dùng video đã merge
					finalVideoPath = mergedVideoPath
				} else {
					finalVideoPath = burnedVideoPath
					log.Printf("Successfully burned subtitle with ASS method: %s", burnedVideoPath)
				}
			} else {
				finalVideoPath = burnedVideoPath
				log.Printf("Successfully burned subtitle with SRT method: %s", burnedVideoPath)
			}
		}
	} else {
		log.Printf("Skipping subtitle burn: mergedVideoPath=%s, translatedSRTPath=%s", mergedVideoPath, translatedSRTPath)
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
		Transcript     string `json:"transcript" binding:"required"`
		TargetLanguage string `json:"target_language"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Default to Vietnamese if not provided
	if req.TargetLanguage == "" {
		req.TargetLanguage = "vi"
	}

	// Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(req.Transcript, apiKey, req.TargetLanguage)
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
	// Lấy file trước để kiểm tra sớm
	file, err := c.FormFile("file")
	if err != nil {
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
		return
	}

	// Tạo thư mục tạm để kiểm tra duration
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(file.Filename))
	tempDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		util.HandleError(c, http.StatusInternalServerError, util.ErrDirectoryCreation, err)
		return
	}

	// Lưu file tạm để kiểm tra duration
	tempVideoPath := filepath.Join(tempDir, uniqueName)
	if err := c.SaveUploadedFile(file, tempVideoPath); err != nil {
		util.CleanupDir(tempDir)
		util.HandleError(c, http.StatusInternalServerError, util.ErrFileUploadFailed, err)
		return
	}

	// Tách audio tạm để kiểm tra duration
	tempAudioPath, err := util.ProcessfileToDir(c, file, tempDir)
	if err != nil {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}

	// Kiểm tra duration < 7 phút
	duration, _ := util.GetAudioDuration(tempAudioPath)

	// Nếu pass tất cả kiểm tra, tiếp tục xử lý
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	creditService := service.NewCreditService()
	pricingService := service.NewPricingService()

	// Tạo trạng thái process TikTok Optimizer (processing)
	processStatus := config.UserProcessStatus{
		UserID:      userID,
		Status:      "processing",
		ProcessType: "tiktok-optimize",
		StartedAt:   time.Now(),
	}
	if err := config.Db.Create(&processStatus).Error; err != nil {
		util.CleanupDir(tempDir)
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

	// Sử dụng thư mục tạm đã tạo
	videoDir := tempDir
	videoPath := tempVideoPath
	audioPath := tempAudioPath

	// Lấy thông tin bổ sung
	currentCaption := c.PostForm("current_caption")
	targetAudience := c.PostForm("target_audience")
	targetLanguage := c.PostForm("target_language")
	if targetAudience == "" {
		targetAudience = "general"
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

	// --- TÍNH PHÍ CHO TIKTOK OPTIMIZATION ---
	// Ước tính cost dựa trên độ phức tạp của analysis
	gptCost := whisperCost * 0.5 // Giảm cost vì sử dụng hybrid approach

	// --- LOCK CREDIT TRƯỚC KHI XỬ LÝ ---
	totalCost := whisperCost + gptCost
	_, err = creditService.LockCredits(userID, totalCost, "tiktok-optimizer", "Lock credit for TikTok Optimizer", nil)
	if err != nil {
		config.Db.Model(&processStatus).Updates(map[string]interface{}{
			"status":       "failed",
			"completed_at": time.Now(),
		})
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
			creditService.UnlockCredits(userID, totalCost, "tiktok-optimizer", "Unlock due to panic", nil)
			panic(r)
		}
	}()

	// --- SỬ DỤNG TIKTOK SERVICE MANAGER VỚI SERVICE_CONFIG VÀ TÍNH PHÍ ---
	// Tạo TikTok Service Manager
	tikTokManager := service.NewTikTokServiceManager(config.Db)

	// Analyze content category
	contentCategory := service.AnalyzeContentCategory(transcript, currentCaption)

	// Generate optimized content với service config
	localizedContent, err := tikTokManager.GenerateOptimizedContentWithConfig(transcript, contentCategory, targetLanguage, duration, apiKey)
	// err ở đây luôn nil vì đã check trước; xóa nhánh điều kiện thừa tránh cảnh báo linter

	// Convert localized content to analysis result
	analysisResult := &service.TikTokAnalysisResult{
		HookScore:         service.CalculateHookScore(transcript, duration, contentCategory),
		ViralPotential:    service.CalculateViralPotential(transcript, duration, contentCategory, targetAudience),
		OptimizationTips:  localizedContent.OptimizationTips,
		TrendingHashtags:  localizedContent.TrendingHashtags,
		SuggestedCaption:  localizedContent.SuggestedCaption,
		BestPostingTime:   service.GetBestPostingTime(targetAudience, contentCategory),
		EngagementPrompts: localizedContent.EngagementPrompts,
		CallToAction:      localizedContent.CallToAction,
		ContentCategory:   contentCategory,
		TargetAudience:    targetAudience,
		TrendingTopics:    localizedContent.TrendingTopics,
		VideoPacing:       service.AnalyzeVideoPacing(duration, contentCategory),
		ThumbnailTips:     service.GenerateThumbnailTips(contentCategory, targetAudience, targetLanguage),
		SoundSuggestions:  service.GenerateSoundSuggestions(contentCategory, targetAudience, targetLanguage),
		AnalysisMethod:    "ai-enhanced",
	}
	if err != nil {
		creditService.UnlockCredits(userID, totalCost, "tiktok-optimizer", "Unlock due to analysis error", nil)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analysis error: " + err.Error()})
		return
	}

	// Convert analysis result to map for compatibility
	result := map[string]interface{}{
		"hook_score":         analysisResult.HookScore,
		"viral_potential":    analysisResult.ViralPotential,
		"optimization_tips":  analysisResult.OptimizationTips,
		"trending_hashtags":  analysisResult.TrendingHashtags,
		"suggested_caption":  analysisResult.SuggestedCaption,
		"best_posting_time":  analysisResult.BestPostingTime,
		"engagement_prompts": analysisResult.EngagementPrompts,
		"call_to_action":     analysisResult.CallToAction,
		"content_category":   analysisResult.ContentCategory,
		"target_audience":    analysisResult.TargetAudience,
		"trending_topics":    analysisResult.TrendingTopics,
		"video_pacing":       analysisResult.VideoPacing,
		"thumbnail_tips":     analysisResult.ThumbnailTips,
		"sound_suggestions":  analysisResult.SoundSuggestions,
		"analysis_method":    analysisResult.AnalysisMethod,
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
	if v, ok := result["hook_score"].(int); ok {
		captionHistory.HookScore = v
	}
	if v, ok := result["viral_potential"].(int); ok {
		captionHistory.ViralPotential = v
	}
	if v, ok := result["trending_hashtags"].([]string); ok {
		b, _ := json.Marshal(v)
		captionHistory.TrendingHashtags = datatypes.JSON(b)
	}
	if v, ok := result["suggested_caption"].(string); ok {
		captionHistory.SuggestedCaption = v
	}
	if v, ok := result["best_posting_time"].(string); ok {
		captionHistory.BestPostingTime = v
	}
	if v, ok := result["optimization_tips"].([]string); ok {
		b, _ := json.Marshal(v)
		captionHistory.OptimizationTips = datatypes.JSON(b)
	}
	if v, ok := result["engagement_prompts"].([]string); ok {
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
		creditService.UnlockCredits(userID, gptCost, "tiktok-optimizer", "Unlock GPT credit after Whisper deduction error", &captionHistory.ID)
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho Whisper",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}
	err = creditService.DeductCredits(userID, gptCost, "tiktok-optimizer", "TikTok Optimization Hybrid", &captionHistory.ID, "per_request", 1.0)
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho GPT",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Trả về kết quả với các trường mới
	result["transcript"] = transcript
	result["segments"] = segments
	result["id"] = captionHistory.ID
	result["process_id"] = processStatus.ID
	result["duration"] = duration
	config.Db.Model(processStatus).Updates(map[string]interface{}{
		"status":      "completed",
		"CompletedAt": time.Now(),
		"video_id":    captionHistory.ID,
	})
	c.JSON(http.StatusOK, result)
}

// CreateSubtitleHandler tạo file SRT từ video
func CreateSubtitleHandler(c *gin.Context) {
	// Lấy thông tin file đã validate từ middleware
	videoFileInterface, exists := c.Get("validated_file")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File chưa được validate"})
		return
	}
	videoFile := videoFileInterface.(*multipart.FileHeader)

	tempDir := c.GetString("temp_dir")
	tempAudioPath := c.GetString("temp_audio_path")

	// Nếu pass tất cả kiểm tra, tiếp tục xử lý
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		util.CleanupDir(tempDir)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	creditService := service.NewCreditService()
	pricingService := service.NewPricingService()

	// Lấy process status từ middleware
	processStatusInterface, exists := c.Get("process_status")
	if !exists {
		util.CleanupDir(tempDir)
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

	// Sử dụng thư mục tạm đã tạo
	videoDir := tempDir
	audioPath := tempAudioPath

	// Lấy các tham số
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi" // Default to Vietnamese
	}

	isBilingual := c.PostForm("is_bilingual") == "true"

	// Tính toán chi phí Whisper
	whisperCost, err := pricingService.CalculateWhisperCost(1.0) // Ước tính 1 phút
	if err != nil {
		config.Db.Model(processStatus).Update("status", "failed")
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí Whisper"})
		return
	}

	// Tính toán chi phí translation (nếu song ngữ)
	var translationCost float64 = 0
	var translationTokens int = 0
	if isBilingual {
		// Ước tính chi phí translation dựa trên độ dài video
		serviceName, _, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ Gemini"})
			return
		}
		if strings.Contains(serviceName, "gpt") {
			var tk int
			translationCost, tk, err = pricingService.CalculateGPTCost("sample text")
			translationTokens = tk
		} else {
			var tk int
			translationCost, tk, _, err = pricingService.CalculateLLMCost("sample text", serviceName)
			translationTokens = tk
		}
		if err != nil {
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí Gemini"})
			return
		}
	}

	totalCost := whisperCost + translationCost

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
		config.Db.Model(&processStatus).Updates(map[string]interface{}{
			"status":       "failed",
			"completed_at": time.Now(),
		})
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
	baseName := strings.TrimSuffix(filepath.Base(videoFile.Filename), filepath.Ext(videoFile.Filename))
	originalSRTPath := filepath.Join(videoDir, baseName+"_original.srt")
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
		// Lấy service config cho translation
		serviceName, srtModelAPIName, err := pricingService.GetActiveServiceForType("srt_translation")
		if err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to service config error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ Gemini"})
			return
		}

		// Dịch SRT theo service được chọn
		var translatedSRTContent string
		if strings.Contains(serviceName, "gpt") {
			translatedSRTContent, err = service.TranslateSRTFileWithGPT(originalSRTPath, apiKey, srtModelAPIName, targetLanguage)
		} else {
			translatedSRTContent, err = service.TranslateSRTFileWithModelAndLanguage(originalSRTPath, geminiKey, srtModelAPIName, targetLanguage)
		}
		if err != nil {
			creditService.UnlockCredits(userID, totalCost, "create-subtitle", "Unlock credits due to translation error", nil)
			config.Db.Model(processStatus).Update("status", "failed")
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể dịch SRT: %v", err)})
			return
		}

		// Lưu file SRT đã dịch
		translatedSRTPath = filepath.Join(videoDir, baseName+"_"+targetLanguage+".srt")
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

		var serviceDescription string
		if strings.Contains(serviceName, "gpt") {
			serviceDescription = "GPT dịch SRT"
		} else {
			serviceDescription = "Gemini dịch SRT"
		}

		if err := creditService.DeductCredits(userID, translationCost, serviceName, serviceDescription, &captionHistory.ID, "per_token", float64(translationTokens)); err != nil {
			creditService.UnlockCredits(userID, totalCost-whisperCost-translationCost, "create-subtitle", "Unlock remaining credits due to translation deduction error", nil)
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

// ProcessVideoParallelHandler xử lý video với parallel processing
func ProcessVideoParallelHandler(c *gin.Context) {
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
	pricingService := service.NewPricingService()

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

	// Get subtitle color parameters
	subtitleColor := c.PostForm("subtitle_color")
	if subtitleColor == "" {
		subtitleColor = "#FFFFFF" // Default to white
	}
	subtitleBgColor := c.PostForm("subtitle_bgcolor")
	if subtitleBgColor == "" {
		subtitleBgColor = "#808080" // Default to gray
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.HandleError(c, http.StatusBadRequest, util.ErrFileUploadFailed, err)
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

	// Check for custom SRT file
	customSrtFile, err := c.FormFile("custom_srt")
	var hasCustomSrt bool = false
	var customSrtPath string

	if err == nil && customSrtFile != nil {
		hasCustomSrt = true
		customSrtPath = filepath.Join(videoDir, "custom.srt")
		if err := c.SaveUploadedFile(customSrtFile, customSrtPath); err != nil {
			if processID > 0 {
				processService.UpdateProcessStatus(processID, "failed")
			}
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save custom SRT file"})
			return
		}
	}

	// Lấy các tham số tuỳ chỉnh từ form-data
	backgroundVolume := 1.2
	ttsVolume := 1.5
	speakingRate := 1.2

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

	// Tính chi phí ước tính
	durationMinutes := duration / 60.0
	estimatedCostWithMarkup, err := pricingService.EstimateProcessVideoCostWithMarkup(durationMinutes, 1000, 1000, userID) // Ước tính
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
	_, err = creditService.LockCredits(userID, estimatedCost, "process-video", "Lock credit for parallel video processing", nil)
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

	// Tạo parallel processor
	parallelProcessor := service.NewProcessVideoParallel(videoPath, audioPath, videoDir, targetLanguage, apiKey, geminiKey)
	parallelProcessor.HasCustomSrt = hasCustomSrt
	parallelProcessor.CustomSrtPath = customSrtPath
	parallelProcessor.SubtitleColor = subtitleColor
	parallelProcessor.SubtitleBgColor = subtitleBgColor
	parallelProcessor.BackgroundVolume = backgroundVolume
	parallelProcessor.TTSVolume = ttsVolume
	parallelProcessor.SpeakingRate = speakingRate

	// Xử lý song song
	result, err := parallelProcessor.ProcessParallel()
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to processing error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)

		// Kiểm tra nếu là lỗi quá tải
		if strings.Contains(err.Error(), "Hệ thống đang quá tải") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang quá tải, vui lòng thử lại sau"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Xử lý video thất bại: %v", err)})
		}
		return
	}

	// Save to database
	segmentsJSON, _ := json.Marshal(result.Segments)
	captionHistory := config.CaptionHistory{
		UserID:              userID,
		VideoFilename:       videoPath,
		VideoFilenameOrigin: file.Filename,
		Transcript:          result.Transcript,
		Segments:            segmentsJSON,
		SegmentsVi:          segmentsJSON, // Sử dụng segments gốc cho segments_vi
		ProcessType:         "process-video",
		SrtFile:             result.TranslatedSRTPath,
		OriginalSrtFile:     result.OriginalSRTPath,
		TTSFile:             result.TTSPath,
		MergedVideoFile:     result.FinalVideoPath,
		BackgroundMusic:     result.BackgroundPath,
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

	// Trừ credit theo chi phí thực tế theo từng dịch vụ để minh bạch transaction
	// 1) Whisper (per_minute)
	whisperBase, err := pricingService.CalculateWhisperCost(durationMinutes)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to cost calculation error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí Whisper"})
		return
	}
	if err := creditService.DeductCredits(userID, whisperBase, "whisper", "Whisper transcribe", &captionHistory.ID, "per_minute", durationMinutes); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase, "process-video", "Unlock remaining credits due to Whisper deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho Whisper"})
		return
	}

	// 2) Translation (Gemini/GPT) per_token
	serviceName, _, err := pricingService.GetActiveServiceForType("srt_translation")
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase, "process-video", "Unlock remaining credits due to translation service error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thông tin dịch vụ dịch SRT"})
		return
	}
	// Tính chi phí dịch theo input/output riêng
	var inputText, outputText string
	if result.OriginalSRTPath != "" {
		if b, e := os.ReadFile(result.OriginalSRTPath); e == nil {
			inputText = string(b)
		}
	}
	if result.TranslatedSRTPath != "" {
		if b, e := os.ReadFile(result.TranslatedSRTPath); e == nil {
			outputText = string(b)
		}
	}
	if inputText == "" {
		inputText = result.Transcript
	}
	if outputText == "" {
		outputText = result.Transcript
	}
	inCost, outCost, inTok, outTok, _, err := pricingService.CalculateLLMCostSplit(inputText, outputText, serviceName)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase, "process-video", "Unlock remaining credits due to translation cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí dịch SRT"})
		return
	}
	translationBase := inCost + outCost
	translationTokens := inTok + outTok
	var translationDesc string
	if strings.Contains(serviceName, "gpt") {
		translationDesc = "GPT dịch SRT"
	} else {
		translationDesc = "Gemini dịch SRT"
	}
	if err := creditService.DeductCredits(userID, translationBase, serviceName, translationDesc, &captionHistory.ID, "per_token", float64(translationTokens)); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase-translationBase, "process-video", "Unlock remaining credits due to translation deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho dịch SRT"})
		return
	}

	// 3) TTS per_character
	ttsBase, err := pricingService.CalculateTTSCost(result.Transcript, true)
	if err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase-translationBase, "process-video", "Unlock remaining credits due to TTS cost error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính toán chi phí TTS"})
		return
	}
	if err := creditService.DeductCredits(userID, ttsBase, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len([]rune(result.Transcript)))); err != nil {
		creditService.UnlockCredits(userID, estimatedCost-whisperBase-translationBase-ttsBase, "process-video", "Unlock remaining credits due to TTS deduction error", nil)
		if processID > 0 {
			processService.UpdateProcessStatus(processID, "failed")
		}
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ credit cho TTS"})
		return
	}

	// 4) Unlock phần còn lại nếu ước tính > chi phí thực tế
	// Tính final price để khóa/ mở khóa chính xác (áp dụng markup)
	finalWhisper, _ := pricingService.CalculateUserPrice(whisperBase, "whisper", userID)
	normalizedTrans := serviceName
	if strings.Contains(serviceName, "gpt") {
		normalizedTrans = "gpt"
	}
	if strings.Contains(serviceName, "gemini") {
		normalizedTrans = "gemini"
	}
	finalTrans, _ := pricingService.CalculateUserPrice(translationBase, normalizedTrans, userID)
	finalTTS, _ := pricingService.CalculateUserPrice(ttsBase, "tts", userID)
	finalTotal := finalWhisper + finalTrans + finalTTS
	if remaining := estimatedCost - finalTotal; remaining > 0.000001 {
		_ = creditService.UnlockCredits(userID, remaining, "process-video", "Unlock remaining credits after processing", nil)
	}

	// Cập nhật trạng thái process thành completed
	if processID > 0 {
		processService.UpdateProcessStatus(processID, "completed")
		processService.UpdateProcessVideoID(processID, captionHistory.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                 "Video processed successfully with parallel processing",
		"background_music":        result.BackgroundPath,
		"srt_file":                result.TranslatedSRTPath,
		"original_srt_file":       result.OriginalSRTPath,
		"tts_file":                result.TTSPath,
		"merged_video":            result.FinalVideoPath,
		"transcript":              result.Transcript,
		"segments":                result.Segments,
		"segments_vi":             result.Segments,
		"id":                      captionHistory.ID,
		"process_id":              processID,
		"processing_time":         result.ProcessingTime.String(),
		"performance_improvement": "Parallel processing completed",
	})
}

// GetProcessingProgressHandler lấy tiến độ xử lý
func GetProcessingProgressHandler(c *gin.Context) {
	processIDStr := c.Param("process_id")
	if processIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Process ID is required"})
		return
	}

	// Convert string to uint
	processID, err := strconv.ParseUint(processIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid process ID format"})
		return
	}

	// Lấy thông tin process từ database
	var processStatus config.UserProcessStatus
	err = config.Db.Where("id = ?", uint(processID)).First(&processStatus).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Process not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"process_id":   processID,
		"status":       processStatus.Status,
		"process_type": processStatus.ProcessType,
		"started_at":   processStatus.StartedAt,
		"completed_at": processStatus.CompletedAt,
		"created_at":   processStatus.CreatedAt,
		"updated_at":   processStatus.UpdatedAt,
	})
}

// ProcessVideoAsyncHandler xử lý video bất đồng bộ qua queue system
func ProcessVideoAsyncHandler(c *gin.Context) {
	// Nhận file video
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

	// Lấy user_id từ token
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy process_id từ middleware
	processID := c.GetUint("process_id")

	// Lấy các tham số
	targetLanguage := c.PostForm("target_language")
	if targetLanguage == "" {
		targetLanguage = "vi"
	}

	serviceName := c.PostForm("service_name")
	if serviceName == "" {
		serviceName = "gpt-4o-mini"
	}

	// Get subtitle color parameters
	subtitleColor := c.PostForm("subtitle_color")
	if subtitleColor == "" {
		subtitleColor = "#FFFFFF" // Default to white
	}
	subtitleBgColor := c.PostForm("subtitle_bgcolor")
	if subtitleBgColor == "" {
		subtitleBgColor = "#808080" // Default to gray
	}

	// Get volume and rate parameters
	backgroundVolume := 1.2
	if v := c.PostForm("background_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			backgroundVolume = f
		}
	}
	ttsVolume := 1.5
	if v := c.PostForm("tts_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ttsVolume = f
		}
	}
	speakingRate := 1.2
	if v := c.PostForm("speaking_rate"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			speakingRate = f
		}
	}

	// Tạo thư mục lưu file
	timestamp := time.Now().UnixNano()
	baseName := strings.TrimSuffix(filepath.Base(videoFile.Filename), filepath.Ext(videoFile.Filename))
	uniqueName := fmt.Sprintf("%d_%s%s", timestamp, baseName, filepath.Ext(videoFile.Filename))
	videoDir := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName)))
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video directory"})
		return
	}

	// Lưu file video
	safeVideoName := strings.ReplaceAll(videoFile.Filename, " ", "_")
	videoPath := filepath.Join(videoDir, safeVideoName)
	if err := c.SaveUploadedFile(videoFile, videoPath); err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}

	// Kiểm tra duration
	audioPath, err := util.ProcessfileToDir(c, videoFile, videoDir)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio"})
		return
	}
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 420 {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 7 phút."})
		return
	}

	// Tính toán chi phí ước tính
	creditService := service.NewCreditService()

	// Ước tính chi phí (sử dụng ước tính đơn giản)
	estimatedCost := 0.1 // Ước tính cơ bản, sẽ được tính chính xác khi xử lý

	// Lock credits
	_, err = creditService.LockCredits(userID, estimatedCost, "process-video", "Lock credit for process video", nil)
	if err != nil {
		util.CleanupDir(videoDir)
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Không đủ credit cho xử lý video",
			"warning": "Số dư tài khoản của bạn không đủ để sử dụng dịch vụ này. Vui lòng nạp thêm credit để tiếp tục sử dụng!",
		})
		return
	}

	// Check for custom SRT file
	customSrtFile, err := c.FormFile("custom_srt")
	var hasCustomSrt bool = false
	var customSrtPath string
	if err == nil && customSrtFile != nil {
		hasCustomSrt = true
		customSrtPath = filepath.Join(videoDir, "custom.srt")
		if err := c.SaveUploadedFile(customSrtFile, customSrtPath); err != nil {
			util.CleanupDir(videoDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save custom SRT file"})
			return
		}
	}

	// Tạo job process-video và enqueue vào queue
	jobID := fmt.Sprintf("processvideo_%d_%d", userID, timestamp)
	job := &service.AudioProcessingJob{
		ID:               jobID,
		JobType:          "process-video",
		UserID:           userID,
		ProcessID:        processID,
		FileName:         safeVideoName,
		VideoDir:         videoDir,
		AudioPath:        audioPath,
		Priority:         5,
		MaxDuration:      600, // 10 phút
		TargetLanguage:   targetLanguage,
		ServiceName:      serviceName,
		SubtitleColor:    subtitleColor,
		SubtitleBgColor:  subtitleBgColor,
		HasCustomSrt:     hasCustomSrt,
		CustomSrtPath:    customSrtPath,
		BackgroundVolume: backgroundVolume,
		TTSVolume:        ttsVolume,
		SpeakingRate:     speakingRate,
	}

	queueService := service.GetQueueService()
	if queueService == nil {
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to queue service not initialized", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Queue service not initialized"})
		return
	}

	if err := queueService.EnqueueJob(job); err != nil {
		creditService.UnlockCredits(userID, estimatedCost, "process-video", "Unlock due to enqueue job error", nil)
		util.CleanupDir(videoDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	// Trả về process_id để frontend tracking
	c.JSON(http.StatusOK, gin.H{
		"message":    "Đã nhận video, đang xử lý...",
		"process_id": jobID,
	})
}
