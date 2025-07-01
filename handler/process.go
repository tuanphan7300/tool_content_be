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
		UserID:        userID,
		VideoFilename: file.Filename,
		Transcript:    transcript,
		Segments:      jsonData,
		CreatedAt:     time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history"})
		return
	}
	videoID := &captionHistory.ID
	if err := DeductUserToken(userID, whisperTokens, "whisper", "Whisper transcribe", videoID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Whisper"})
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
	if err := DeductUserToken(userID, geminiTokens, "gemini", "Gemini dịch caption", videoID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Gemini"})
		return
	}
	translatedSegments, err := service.TranslateSegmentsWithGemini(string(jsonData), geminiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gemini error"})
		return
	}
	jsonDataVi, _ := json.Marshal(translatedSegments)
	// Create SRT file from translated segments
	srtContent := createSRT(translatedSegments)
	srtPath := filepath.Join("./storage", strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))+"_vi.srt")
	_ = os.WriteFile(srtPath, []byte(srtContent), 0644)
	// Convert translated SRT to speech
	ttsPath, _ := service.ConvertSRTToSpeech(srtContent, "./storage", 1.2)
	// Update history
	captionHistory.Suggestion = captionsAndHashtag
	captionHistory.SegmentsVi = jsonDataVi
	captionHistory.SrtFile = srtPath
	captionHistory.TTSFile = ttsPath
	config.Db.Save(&captionHistory)
	c.JSON(http.StatusOK, gin.H{
		"transcript":         transcript,
		"captionsAndHashtag": captionsAndHashtag,
		"srt_file":           srtPath,
		"tts_file":           ttsPath,
		"id":                 captionHistory.ID,
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

	// Kiểm tra giới hạn Free (3 video/ngày/IP)
	if !limit.CheckFreeLimit(c.ClientIP()) {
		log.Warn("User exceeded free plan limit")
		c.JSON(http.StatusForbidden, gin.H{"error": "Vượt giới hạn 3 video/ngày. Nâng cấp Pro!"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("Error getting video file: %v", err)
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create video directory",
		})
		return
	}

	// Lưu video vào thư mục riêng
	videoPath := filepath.Join(videoDir, uniqueName)
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		log.Printf("Error saving video file: %v", err)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	// Kiểm tra duration < 10 phút
	duration, _ := util.GetAudioDuration(audioPath)
	if duration > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ cho phép video/audio dưới 10 phút."})
		return
	}

	// Gọi Whisper để lấy transcript và usage thực tế
	transcript, segments, whisperUsage, err := service.TranscribeWhisperOpenAI(audioPath, apiKey)
	if err != nil {
		log.Printf("Error transcribing vocals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transcribe vocals: %v", err)})
		return
	}
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
	// Save to database trước để lấy video_id
	segmentsJSON, _ := json.Marshal(segments)
	captionHistory := config.CaptionHistory{
		UserID:        userID,
		VideoFilename: uniqueName,
		Transcript:    transcript,
		Segments:      segmentsJSON,
		CreatedAt:     time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("Error saving to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save to database"})
		return
	}
	videoID := &captionHistory.ID
	if err := DeductUserToken(userID, whisperTokens, "whisper", "Whisper transcribe", videoID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Whisper"})
		return
	}
	// Translate segments to Vietnamese
	jsonData, _ := json.Marshal(segments)
	geminiTokens := int(float64(len(jsonData))/62.5 + 0.9999)
	if geminiTokens < 1 {
		geminiTokens = 1
	}
	if err := DeductUserToken(userID, geminiTokens, "gemini", "Gemini dịch caption", videoID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Gemini"})
		return
	}
	translatedSegments, err := service.TranslateSegmentsWithGemini(string(jsonData), geminiKey)
	if err != nil {
		log.Printf("Error translating segments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to translate segments: %v", err)})
		return
	}
	segmentsViJSON, _ := json.Marshal(translatedSegments)
	// Create SRT file from translated segments
	srtContent := createSRT(translatedSegments)
	srtPath := filepath.Join(videoDir, strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_vi.srt")
	_ = os.WriteFile(srtPath, []byte(srtContent), 0644)
	// Lấy các tham số tuỳ chỉnh từ form-data
	backgroundVolume := 1.2
	ttsVolume := 0.66
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
	// Trừ token cho Google TTS (tính theo ký tự của srtContent)
	tokenPerChar := 1.0 / 62.5
	ttsTokens := int(float64(len(srtContent))*tokenPerChar + 0.9999)
	if ttsTokens < 1 {
		ttsTokens = 1
	}
	if err := DeductUserToken(userID, ttsTokens, "tts", "Google TTS", videoID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho TTS"})
		return
	}
	// Convert translated SRT to speech
	ttsPath, _ := service.ConvertSRTToSpeech(srtContent, videoDir, speakingRate)
	// Merge video with background music and TTS audio
	backgroundPath, _ := service.ExtractBackgroundMusic(audioPath, uniqueName, videoDir)
	mergedVideoPath, _ := service.MergeVideoWithAudio(videoPath, backgroundPath, ttsPath, videoDir, backgroundVolume, ttsVolume)
	// Update history
	captionHistory.SegmentsVi = segmentsViJSON
	captionHistory.SrtFile = srtPath
	captionHistory.TTSFile = ttsPath
	captionHistory.MergedVideoFile = mergedVideoPath
	captionHistory.BackgroundMusic = backgroundPath
	config.Db.Save(&captionHistory)
	c.JSON(http.StatusOK, gin.H{
		"message":          "Video processed successfully",
		"background_music": backgroundPath,
		"srt_file":         srtPath,
		"tts_file":         ttsPath,
		"merged_video":     mergedVideoPath,
		"transcript":       transcript,
		"segments":         segments,
		"segments_vi":      translatedSegments,
		"id":               captionHistory.ID,
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
