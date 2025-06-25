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

	// Tính duration file audio (Whisper)
	duration, _ := util.GetAudioDuration(saveAudioPath) // trả về giây
	whisperTokens := int(duration/60.0*6 + 0.5)         // Làm tròn lên
	if whisperTokens < 6 {
		whisperTokens = 6 // Tối thiểu 6 tokens
	}
	if err := DeductUserToken(userID, whisperTokens, "whisper", "Whisper transcribe", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Whisper"})
		return
	}
	// Gọi Whisper để lấy transcript
	transcript, segments, err := service.TranscribeWhisperOpenAI(saveAudioPath, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Whisper error"})
		return
	}

	//Gọi GPT để gợi ý caption & hashtag
	captionsAndHashtag, err := service.GenerateSuggestion(transcript, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	jsonData, err := json.Marshal(segments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}
	// Trừ token cho Gemini dịch
	geminiTokens := 3
	if err := DeductUserToken(userID, geminiTokens, "gemini", "Gemini dịch caption", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Gemini"})
		return
	}
	translatedSegments, err := service.TranslateSegmentsWithGemini(string(jsonData), geminiKey)
	if err != nil {
		fmt.Println("Error translating segments:", err)
		return
	}
	jsonDataVi, err := json.Marshal(translatedSegments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GPT error"})
		return
	}

	// Create SRT file from translated segments
	srtContent := createSRT(translatedSegments)
	srtPath := filepath.Join("./storage", strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))+"_vi.srt")
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		log.Printf("Error creating SRT file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create SRT file: %v", err),
		})
		return
	}

	// Convert translated SRT to speech
	ttsPath, err := service.ConvertSRTToSpeech(srtContent)
	if err != nil {
		log.Printf("Error converting SRT to speech: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to convert SRT to speech",
		})
		return
	}

	// Lưu history
	result := config.Db.Create(&config.CaptionHistory{
		UserID:        userID,
		VideoFilename: file.Filename,
		Transcript:    transcript,
		Suggestion:    captionsAndHashtag,
		Segments:      jsonData,
		SegmentsVi:    jsonDataVi,
		SrtFile:       srtPath,
		TTSFile:       ttsPath,
		CreatedAt:     time.Now(),
	})

	if result.Error != nil {
		log.WithError(result.Error).Error("Failed to save history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transcript":         transcript,
		"captionsAndHashtag": captionsAndHashtag,
		"srt_file":           srtPath,
		"tts_file":           ttsPath,
		"id":                 result.RowsAffected,
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
	tempDir := "./storage/temp"
	filename := filepath.Join(tempDir, uniqueName)

	// Save the uploaded file
	if err := c.SaveUploadedFile(file, filename); err != nil {
		log.Printf("Error saving video file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save video file",
		})
		return
	}

	log.Printf("Video file saved to: %s", filename)

	// Process the video to extract audio
	audioPath, err := service.ProcessVideoToAudio(filename)
	if err != nil {
		log.Printf("Error processing video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process video: %v", err),
		})
		return
	}

	// Tính duration file audio (Whisper)
	duration, _ := util.GetAudioDuration(audioPath)
	whisperTokens := int(duration/60.0*6 + 0.5)
	if whisperTokens < 6 {
		whisperTokens = 6
	}
	if err := DeductUserToken(userID, whisperTokens, "whisper", "Whisper transcribe", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Whisper"})
		return
	}

	// Extract background music
	backgroundPath, err := service.ExtractBackgroundMusic(audioPath, uniqueName)
	if err != nil {
		log.Printf("Error extracting background music: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to extract background music: %v", err),
		})
		return
	}

	// Extract vocals
	vocalsPath, err := service.ExtractVocals(audioPath, uniqueName)
	if err != nil {
		log.Printf("Error extracting vocals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to extract vocals: %v", err),
		})
		return
	}

	// Get transcript from vocals
	transcript, segments, err := service.TranscribeWhisperOpenAI(vocalsPath, apiKey)
	if err != nil {
		log.Printf("Error transcribing vocals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to transcribe vocals: %v", err),
		})
		return
	}

	// Create original SRT file from segments
	originalSrtContent := createSRT(segments)
	originalSrtPath := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_or.srt")
	if err := os.WriteFile(originalSrtPath, []byte(originalSrtContent), 0644); err != nil {
		log.Printf("Error creating original SRT file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create original SRT file: %v", err),
		})
		return
	}

	// Translate segments to Vietnamese
	jsonData, err := json.Marshal(segments)
	if err != nil {
		log.Printf("Error marshaling segments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process segments: %v", err),
		})
		return
	}

	// Trừ token cho Gemini dịch
	geminiTokens := 3
	if err := DeductUserToken(userID, geminiTokens, "gemini", "Gemini dịch caption", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho Gemini"})
		return
	}
	translatedSegments, err := service.TranslateSegmentsWithGemini(string(jsonData), geminiKey)
	if err != nil {
		log.Printf("Error translating segments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to translate segments: %v", err),
		})
		return
	}

	// Create SRT file from translated segments
	srtContent := createSRT(translatedSegments)
	srtPath := filepath.Join("./storage", strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))+"_vi.srt")
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		log.Printf("Error creating SRT file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create SRT file: %v", err),
		})
		return
	}

	// Trừ token cho Google TTS (ước tính: 1 token/62.5 ký tự, dùng tổng ký tự của srtContent)
	tokenPerChar := 1.0 / 62.5
	ttsTokens := int(float64(len(srtContent))*tokenPerChar + 0.9999)
	if ttsTokens < 1 {
		ttsTokens = 1
	}
	if err := DeductUserToken(userID, ttsTokens, "tts", "Google TTS", nil); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Không đủ token cho TTS"})
		return
	}

	// Convert translated SRT to speech
	ttsPath, err := service.ConvertSRTToSpeech(srtContent)
	if err != nil {
		log.Printf("Error converting SRT to speech: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to convert SRT to speech",
		})
		return
	}

	// Merge video with background music and TTS audio
	mergedVideoPath, err := service.MergeVideoWithAudio(filename, backgroundPath, ttsPath)
	if err != nil {
		log.Printf("Error merging video with audio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to merge video with audio",
		})
		return
	}

	// Convert segments to JSON
	segmentsJSON, err := json.Marshal(segments)
	if err != nil {
		log.Printf("Error marshaling segments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process segments",
		})
		return
	}

	segmentsViJSON, err := json.Marshal(translatedSegments)
	if err != nil {
		log.Printf("Error marshaling Vietnamese segments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process Vietnamese segments",
		})
		return
	}

	// Save to database
	captionHistory := config.CaptionHistory{
		UserID:          userID,
		VideoFilename:   uniqueName,
		Transcript:      transcript,
		Segments:        segmentsJSON,
		SegmentsVi:      segmentsViJSON,
		BackgroundMusic: backgroundPath,
		SrtFile:         srtPath,
		OriginalSrtFile: originalSrtPath,
		TTSFile:         ttsPath,
		MergedVideoFile: mergedVideoPath,
		CreatedAt:       time.Now(),
	}

	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("Error saving to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save to database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Video processed successfully",
		"background_music":  backgroundPath,
		"srt_file":          srtPath,
		"original_srt_file": originalSrtPath,
		"tts_file":          ttsPath,
		"merged_video":      mergedVideoPath,
		"transcript":        transcript,
		"segments":          segments,
		"segments_vi":       translatedSegments,
		"id":                captionHistory.ID,
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
