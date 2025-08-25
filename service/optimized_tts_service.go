package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
)

// OptimizedTTSService xử lý TTS với concurrent processing và rate limiting
type OptimizedTTSService struct {
	client         *texttospeech.Client
	rateLimiter    *TTSRateLimiter
	mappingService *TTSMappingService
	maxConcurrent  int
	workerPool     chan struct{}
	ctx            context.Context
}

// TTSProcessingResult kết quả xử lý TTS
type TTSProcessingResult struct {
	SegmentIndex   int
	AudioPath      string
	Duration       float64
	Error          error
	ProcessingTime time.Duration
}

// TTSProcessingOptions tùy chọn xử lý TTS
type TTSProcessingOptions struct {
	TargetLanguage   string
	ServiceName      string
	SubtitleColor    string
	SubtitleBgColor  string
	BackgroundVolume float64
	TTSVolume        float64
	SpeakingRate     float64
	MaxConcurrent    int
	UserID           uint
	VoiceName        string // Thêm trường chọn giọng đọc
}

var (
	optimizedTTSService *OptimizedTTSService
	ttsServiceMutex     sync.Mutex
)

// InitOptimizedTTSService khởi tạo Optimized TTS Service
func InitOptimizedTTSService(apiKeyPath string, maxConcurrent int) (*OptimizedTTSService, error) {
	ttsServiceMutex.Lock()
	defer ttsServiceMutex.Unlock()

	if optimizedTTSService != nil {
		return optimizedTTSService, nil
	}

	// Khởi tạo Google TTS client
	ctx := context.Background()
	if apiKeyPath == "" {
		apiKeyPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if apiKeyPath == "" {
			apiKeyPath = "data/google_clound_tts_api.json"
		}
	}
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(apiKeyPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google TTS client: %v", err)
	}

	// Khởi tạo rate limiter
	rateLimiter := GetTTSRateLimiter()
	if rateLimiter == nil {
		return nil, fmt.Errorf("TTS rate limiter not initialized")
	}

	// Khởi tạo mapping service
	mappingService := GetTTSMappingService()

	// Tạo worker pool
	if maxConcurrent <= 0 {
		maxConcurrent = 6 // Mặc định 6 workers để tránh oversubscription trên máy ít CPU
	}

	workerPool := make(chan struct{}, maxConcurrent)

	optimizedTTSService = &OptimizedTTSService{
		client:         client,
		rateLimiter:    rateLimiter,
		mappingService: mappingService,
		maxConcurrent:  maxConcurrent,
		workerPool:     workerPool,
		ctx:            ctx,
	}

	log.Printf("Optimized TTS Service initialized with %d concurrent workers", maxConcurrent)
	return optimizedTTSService, nil
}

// GetOptimizedTTSService trả về instance của Optimized TTS Service
func GetOptimizedTTSService() *OptimizedTTSService {
	return optimizedTTSService
}

// ProcessSRTConcurrent xử lý SRT với concurrent processing
func (s *OptimizedTTSService) ProcessSRTConcurrent(
	srtContent string,
	videoDir string,
	options TTSProcessingOptions,
	jobID string,
) (string, error) {
	startTime := time.Now()
	log.Printf("🚀 [OPTIMIZED TTS] Bắt đầu concurrent TTS processing cho job %s", jobID)
	log.Printf("🔧 [OPTIMIZED TTS] Config: max_concurrent=%d, target_language=%s, speaking_rate=%.2f",
		options.MaxConcurrent, options.TargetLanguage, options.SpeakingRate)

	// Parse SRT content
	entries, err := parseSRT(srtContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse SRT: %v", err)
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no entries found in SRT content")
	}

	log.Printf("📊 [OPTIMIZED TTS] Đã parse được %d SRT entries", len(entries))

	// Tạo mapping cho job
	s.mappingService.CreateJobMapping(jobID, entries)

	// Tạo thư mục tạm cho segments
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("tts_concurrent_%s", jobID))
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("⚡ [OPTIMIZED TTS] Khởi động %d tasks để xử lý TTS (pool size: %d)...", len(entries), s.maxConcurrent)
	// Xử lý TTS với concurrent workers
	results := s.processTTSConcurrent(entries, tempDir, options, jobID)

	// Kiểm tra lỗi
	var failedSegments []int
	for _, result := range results {
		if result.Error != nil {
			failedSegments = append(failedSegments, result.SegmentIndex)
		}
	}

	if len(failedSegments) > 0 {
		log.Printf("⚠️ [OPTIMIZED TTS] %d segments failed processing: %v", len(failedSegments), failedSegments)
	} else {
		log.Printf("✅ [OPTIMIZED TTS] Tất cả %d segments đã được xử lý thành công!", len(entries))
	}

	log.Printf("🎵 [OPTIMIZED TTS] Bắt đầu tạo audio cuối cùng...")
	// Tạo audio cuối cùng
	outputPath := filepath.Join(videoDir, "tts_output.mp3")
	err = s.createFinalAudio(results, entries, outputPath, tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to create final audio: %v", err)
	}

	totalTime := time.Since(startTime)
	log.Printf("🏁 [OPTIMIZED TTS] Concurrent TTS processing hoàn thành cho job %s trong %v", jobID, totalTime)
	log.Printf("📈 [OPTIMIZED TTS] Performance: %d segments / %v = %.2f segments/second",
		len(entries), totalTime, float64(len(entries))/totalTime.Seconds())

	return outputPath, nil
}

// processTTSConcurrent xử lý TTS với concurrent workers
func (s *OptimizedTTSService) processTTSConcurrent(
	entries []SRTEntry,
	tempDir string,
	options TTSProcessingOptions,
	jobID string,
) []*TTSProcessingResult {
	log.Printf("🔄 [OPTIMIZED TTS] Bắt đầu concurrent processing: tasks=%d, pool=%d", len(entries), s.maxConcurrent)

	results := make([]*TTSProcessingResult, len(entries))
	var wg sync.WaitGroup
	var resultMutex sync.Mutex

	// Khởi động workers
	for i := 0; i < len(entries); i++ {
		wg.Add(1)
		go func(entry SRTEntry, index int) {
			defer wg.Done()

			log.Printf("🎯 [OPTIMIZED TTS] Worker %d bắt đầu xử lý segment %d: '%s'", index, index, truncateText(entry.Text, 50))

			// Acquire worker slot
			s.workerPool <- struct{}{}
			defer func() { <-s.workerPool }()

			log.Printf("⚡ [OPTIMIZED TTS] Worker %d đã acquire slot, bắt đầu xử lý TTS...", index)

			// Xử lý TTS cho segment này
			result := s.processSingleSegment(entry, index, tempDir, options, jobID)

			// Lưu kết quả thread-safe
			resultMutex.Lock()
			results[index] = result
			resultMutex.Unlock()

			if result.Error != nil {
				log.Printf("❌ [OPTIMIZED TTS] Worker %d failed: %v", index, result.Error)
			} else {
				log.Printf("✅ [OPTIMIZED TTS] Worker %d completed trong %v", index, result.ProcessingTime)
			}
		}(entries[i], i)
	}

	log.Printf("⏳ [OPTIMIZED TTS] Đang chờ tất cả %d workers hoàn thành...", len(entries))
	wg.Wait()
	log.Printf("🎯 [OPTIMIZED TTS] Tất cả workers đã hoàn thành!")

	return results
}

// processSingleSegment xử lý một segment đơn lẻ
func (s *OptimizedTTSService) processSingleSegment(
	entry SRTEntry,
	index int,
	tempDir string,
	options TTSProcessingOptions,
	jobID string,
) *TTSProcessingResult {
	startTime := time.Now()
	result := &TTSProcessingResult{
		SegmentIndex: index,
	}

	// Chờ rate limiter
	if !s.rateLimiter.WaitForSlot(30 * time.Second) {
		result.Error = fmt.Errorf("timeout waiting for rate limit slot")
		return result
	}

	// Reserve slot
	if !s.rateLimiter.ReserveSlot(options.UserID, entry.Text, fmt.Sprintf("%s_%d", jobID, index)) {
		result.Error = fmt.Errorf("failed to reserve rate limit slot")
		return result
	}

	// Gọi Google TTS API
	audioContent, err := s.callGoogleTTS(entry.Text, options)
	if err != nil {
		result.Error = fmt.Errorf("Google TTS API call failed: %v", err)
		s.updateSegmentMapping(jobID, index, map[string]interface{}{"error": result.Error})
		return result
	}

	// Lưu audio content
	segmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.mp3", index))
	err = os.WriteFile(segmentFile, audioContent, 0644)
	if err != nil {
		result.Error = fmt.Errorf("failed to save segment file: %v", err)
		s.updateSegmentMapping(jobID, index, map[string]interface{}{"error": result.Error})
		return result
	}

	// Convert to WAV và xử lý audio
	wavPath, duration, err := s.processAudioSegment(segmentFile, tempDir, index, options)
	if err != nil {
		result.Error = fmt.Errorf("audio processing failed: %v", err)
		s.updateSegmentMapping(jobID, index, map[string]interface{}{"error": result.Error})
		return result
	}

	// Cập nhật mapping
	s.updateSegmentMapping(jobID, index, map[string]interface{}{
		"google_api_resp": string(audioContent),
		"audio_duration":  duration,
		"adjusted_path":   wavPath,
		"processing_time": time.Since(startTime),
	})

	result.AudioPath = wavPath
	result.Duration = duration
	result.ProcessingTime = time.Since(startTime)

	log.Printf("Segment %d processed successfully in %v", index, result.ProcessingTime)
	return result
}

// callGoogleTTS gọi Google TTS API
func (s *OptimizedTTSService) callGoogleTTS(text string, options TTSProcessingOptions) ([]byte, error) {
	// Lấy voice settings cho target language với voice selection
	var languageCode, voiceName string
	if options.VoiceName != "" {
		// Sử dụng voice được chọn
		voices := GetAvailableVoices()
		if languageVoices, exists := voices[options.TargetLanguage]; exists {
			for _, voice := range languageVoices {
				if voice.Name == options.VoiceName {
					languageCode = voice.LanguageCode
					voiceName = voice.Name
					break
				}
			}
		}
	}

	// Fallback về default voice nếu không tìm thấy
	if languageCode == "" || voiceName == "" {
		languageCode, voiceName = getVoiceForLanguage(options.TargetLanguage)
	}

	// Tạo request
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         voiceName,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
			SpeakingRate:    options.SpeakingRate,
			SampleRateHertz: 44100,
		},
	}

	// Gọi API
	resp, err := s.client.SynthesizeSpeech(s.ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.AudioContent, nil
}

// processAudioSegment xử lý audio segment
func (s *OptimizedTTSService) processAudioSegment(
	mp3Path string,
	tempDir string,
	index int,
	options TTSProcessingOptions,
) (string, float64, error) {
	// Convert MP3 to WAV với volume boost
	wavPath := filepath.Join(tempDir, fmt.Sprintf("segment_%d.wav", index))
	cmd := exec.Command("ffmpeg",
		"-i", mp3Path,
		"-af", fmt.Sprintf("volume=%.2f", options.TTSVolume),
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "pcm_s16le",
		"-y",
		wavPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("FFmpeg conversion failed: %s", string(output))
	}

	// Lấy duration
	cmd = exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", wavPath)
	durationStr, err := cmd.Output()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get audio duration: %v", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationStr)), 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse audio duration: %v", err)
	}

	return wavPath, duration, nil
}

// createFinalAudio tạo audio cuối cùng từ tất cả segments
func (s *OptimizedTTSService) createFinalAudio(
	results []*TTSProcessingResult,
	entries []SRTEntry,
	outputPath string,
	tempDir string,
) error {
	// Tạo delayed files với adelay
	var delayedFiles []string
	for i, result := range results {
		if result.Error != nil || result.AudioPath == "" {
			continue
		}

		entry := entries[i]
		delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_%d.wav", i))

		// Áp dụng adelay để căn đúng thời điểm
		cmd := exec.Command("ffmpeg",
			"-i", result.AudioPath,
			"-af", fmt.Sprintf("adelay=%d|%d", int(entry.Start*1000), int(entry.Start*1000)),
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			delayedFile)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg delay error for segment %d: %s", i, string(output))
			continue
		}

		delayedFiles = append(delayedFiles, delayedFile)
	}

	if len(delayedFiles) == 0 {
		return fmt.Errorf("no valid segments to process")
	}

	// Chọn chiến lược mix: nếu số lượng file lớn, mix theo batch để tránh amix quá nhiều input cùng lúc
	const defaultBatchSize = 30
	if len(delayedFiles) > defaultBatchSize {
		return s.mixAudioFilesInBatches(delayedFiles, outputPath, tempDir, defaultBatchSize)
	}

	// Mix trực tiếp nếu số lượng nhỏ
	return s.mixAudioFiles(delayedFiles, outputPath)
}

// mixAudioFiles mix tất cả audio files
func (s *OptimizedTTSService) mixAudioFiles(inputFiles []string, outputPath string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files to mix")
	}

	if len(inputFiles) == 1 {
		// Chỉ có 1 file, copy trực tiếp
		cmd := exec.Command("ffmpeg",
			"-i", inputFiles[0],
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg copy failed: %s", string(output))
		}
		return nil
	}

	// Mix nhiều files
	args := []string{"-i"}
	args = append(args, inputFiles[0])
	for _, file := range inputFiles[1:] {
		args = append(args, "-i", file)
	}

	filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(inputFiles))
	args = append(args,
		"-filter_complex", filter,
		"-map", "[out]",
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "libmp3lame",
		"-b:a", "192k",
		"-y",
		outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg mix failed: %s", string(output))
	}

	return nil
}

// mixAudioFilesInBatches trộn nhiều file audio theo lô để giảm độ phức tạp khi số input rất lớn
func (s *OptimizedTTSService) mixAudioFilesInBatches(inputFiles []string, outputPath string, tempDir string, batchSize int) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files to mix")
	}

	if batchSize <= 1 {
		batchSize = 30
	}

	if len(inputFiles) <= batchSize {
		return s.mixAudioFiles(inputFiles, outputPath)
	}

	var batchFiles []string
	for i := 0; i < len(inputFiles); i += batchSize {
		end := i + batchSize
		if end > len(inputFiles) {
			end = len(inputFiles)
		}

		chunk := inputFiles[i:end]
		batchFile := filepath.Join(tempDir, fmt.Sprintf("batch_mix_%d.wav", i/batchSize))

		args := []string{}
		for _, file := range chunk {
			args = append(args, "-i", file)
		}

		filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(chunk))
		args = append(args,
			"-filter_complex", filter,
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			batchFile,
		)

		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg batch mix failed: %s", string(output))
		}

		batchFiles = append(batchFiles, batchFile)
	}

	if len(batchFiles) == 1 {
		cmd := exec.Command("ffmpeg",
			"-i", batchFiles[0],
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg final convert failed: %s", string(output))
		}
		return nil
	}

	args := []string{}
	for _, file := range batchFiles {
		args = append(args, "-i", file)
	}

	filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(batchFiles))
	args = append(args,
		"-filter_complex", filter,
		"-map", "[out]",
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "libmp3lame",
		"-b:a", "192k",
		"-y",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg final batch mix failed: %s", string(output))
	}

	return nil
}

// updateSegmentMapping cập nhật mapping cho segment
func (s *OptimizedTTSService) updateSegmentMapping(jobID string, segmentIndex int, updates map[string]interface{}) {
	err := s.mappingService.UpdateSegmentMapping(jobID, segmentIndex, updates)
	if err != nil {
		log.Printf("Failed to update segment mapping: %v", err)
	}
}

// GetProcessingStatus lấy trạng thái xử lý của job
func (s *OptimizedTTSService) GetProcessingStatus(jobID string) map[string]interface{} {
	return s.mappingService.GetJobProgress(jobID)
}

// GetServiceStatistics lấy thống kê của service
func (s *OptimizedTTSService) GetServiceStatistics() map[string]interface{} {
	stats := s.mappingService.GetJobStatistics()

	// Thêm thông tin rate limiter
	if s.rateLimiter != nil {
		rateLimitStats := s.rateLimiter.GetCurrentUsage()
		stats["rate_limiter"] = rateLimitStats
	}

	stats["max_concurrent_workers"] = s.maxConcurrent
	stats["active_workers"] = len(s.workerPool)

	return stats
}

// truncateText helper function để truncate text dài
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}
