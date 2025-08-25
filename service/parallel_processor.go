package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProcessingTask đại diện cho một tác vụ xử lý
type ProcessingTask struct {
	ID       string
	Type     string
	Status   string // "pending", "running", "completed", "failed"
	Result   interface{}
	Error    error
	Progress float64 // 0-100
}

// ParallelProcessor xử lý song song các tác vụ
type ParallelProcessor struct {
	tasks  map[string]*ProcessingTask
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewParallelProcessor tạo processor mới
func NewParallelProcessor() *ParallelProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ParallelProcessor{
		tasks:  make(map[string]*ProcessingTask),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddTask thêm tác vụ mới
func (p *ParallelProcessor) AddTask(id, taskType string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tasks[id] = &ProcessingTask{
		ID:       id,
		Type:     taskType,
		Status:   "pending",
		Progress: 0,
	}
}

// UpdateTaskProgress cập nhật tiến độ tác vụ
func (p *ParallelProcessor) UpdateTaskProgress(id string, progress float64, status string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if task, exists := p.tasks[id]; exists {
		task.Progress = progress
		task.Status = status
	}
}

// GetTaskStatus lấy trạng thái tác vụ
func (p *ParallelProcessor) GetTaskStatus(id string) (*ProcessingTask, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	task, exists := p.tasks[id]
	return task, exists
}

// GetAllTasks lấy tất cả tác vụ
func (p *ParallelProcessor) GetAllTasks() map[string]*ProcessingTask {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]*ProcessingTask)
	for id, task := range p.tasks {
		result[id] = task
	}
	return result
}

// ProcessVideoParallel xử lý video song song
type ProcessVideoParallel struct {
	VideoPath        string
	AudioPath        string
	VideoDir         string
	TargetLanguage   string
	SubtitleColor    string
	SubtitleBgColor  string
	BackgroundVolume float64
	TTSVolume        float64
	SpeakingRate     float64
	VoiceName        string // Thêm trường chọn giọng đọc
	HasCustomSrt     bool
	CustomSrtPath    string
	Processor        *ParallelProcessor
	APIKey           string
	GeminiKey        string
	CacheService     *CacheService
	PricingService   *PricingService
}

// NewProcessVideoParallel tạo processor mới
func NewProcessVideoParallel(videoPath, audioPath, videoDir, targetLanguage, apiKey, geminiKey string) *ProcessVideoParallel {
	return &ProcessVideoParallel{
		VideoPath:        videoPath,
		AudioPath:        audioPath,
		VideoDir:         videoDir,
		TargetLanguage:   targetLanguage,
		SubtitleColor:    "#FFFFFF",
		SubtitleBgColor:  "#808080",
		BackgroundVolume: 1.2,
		TTSVolume:        1.5,
		SpeakingRate:     1.2,
		VoiceName:        "", // Sẽ được set sau từ job
		Processor:        NewParallelProcessor(),
		APIKey:           apiKey,
		GeminiKey:        geminiKey,
		CacheService:     NewCacheService(),
		PricingService:   NewPricingService(),
	}
}

// ProcessParallel xử lý song song
func (p *ProcessVideoParallel) ProcessParallel() (*ProcessVideoResult, error) {
	log.Printf("🚀 [PARALLEL PROCESSING] Bắt đầu parallel video processing...")
	startTime := time.Now()

	// Khởi tạo các tác vụ
	p.Processor.AddTask("whisper", "speech_to_text")
	p.Processor.AddTask("background", "audio_separation")

	log.Printf("🔧 [PARALLEL PROCESSING] Đã khởi tạo tasks: whisper + background extraction")

	// Bước 1: Xử lý song song Whisper và Background extraction
	var wg sync.WaitGroup
	var whisperResult *WhisperResult
	var backgroundResult *BackgroundResult
	var whisperErr, backgroundErr error

	log.Printf("⚡ [PARALLEL PROCESSING] Khởi động 2 goroutines chạy song song...")

	// Whisper processing
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🎤 [PARALLEL-WHISPER] Worker bắt đầu xử lý Whisper...")
		p.Processor.UpdateTaskProgress("whisper", 10, "running")
		whisperResult, whisperErr = p.processWhisper()
		if whisperErr != nil {
			log.Printf("❌ [PARALLEL-WHISPER] Whisper failed: %v", whisperErr)
			p.Processor.UpdateTaskProgress("whisper", 0, "failed")
		} else {
			log.Printf("✅ [PARALLEL-WHISPER] Whisper completed successfully")
			p.Processor.UpdateTaskProgress("whisper", 100, "completed")
		}
	}()

	// Background extraction
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🎵 [PARALLEL-BACKGROUND] Worker bắt đầu xử lý background extraction...")
		p.Processor.UpdateTaskProgress("background", 10, "running")
		backgroundResult, backgroundErr = p.processBackground()
		if backgroundErr != nil {
			log.Printf("❌ [PARALLEL-BACKGROUND] Background extraction failed: %v", backgroundErr)
			p.Processor.UpdateTaskProgress("background", 0, "failed")
		} else {
			log.Printf("✅ [PARALLEL-BACKGROUND] Background extraction completed successfully")
			p.Processor.UpdateTaskProgress("background", 100, "completed")
		}
	}()

	log.Printf("⏳ [PARALLEL PROCESSING] Đang chờ 2 goroutines hoàn thành...")
	wg.Wait()
	log.Printf("🎯 [PARALLEL PROCESSING] Cả 2 goroutines đã hoàn thành!")

	// Kiểm tra lỗi
	if whisperErr != nil {
		return nil, fmt.Errorf("whisper processing failed: %v", whisperErr)
	}
	if backgroundErr != nil {
		log.Printf("⚠️ [PARALLEL PROCESSING] Background extraction failed, sử dụng fallback: %v", backgroundErr)
		// Sử dụng fallback
		backgroundResult = &BackgroundResult{
			Path: p.AudioPath, // Sử dụng audio gốc
		}
	}

	log.Printf("🔤 [PARALLEL PROCESSING] Bước 2: Bắt đầu translation (phụ thuộc vào Whisper)...")
	// Bước 2: Translation (phụ thuộc vào Whisper)
	p.Processor.AddTask("translation", "srt_translation")
	p.Processor.UpdateTaskProgress("translation", 10, "running")

	translationResult, err := p.processTranslation(whisperResult)
	if err != nil {
		p.Processor.UpdateTaskProgress("translation", 0, "failed")
		return nil, fmt.Errorf("Lỗi dịch thuật: %v", err)
	}
	p.Processor.UpdateTaskProgress("translation", 100, "completed")
	log.Printf("✅ [PARALLEL PROCESSING] Translation completed successfully")

	log.Printf("🎙️ [PARALLEL PROCESSING] Bước 3: Bắt đầu TTS processing...")
	// Bước 3: TTS
	p.Processor.AddTask("tts", "text_to_speech")
	p.Processor.UpdateTaskProgress("tts", 10, "running")

	ttsResult, err := p.processTTS(translationResult)
	if err != nil {
		p.Processor.UpdateTaskProgress("tts", 0, "failed")
		return nil, fmt.Errorf("Lỗi TTS: %v", err)
	}
	p.Processor.UpdateTaskProgress("tts", 100, "completed")
	log.Printf("✅ [PARALLEL PROCESSING] TTS completed successfully")

	log.Printf("🎬 [PARALLEL PROCESSING] Bước 4: Bắt đầu video processing...")
	// Bước 4: Video processing
	p.Processor.AddTask("video", "video_processing")
	p.Processor.UpdateTaskProgress("video", 10, "running")

	videoResult, err := p.processVideo(ttsResult, backgroundResult, translationResult)
	if err != nil {
		p.Processor.UpdateTaskProgress("video", 0, "failed")
		return nil, fmt.Errorf("Lỗi video processing: %v", err)
	}
	p.Processor.UpdateTaskProgress("video", 100, "completed")
	log.Printf("✅ [PARALLEL PROCESSING] Video processing completed successfully")

	processingTime := time.Since(startTime)
	log.Printf("🏁 [PARALLEL PROCESSING] Tất cả parallel processing hoàn thành trong %v", processingTime)

	// Set thông tin bổ sung
	videoResult.OriginalSRTPath = whisperResult.SRTPath
	videoResult.Transcript = whisperResult.Transcript
	videoResult.Segments = whisperResult.Segments
	videoResult.ProcessingTime = processingTime

	return videoResult, nil
}

// WhisperResult kết quả từ Whisper
type WhisperResult struct {
	Transcript string
	Segments   []Segment
	SRTPath    string
}

// BackgroundResult kết quả từ background extraction
type BackgroundResult struct {
	Path string
}

// TranslationResult kết quả từ translation
type TranslationResult struct {
	TranslatedSRTPath string
	TranslatedContent string
}

// TTSResult kết quả từ TTS
type TTSResult struct {
	TTSPath string
}

// ProcessVideoResult kết quả cuối cùng
type ProcessVideoResult struct {
	FinalVideoPath    string
	BackgroundPath    string
	TTSPath           string
	OriginalSRTPath   string
	TranslatedSRTPath string
	Transcript        string
	Segments          []Segment
	ProcessingTime    time.Duration
}

// processWhisper xử lý Whisper
func (p *ProcessVideoParallel) processWhisper() (*WhisperResult, error) {
	log.Printf("Processing Whisper...")

	var transcript string
	var segments []Segment

	if p.HasCustomSrt {
		// Sử dụng custom SRT - parse file SRT để lấy segments và transcript
		// Parse SRT file để lấy segments
		parsedSegments, err := ParseSRTToSegments(p.CustomSrtPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse custom SRT file: %v", err)
		}

		// Tạo transcript từ segments
		var transcriptLines []string
		for _, segment := range parsedSegments {
			transcriptLines = append(transcriptLines, segment.Text)
		}
		transcript = strings.Join(transcriptLines, " ")
		segments = parsedSegments
	} else {
		// Kiểm tra cache trước
		if cachedResult, err := p.CacheService.GetCachedWhisperResult(p.AudioPath); err == nil {
			log.Printf("Using cached Whisper result")
			return cachedResult, nil
		}

		// Sử dụng Whisper
		var err error
		transcript, segments, _, err = TranscribeWhisperOpenAI(p.AudioPath, p.APIKey)
		if err != nil {
			return nil, err
		}
	}

	// Tạo SRT file
	srtPath := filepath.Join(p.VideoDir, "original.srt")
	srtContent := createSRT(segments)
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		return nil, err
	}

	result := &WhisperResult{
		Transcript: transcript,
		Segments:   segments,
		SRTPath:    srtPath,
	}

	// Cache kết quả
	if !p.HasCustomSrt {
		p.CacheService.CacheWhisperResult(p.AudioPath, result)
	}

	return result, nil
}

// processBackground xử lý background extraction
func (p *ProcessVideoParallel) processBackground() (*BackgroundResult, error) {
	log.Printf("Processing background extraction...")

	// Kiểm tra cache trước
	if cachedPath, err := p.CacheService.GetCachedBackgroundResult(p.AudioPath); err == nil {
		log.Printf("Using cached background result")
		return &BackgroundResult{Path: cachedPath}, nil
	}

	// Sử dụng optimized background extractor
	extractor := NewOptimizedBackgroundExtractor(p.AudioPath, p.VideoDir)
	backgroundPath, err := extractor.ExtractWithFallback()
	if err != nil {
		return nil, err
	}

	// Cache kết quả
	p.CacheService.CacheBackgroundResult(p.AudioPath, backgroundPath)

	return &BackgroundResult{
		Path: backgroundPath,
	}, nil
}

// processTranslation xử lý translation
func (p *ProcessVideoParallel) processTranslation(whisperResult *WhisperResult) (*TranslationResult, error) {
	log.Printf("Processing translation...")

	if p.HasCustomSrt {
		// Sử dụng custom SRT
		return &TranslationResult{
			TranslatedSRTPath: p.CustomSrtPath,
			TranslatedContent: "", // Sẽ đọc từ file
		}, nil
	}

	// Lấy service_name và model_api_name cho nghiệp vụ dịch SRT từ bảng service_config
	serviceName, srtModelAPIName, err := p.PricingService.GetActiveServiceForType("srt_translation")
	if err != nil {
		return nil, fmt.Errorf("failed to get active SRT translation service: %v", err)
	}

	// Dịch SRT với context-aware translation
	var translatedContent string
	if strings.Contains(serviceName, "gpt") {
		// Use GPT for translation with context awareness
		translatedContent, err = TranslateSRTWithContextAwareness(whisperResult.SRTPath, p.APIKey, srtModelAPIName, p.TargetLanguage)
	} else {
		// Use Gemini for translation with context awareness (default)
		translatedContent, err = TranslateSRTWithContextAwareness(whisperResult.SRTPath, p.GeminiKey, srtModelAPIName, p.TargetLanguage)
	}
	if err != nil {
		return nil, err
	}

	// Lưu file đã dịch
	translatedSRTPath := filepath.Join(p.VideoDir, "translated.srt")
	if err := os.WriteFile(translatedSRTPath, []byte(translatedContent), 0644); err != nil {
		return nil, err
	}

	return &TranslationResult{
		TranslatedSRTPath: translatedSRTPath,
		TranslatedContent: translatedContent,
	}, nil
}

// processTTS xử lý TTS
func (p *ProcessVideoParallel) processTTS(translationResult *TranslationResult) (*TTSResult, error) {
	log.Printf("Processing TTS...")

	// Đọc nội dung SRT đã dịch
	content := translationResult.TranslatedContent
	if content == "" {
		// Đọc từ file
		contentBytes, err := os.ReadFile(translationResult.TranslatedSRTPath)
		if err != nil {
			return nil, err
		}
		content = string(contentBytes)
	}

	// Xác định ngôn ngữ cho TTS
	var ttsLanguage string
	if p.HasCustomSrt {
		// Nếu user upload SRT custom, detect ngôn ngữ từ nội dung SRT
		ttsLanguage = DetectSRTLanguage(content)
		log.Printf("Using detected language for TTS: %s (from custom SRT)", ttsLanguage)
		log.Printf("Note: targetLanguage parameter is ignored when custom SRT is provided. Using detected language: %s", ttsLanguage)
	} else {
		// Nếu không có SRT custom, sử dụng targetLanguage
		ttsLanguage = p.TargetLanguage
		log.Printf("Using target language for TTS: %s", ttsLanguage)
	}

	// Sử dụng Optimized TTS Service thay vì TTS cũ
	ttsPath, err := p.processTTSWithOptimizedService(content, ttsLanguage)
	if err != nil {
		return nil, err
	}

	return &TTSResult{
		TTSPath: ttsPath,
	}, nil
}

// processVideo xử lý video cuối cùng
func (p *ProcessVideoParallel) processVideo(ttsResult *TTSResult, backgroundResult *BackgroundResult, translationResult *TranslationResult) (*ProcessVideoResult, error) {
	log.Printf("Processing final video...")

	// Merge video với audio
	mergedPath, err := MergeVideoWithAudio(p.VideoPath, backgroundResult.Path, ttsResult.TTSPath, p.VideoDir, p.BackgroundVolume, p.TTSVolume)
	if err != nil {
		return nil, err
	}

	// Burn subtitle
	finalPath := mergedPath
	if translationResult.TranslatedSRTPath != "" {
		burnedPath, err := BurnSubtitleWithBackground(mergedPath, translationResult.TranslatedSRTPath, p.VideoDir, p.SubtitleColor, p.SubtitleBgColor)
		if err != nil {
			log.Printf("Subtitle burn failed, using merged video: %v", err)
		} else {
			finalPath = burnedPath
		}
	}

	return &ProcessVideoResult{
		FinalVideoPath:    finalPath,
		BackgroundPath:    backgroundResult.Path,
		TTSPath:           ttsResult.TTSPath,
		OriginalSRTPath:   "", // Sẽ được set sau
		TranslatedSRTPath: translationResult.TranslatedSRTPath,
		Transcript:        "",  // Sẽ được set sau
		Segments:          nil, // Sẽ được set sau
	}, nil
}

// processTTSWithOptimizedService xử lý TTS với Optimized TTS Service
func (p *ProcessVideoParallel) processTTSWithOptimizedService(srtContent, targetLanguage string) (string, error) {
	log.Printf("Processing TTS with Optimized TTS Service...")

	// Khởi tạo Optimized TTS Service
	ttsService, err := InitOptimizedTTSService("", 6)
	if err != nil {
		log.Printf("Failed to initialize Optimized TTS Service, falling back to old TTS: %v", err)
		// Fallback về TTS cũ nếu không thể khởi tạo service mới
		return ConvertSRTToSpeechWithLanguageAndVoice(srtContent, p.VideoDir, p.SpeakingRate, targetLanguage, p.VoiceName)
	}

	// Tạo job ID cho TTS processing
	jobID := fmt.Sprintf("tts_%s_%d", filepath.Base(p.VideoDir), time.Now().UnixNano())

	// Tạo options cho TTS
	options := TTSProcessingOptions{
		TargetLanguage:   targetLanguage,
		ServiceName:      "gpt-4o-mini", // Default service
		SubtitleColor:    p.SubtitleColor,
		SubtitleBgColor:  p.SubtitleBgColor,
		BackgroundVolume: p.BackgroundVolume,
		TTSVolume:        p.TTSVolume,
		SpeakingRate:     p.SpeakingRate,
		MaxConcurrent:    6,
		UserID:           0,           // Không có user ID trong context này
		VoiceName:        p.VoiceName, // Thêm voice selection
	}

	// Xử lý TTS với concurrent processing
	audioPath, err := ttsService.ProcessSRTConcurrent(srtContent, p.VideoDir, options, jobID)
	if err != nil {
		log.Printf("Optimized TTS failed, falling back to old TTS: %v", err)
		// Fallback về TTS cũ nếu service mới thất bại
		return ConvertSRTToSpeechWithLanguageAndVoice(srtContent, p.VideoDir, p.SpeakingRate, targetLanguage, p.VoiceName)
	}

	log.Printf("Optimized TTS completed successfully: %s", audioPath)
	return audioPath, nil
}

// Helper functions
func createSRT(segments []Segment) string {
	var result strings.Builder
	for i, segment := range segments {
		result.WriteString(fmt.Sprintf("%d\n", i+1))
		result.WriteString(fmt.Sprintf("%s --> %s\n", formatTimeForSRT(segment.Start), formatTimeForSRT(segment.End)))
		result.WriteString(segment.Text + "\n\n")
	}
	return result.String()
}

func formatTimeForSRT(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := int(seconds) % 3600 / 60
	secs := int(seconds) % 60
	millisecs := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millisecs)
}
