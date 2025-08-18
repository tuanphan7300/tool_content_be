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
		maxConcurrent = 15 // Mặc định 15 workers để match rate limit (sẽ bị override bởi caller)
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

	log.Printf("⚡ [OPTIMIZED TTS] Khởi động %d concurrent workers để xử lý TTS...", len(entries))
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
	log.Printf("🔄 [OPTIMIZED TTS] Bắt đầu concurrent processing với %d workers (pool size: %d)", len(entries), s.maxConcurrent)

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

// processSingleSegment xử lý một segment TTS
func (s *OptimizedTTSService) processSingleSegment(entry SRTEntry, index int, tempDir string, options TTSProcessingOptions, jobID string) *TTSProcessingResult {
	startTime := time.Now()
	result := &TTSProcessingResult{SegmentIndex: index}

	// Gọi Google TTS API
	audioContent, err := s.callGoogleTTS(entry.Text, options)
	if err != nil {
		result.Error = err
		s.updateSegmentMapping(jobID, index, map[string]interface{}{
			"status":          "failed",
			"error":           err.Error(),
			"processing_time": time.Since(startTime),
		})
		return result
	}

	// Lưu MP3 tạm
	mp3Path := filepath.Join(tempDir, fmt.Sprintf("segment_%d.mp3", index))
	if err := os.WriteFile(mp3Path, audioContent, 0644); err != nil {
		result.Error = fmt.Errorf("failed to write MP3: %v", err)
		s.updateSegmentMapping(jobID, index, map[string]interface{}{
			"status":          "failed",
			"error":           err.Error(),
			"processing_time": time.Since(startTime),
		})
		return result
	}

	// Xử lý audio segment (MP3 -> WAV, đo duration, apply volume)
	wavPath, duration, err := s.processAudioSegment(mp3Path, tempDir, index, options)
	if err != nil {
		result.Error = err
		s.updateSegmentMapping(jobID, index, map[string]interface{}{
			"status":          "failed",
			"error":           err.Error(),
			"processing_time": time.Since(startTime),
		})
		return result
	}

	result.AudioPath = wavPath
	result.Duration = duration
	result.ProcessingTime = time.Since(startTime)

	s.updateSegmentMapping(jobID, index, map[string]interface{}{
		"status":          "completed",
		"audio_path":      wavPath,
		"duration":        duration,
		"processing_time": time.Since(startTime),
	})

	log.Printf("Segment %d processed successfully in %v", index, result.ProcessingTime)
	return result
}

// callGoogleTTS gọi Google TTS API
func (s *OptimizedTTSService) callGoogleTTS(text string, options TTSProcessingOptions) ([]byte, error) {
	// Lấy voice settings cho target language
	languageCode, voiceName := getVoiceForLanguage(options.TargetLanguage)

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
		"-threads", "2",
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
			"-threads", "2",
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

	// Mix tất cả delayed files theo batch để đảm bảo hiệu năng, giữ nguyên thời điểm do đã adelay
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
			"-threads", "2",
			"-y",
			outputPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg copy failed: %s", string(output))
		}
		return nil
	}

	const batchSize = 32
	if len(inputFiles) <= batchSize {
		// Mix trực tiếp nếu số lượng vừa phải
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
			"-threads", "2",
			"-y",
			outputPath)

		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg mix failed: %s", string(output))
		}
		return nil
	}

	// Batch mixing: tạo các batch WAV trung gian, sau đó amix các batch -> MP3 cuối
	tempDir := filepath.Dir(outputPath)
	var batchOutputs []string
	for i := 0; i < len(inputFiles); i += batchSize {
		end := i + batchSize
		if end > len(inputFiles) {
			end = len(inputFiles)
		}
		batch := inputFiles[i:end]
		batchOut := filepath.Join(tempDir, fmt.Sprintf("batch_mix_%d.wav", i/batchSize))

		args := []string{"-i"}
		args = append(args, batch[0])
		for _, file := range batch[1:] {
			args = append(args, "-i", file)
		}
		filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(batch))
		args = append(args,
			"-filter_complex", filter,
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-threads", "2",
			"-y",
			batchOut)

		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg batch mix failed: %s", string(output))
		}
		batchOutputs = append(batchOutputs, batchOut)
	}

	args := []string{"-i"}
	args = append(args, batchOutputs[0])
	for _, file := range batchOutputs[1:] {
		args = append(args, "-i", file)
	}
	filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(batchOutputs))
	args = append(args,
		"-filter_complex", filter,
		"-map", "[out]",
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "libmp3lame",
		"-b:a", "192k",
		"-threads", "2",
		"-y",
		outputPath)

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
