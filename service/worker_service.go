package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"creator-tool-backend/config"

	"gorm.io/datatypes"
)

type WorkerService struct {
	queueService  *QueueService
	maxWorkers    int
	maxConcurrent int
	semaphore     chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	isRunning     bool
	mu            sync.Mutex
}

var (
	workerService *WorkerService
	workerMutex   sync.Mutex
)

// InitWorkerService khởi tạo worker service
func InitWorkerService(queueService *QueueService) *WorkerService {
	workerMutex.Lock()
	defer workerMutex.Unlock()

	if workerService != nil {
		return workerService
	}

	// Tính toán số worker tối ưu dựa trên CPU cores
	maxWorkers := runtime.NumCPU()
	if maxWorkers > 4 {
		maxWorkers = 4 // Giới hạn tối đa 4 worker để tránh quá tải
	}

	// Giới hạn concurrent Demucs processes - Tối ưu cho 4 CPU cores
	maxConcurrent := 3 // Tăng lên 3 để tận dụng tốt hơn 4 CPU cores

	ctx, cancel := context.WithCancel(context.Background())

	workerService = &WorkerService{
		queueService:  queueService,
		maxWorkers:    maxWorkers,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		ctx:           ctx,
		cancel:        cancel,
		isRunning:     false,
	}

	log.Printf("Worker service initialized with %d workers, max concurrent: %d", maxWorkers, maxConcurrent)
	return workerService
}

// GetWorkerService trả về instance của worker service
func GetWorkerService() *WorkerService {
	return workerService
}

// Start bắt đầu worker service
func (ws *WorkerService) Start() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.isRunning {
		log.Println("Worker service is already running")
		return
	}

	ws.isRunning = true
	log.Println("Starting worker service...")

	// Khởi động các worker goroutines
	for i := 0; i < ws.maxWorkers; i++ {
		ws.wg.Add(1)
		go ws.worker(i)
	}

	// Khởi động monitor goroutine
	go ws.monitor()
}

// Stop dừng worker service
func (ws *WorkerService) Stop() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.isRunning {
		return
	}

	log.Println("Stopping worker service...")
	ws.isRunning = false
	ws.cancel()
	ws.wg.Wait()
	log.Println("Worker service stopped")
}

// worker xử lý các job từ queue
func (ws *WorkerService) worker(id int) {
	defer ws.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ws.ctx.Done():
			log.Printf("Worker %d stopped", id)
			return
		default:
			// Lấy job từ queue
			job, err := ws.queueService.DequeueJob()
			if err != nil {
				log.Printf("Worker %d: Failed to dequeue job: %v", id, err)
				time.Sleep(time.Second)
				continue
			}

			if job == nil {
				// Không có job nào, sleep một chút
				time.Sleep(time.Second)
				continue
			}

			log.Printf("Worker %d: Processing job %s", id, job.ID)
			ws.processJob(job)
		}
	}
}

// processJob xử lý một job cụ thể
func (ws *WorkerService) processJob(job *AudioProcessingJob) {
	ws.queueService.UpdateJobStatus(job.ID, "processing")

	if job.JobType == "burn-sub" {
		// Xử lý burn subtitle vào video
		resultPath, err := ws.runBurnSubtitle(job)
		if err != nil {
			log.Printf("Job %s: Failed to burn subtitle: %v", job.ID, err)
			ws.queueService.UpdateJobStatus(job.ID, "failed")
			return
		}
		err = ws.queueService.StoreJobResult(job.ID, resultPath)
		if err != nil {
			log.Printf("Job %s: Failed to store result: %v", job.ID, err)
			ws.queueService.UpdateJobStatus(job.ID, "failed")
			return
		}
		ws.queueService.UpdateJobStatus(job.ID, "completed")
		log.Printf("Job %s: Burn subtitle completed successfully", job.ID)
		return
	}

	if job.JobType == "process-video" {
		// Xử lý process video (parallel processing)
		resultPath, err := ws.runProcessVideo(job)
		if err != nil {
			log.Printf("Job %s: Failed to process video: %v", job.ID, err)
			ws.queueService.UpdateJobStatus(job.ID, "failed")
			return
		}
		err = ws.queueService.StoreJobResult(job.ID, resultPath)
		if err != nil {
			log.Printf("Job %s: Failed to store result: %v", job.ID, err)
			ws.queueService.UpdateJobStatus(job.ID, "failed")
			return
		}
		ws.queueService.UpdateJobStatus(job.ID, "completed")
		log.Printf("Job %s: Process video completed successfully", job.ID)
		return
	}

	// Kiểm tra file tồn tại
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		log.Printf("Job %s: Audio file not found: %s", job.ID, job.AudioPath)
		ws.queueService.UpdateJobStatus(job.ID, "failed")
		return
	}

	// Lấy semaphore để giới hạn concurrent processes
	ws.semaphore <- struct{}{}
	defer func() { <-ws.semaphore }()

	// Tạo context với timeout
	ctx, cancel := context.WithTimeout(ws.ctx, time.Duration(job.MaxDuration)*time.Second)
	defer cancel()

	// Xử lý audio với Demucs
	resultPath, err := ws.runDemucs(ctx, job)
	if err != nil {
		log.Printf("Job %s: Failed to process audio: %v", job.ID, err)
		ws.queueService.UpdateJobStatus(job.ID, "failed")
		return
	}

	// Lưu kết quả
	err = ws.queueService.StoreJobResult(job.ID, resultPath)
	if err != nil {
		log.Printf("Job %s: Failed to store result: %v", job.ID, err)
		ws.queueService.UpdateJobStatus(job.ID, "failed")
		return
	}

	// Cập nhật trạng thái thành công
	ws.queueService.UpdateJobStatus(job.ID, "completed")
	log.Printf("Job %s: Completed successfully", job.ID)
}

// runDemucs chạy Demucs để tách audio
func (ws *WorkerService) runDemucs(ctx context.Context, job *AudioProcessingJob) (string, error) {
	log.Printf("Running Demucs for job %s", job.ID)

	// Tạo output directory
	outputDir := filepath.Join(job.VideoDir, "separated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	fileNameWithoutExt := strings.TrimSuffix(job.FileName, filepath.Ext(job.FileName))
	timestamp := time.Now().UnixNano()
	uniquePrefix := fmt.Sprintf("%d_%s", timestamp, fileNameWithoutExt)

	// Tìm đường dẫn Demucs
	demucsPath := GetDemucsPath()
	if demucsPath == "" {
		return "", fmt.Errorf("demucs not found. Please install demucs: pip3 install -U demucs")
	}

	log.Printf("Using Demucs at: %s", demucsPath)

	// Chạy Demucs với context timeout
	cmd := exec.CommandContext(ctx, demucsPath,
		"-n", "htdemucs",
		"--two-stems", "vocals",
		"-o", outputDir,
		job.AudioPath,
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("demucs failed: %v, output: %s", err, string(output))
	}

	// Tìm file kết quả
	htdemucsDir := filepath.Join(outputDir, "htdemucs")
	subDirs, err := os.ReadDir(htdemucsDir)
	if err != nil || len(subDirs) == 0 {
		return "", fmt.Errorf("demucs output folder not found: %v", err)
	}

	actualSubDir := subDirs[0].Name()
	stemPath := filepath.Join(htdemucsDir, actualSubDir, job.StemType+".wav")

	// Convert WAV to MP3
	mp3Path := filepath.Join(outputDir, uniquePrefix+"_"+job.StemType+".mp3")
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", stemPath,
		"-codec:a", "libmp3lame",
		"-qscale:a", "2",
		"-y",
		mp3Path,
	)

	ffmpegOutput, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to MP3: %v, output: %s", err, string(ffmpegOutput))
	}

	// Clean up temporary files
	os.RemoveAll(filepath.Join(outputDir, fileNameWithoutExt))

	return mp3Path, nil
}

func hexToASSColor(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "&H00FFFFFF" // fallback trắng
	}
	// Định dạng ARGB: &HAABBGGRR
	// hex: RRGGBB
	bb := hex[4:6]
	gg := hex[2:4]
	rr := hex[0:2]
	return fmt.Sprintf("&H00%s%s%s", bb, gg, rr)
}

func (ws *WorkerService) runBurnSubtitle(job *AudioProcessingJob) (string, error) {
	videoPath := filepath.Join(job.VideoDir, job.FileName)
	subPath := job.SubtitlePath
	outputDir := filepath.Join(job.VideoDir, "burned")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("burned_%s.mp4", timestamp))

	// Xử lý màu chữ và màu nền với solid background box
	color := hexToASSColor(job.SubtitleColor)
	bgcolor := hexToASSColor(job.SubtitleBgColor)
	forceStyle := fmt.Sprintf("Fontsize=24,PrimaryColour=%s,BackColour=%s,Outline=2,Shadow=0,BorderStyle=3", color, bgcolor)

	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("subtitles='%s':force_style='%s'", subPath, forceStyle),
		"-c:a", "copy",
		"-y",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg burn subtitle error: %s", string(output))
		return "", fmt.Errorf("failed to burn subtitle: %v, output: %s", err, string(output))
	}

	// Lưu lịch sử vào database
	captionHistory := config.CaptionHistory{
		UserID:              job.UserID,
		VideoFilename:       filepath.Join(job.VideoDir, job.FileName),
		VideoFilenameOrigin: job.FileName,
		SrtFile:             job.SubtitlePath,
		MergedVideoFile:     outputPath,
		ProcessType:         "burn-sub",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("Failed to save burn-sub history: %v", err)
	}

	// Cập nhật trạng thái process thành completed
	processService := NewProcessStatusService()
	processService.UpdateProcessStatus(job.ProcessID, "completed")
	processService.UpdateProcessVideoID(job.ProcessID, captionHistory.ID)

	return outputPath, nil
}

// runProcessVideo xử lý video với parallel processing
func (ws *WorkerService) runProcessVideo(job *AudioProcessingJob) (string, error) {
	log.Printf("🚀 [WORKER SERVICE] Bắt đầu xử lý process-video cho job %s", job.ID)
	log.Printf("🔧 [WORKER SERVICE] Job config: user_id=%d, target_language=%s, has_custom_srt=%v",
		job.UserID, job.TargetLanguage, job.HasCustomSrt)

	// Lấy API keys từ config
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	geminiKey := configg.GeminiKey

	log.Printf("⚡ [WORKER SERVICE] Khởi tạo ProcessVideoParallel processor...")
	// Tạo task xử lý video với đầy đủ thông tin
	videoPath := filepath.Join(job.VideoDir, job.FileName)
	task := NewProcessVideoParallel(videoPath, job.AudioPath, job.VideoDir, job.TargetLanguage, apiKey, geminiKey)

	// Cấu hình các thuộc tính bổ sung
	task.HasCustomSrt = job.HasCustomSrt
	task.CustomSrtPath = job.CustomSrtPath
	task.SubtitleColor = job.SubtitleColor
	task.SubtitleBgColor = job.SubtitleBgColor
	task.BackgroundVolume = job.BackgroundVolume
	task.TTSVolume = job.TTSVolume
	task.SpeakingRate = job.SpeakingRate

	log.Printf("🎬 [WORKER SERVICE] Bắt đầu parallel processing với ProcessParallel()...")
	// Xử lý song song
	result, err := task.ProcessParallel()
	if err != nil {
		log.Printf("❌ [WORKER SERVICE] Parallel processing failed: %v", err)
		return "", fmt.Errorf("parallel processing failed: %v", err)
	}

	log.Printf("✅ [WORKER SERVICE] Parallel processing completed successfully!")
	log.Printf("📊 [WORKER SERVICE] Results: srt=%s, tts=%s, video=%s",
		result.TranslatedSRTPath, result.TTSPath, result.FinalVideoPath)

	// Lưu lịch sử vào database
	segmentsJSON, _ := json.Marshal(result.Segments)
	captionHistory := config.CaptionHistory{
		UserID:              job.UserID,
		VideoFilename:       filepath.Join(job.VideoDir, job.FileName),
		VideoFilenameOrigin: job.FileName,
		Transcript:          result.Transcript,
		Segments:            datatypes.JSON(segmentsJSON),
		SegmentsVi:          datatypes.JSON(segmentsJSON), // Sử dụng segments gốc cho segments_vi
		SrtFile:             result.TranslatedSRTPath,
		OriginalSrtFile:     result.OriginalSRTPath,
		TTSFile:             result.TTSPath,
		MergedVideoFile:     result.FinalVideoPath,
		BackgroundMusic:     result.BackgroundPath,
		ProcessType:         "process-video",
		CreatedAt:           time.Now(),
	}
	if err := config.Db.Create(&captionHistory).Error; err != nil {
		log.Printf("⚠️ [WORKER SERVICE] Failed to save process-video history: %v", err)
		return "", fmt.Errorf("failed to save to database: %v", err)
	}

	// Xử lý credit deduction giống như trong ProcessVideoParallelHandler
	creditService := NewCreditService()
	pricingService := NewPricingService()

	// Tính duration để tính chi phí
	duration := getAudioDuration(job.AudioPath)
	durationMinutes := duration / 60.0

	// 1) Whisper (per_minute)
	whisperBase, err := pricingService.CalculateWhisperCost(durationMinutes)
	if err != nil {
		log.Printf("⚠️ [WORKER SERVICE] Failed to calculate Whisper cost: %v", err)
	} else {
		if err := creditService.DeductCredits(job.UserID, whisperBase, "whisper", "Whisper transcribe", &captionHistory.ID, "per_minute", durationMinutes); err != nil {
			log.Printf("⚠️ [WORKER SERVICE] Failed to deduct Whisper credits: %v", err)
		} else {
			log.Printf("✅ [WORKER SERVICE] Deducted %.6f credits for Whisper", whisperBase)
		}
	}

	// 2) Translation (Gemini/GPT) per_token
	serviceName, _, err := pricingService.GetActiveServiceForType("srt_translation")
	if err != nil {
		log.Printf("⚠️ [WORKER SERVICE] Failed to get translation service: %v", err)
	} else {
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
			log.Printf("⚠️ [WORKER SERVICE] Failed to calculate translation cost: %v", err)
		} else {
			translationBase := inCost + outCost
			translationTokens := inTok + outTok
			var translationDesc string
			if strings.Contains(serviceName, "gpt") {
				translationDesc = "GPT dịch SRT"
			} else {
				translationDesc = "Gemini dịch SRT"
			}

			if err := creditService.DeductCredits(job.UserID, translationBase, serviceName, translationDesc, &captionHistory.ID, "per_token", float64(translationTokens)); err != nil {
				log.Printf("⚠️ [WORKER SERVICE] Failed to deduct translation credits: %v", err)
			} else {
				log.Printf("✅ [WORKER SERVICE] Deducted %.6f credits for translation", translationBase)
			}
		}
	}

	// 3) TTS per_character
	if result.Transcript != "" {
		ttsBase, err := pricingService.CalculateTTSCost(result.Transcript, true)
		if err != nil {
			log.Printf("⚠️ [WORKER SERVICE] Failed to calculate TTS cost: %v", err)
		} else {
			if err := creditService.DeductCredits(job.UserID, ttsBase, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len([]rune(result.Transcript)))); err != nil {
				log.Printf("⚠️ [WORKER SERVICE] Failed to deduct TTS credits: %v", err)
			} else {
				log.Printf("✅ [WORKER SERVICE] Deducted %.6f credits for TTS", ttsBase)
			}
		}
	}

	// Cập nhật trạng thái process thành completed
	processService := NewProcessStatusService()
	processService.UpdateProcessStatus(job.ProcessID, "completed")
	processService.UpdateProcessVideoID(job.ProcessID, captionHistory.ID)

	log.Printf("🏁 [WORKER SERVICE] Process-video job %s hoàn thành thành công!", job.ID)
	return result.FinalVideoPath, nil
}

// getAudioDuration trả về duration (giây) của file audio/video
func getAudioDuration(filePath string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️ [WORKER SERVICE] Failed to get audio duration: %v", err)
		return 0
	}
	durStr := strings.TrimSpace(string(output))
	dur, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		log.Printf("⚠️ [WORKER SERVICE] Failed to parse audio duration: %v", err)
		return 0
	}
	return dur
}

// monitor theo dõi trạng thái của worker service
func (ws *WorkerService) monitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case <-ticker.C:
			status, err := ws.queueService.GetQueueStatus()
			if err != nil {
				log.Printf("Failed to get queue status: %v", err)
				continue
			}

			totalJobs := int64(0)
			for priority, count := range status {
				if count > 0 {
					log.Printf("Queue %s: %d jobs", priority, count)
					totalJobs += count
				}
			}

			if totalJobs > 0 {
				log.Printf("Total jobs in queue: %d", totalJobs)
			}
		}
	}
}

// GetStatus trả về trạng thái của worker service
func (ws *WorkerService) GetStatus() map[string]interface{} {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	queueStatus, _ := ws.queueService.GetQueueStatus()

	return map[string]interface{}{
		"is_running":     ws.isRunning,
		"max_workers":    ws.maxWorkers,
		"max_concurrent": ws.maxConcurrent,
		"active_workers": len(ws.semaphore),
		"queue_status":   queueStatus,
	}
}
