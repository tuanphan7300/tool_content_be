package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
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

	// Giới hạn concurrent Demucs processes
	maxConcurrent := 2 // Chỉ cho phép 2 Demucs chạy cùng lúc

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
	// Cập nhật trạng thái
	ws.queueService.UpdateJobStatus(job.ID, "processing")

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

	log.Printf("Demucs output: %s", string(output))

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
