package service

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExtractBackgroundMusicAsync thêm job tách nhạc nền vào queue
func ExtractBackgroundMusicAsync(audioPath string, fileName string, videoDir string, userID uint, videoID uint) (string, error) {
	// Tạo job ID duy nhất
	jobID := uuid.New().String()

	// Tính toán priority dựa trên thời gian chờ
	priority := 5 // Priority mặc định

	// Tính toán max duration (2x thời gian audio + buffer)
	duration, err := GetAudioDuration(audioPath)
	if err != nil {
		duration = 300 // Default 5 phút nếu không lấy được duration
	}
	maxDuration := duration*2 + 60 // 2x duration + 1 phút buffer

	// Tạo job
	job := &AudioProcessingJob{
		ID:          jobID,
		AudioPath:   audioPath,
		FileName:    fileName,
		VideoDir:    videoDir,
		StemType:    "no_vocals",
		UserID:      userID,
		VideoID:     videoID,
		Priority:    priority,
		MaxDuration: maxDuration,
	}

	// Thêm vào queue
	queueService := GetQueueService()
	if queueService == nil {
		return "", fmt.Errorf("queue service not initialized")
	}

	err = queueService.EnqueueJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %v", err)
	}

	// Cập nhật trạng thái
	queueService.UpdateJobStatus(jobID, "queued")

	log.Printf("Background music extraction job queued: %s", jobID)
	return jobID, nil
}

// ExtractVocalsAsync thêm job tách giọng nói vào queue
func ExtractVocalsAsync(audioPath string, fileName string, videoDir string, userID uint, videoID uint) (string, error) {
	// Tạo job ID duy nhất
	jobID := uuid.New().String()

	// Tính toán priority dựa trên thời gian chờ
	priority := 5 // Priority mặc định

	// Tính toán max duration (2x thời gian audio + buffer)
	duration, err := GetAudioDuration(audioPath)
	if err != nil {
		duration = 300 // Default 5 phút nếu không lấy được duration
	}
	maxDuration := duration*2 + 60 // 2x duration + 1 phút buffer

	// Tạo job
	job := &AudioProcessingJob{
		ID:          jobID,
		AudioPath:   audioPath,
		FileName:    fileName,
		VideoDir:    videoDir,
		StemType:    "vocals",
		UserID:      userID,
		VideoID:     videoID,
		Priority:    priority,
		MaxDuration: maxDuration,
	}

	// Thêm vào queue
	queueService := GetQueueService()
	if queueService == nil {
		return "", fmt.Errorf("queue service not initialized")
	}

	err = queueService.EnqueueJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %v", err)
	}

	// Cập nhật trạng thái
	queueService.UpdateJobStatus(jobID, "queued")

	log.Printf("Vocals extraction job queued: %s", jobID)
	return jobID, nil
}

// GetJobStatus trả về trạng thái của job
func GetJobStatus(jobID string) (string, error) {
	queueService := GetQueueService()
	if queueService == nil {
		return "", fmt.Errorf("queue service not initialized")
	}

	return queueService.GetJobStatus(jobID)
}

// GetJobResult trả về kết quả của job
func GetJobResult(jobID string) (string, error) {
	queueService := GetQueueService()
	if queueService == nil {
		return "", fmt.Errorf("queue service not initialized")
	}

	return queueService.GetJobResult(jobID)
}

// WaitForJobCompletion chờ job hoàn thành và trả về kết quả
func WaitForJobCompletion(jobID string, timeout time.Duration) (string, error) {
	queueService := GetQueueService()
	if queueService == nil {
		return "", fmt.Errorf("queue service not initialized")
	}

	startTime := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Kiểm tra timeout
			if time.Since(startTime) > timeout {
				return "", fmt.Errorf("job timeout after %v", timeout)
			}

			// Kiểm tra trạng thái
			status, err := queueService.GetJobStatus(jobID)
			if err != nil {
				return "", fmt.Errorf("failed to get job status: %v", err)
			}

			switch status {
			case "completed":
				// Lấy kết quả
				result, err := queueService.GetJobResult(jobID)
				if err != nil {
					return "", fmt.Errorf("failed to get job result: %v", err)
				}
				return result, nil

			case "failed":
				return "", fmt.Errorf("job failed")

			case "not_found":
				return "", fmt.Errorf("job not found")

			case "queued", "processing":
				// Tiếp tục chờ
				continue

			default:
				log.Printf("Unknown job status: %s", status)
				continue
			}
		}
	}
}

// GetQueueStatus trả về trạng thái của queue
func GetQueueStatus() (map[string]int64, error) {
	queueService := GetQueueService()
	if queueService == nil {
		return nil, fmt.Errorf("queue service not initialized")
	}

	return queueService.GetQueueStatus()
}

// GetWorkerStatus trả về trạng thái của worker service
func GetWorkerStatus() map[string]interface{} {
	workerService := GetWorkerService()
	if workerService == nil {
		return map[string]interface{}{
			"error": "worker service not initialized",
		}
	}

	return workerService.GetStatus()
}

// Helper function để lấy audio duration
func GetAudioDuration(filePath string) (float64, error) {
	// Sử dụng ffprobe để lấy duration
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %v", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}
