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
	"time"
)

// DemucsConfig cấu hình cho Demucs
type DemucsConfig struct {
	DemucsPath string
	ModelName  string
	Stems      string
	OutputDir  string
}

// GetDemucsPath tìm đường dẫn Demucs
func GetDemucsPath() string {
	// Danh sách các đường dẫn có thể có
	possiblePaths := []string{
		"/Library/Frameworks/Python.framework/Versions/3.11/bin/demucs",
		"/Users/phantuan/Library/Frameworks/Python.framework/Versions/3.11/bin/demucs",
		"/Users/phantuan/Library/Python/3.9/bin/demucs",
		"/usr/local/bin/demucs",
		"demucs", // Sử dụng PATH
	}

	for _, path := range possiblePaths {
		if path == "demucs" {
			// Kiểm tra trong PATH
			if cmd, err := exec.LookPath("demucs"); err == nil {
				return cmd
			}
		} else {
			// Kiểm tra file tồn tại và có quyền thực thi
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// SeparateAudioAsync tách audio với error handling tốt hơn
func SeparateAudioAsync(audioPath string, fileName string, stemType string, videoDir string) (string, error) {
	log.Printf("Starting to separate %s from: %s", audioPath, stemType)

	// Kiểm tra file audio tồn tại
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file not found: %s", audioPath)
	}

	// Tìm đường dẫn Demucs
	demucsPath := GetDemucsPath()
	if demucsPath == "" {
		return "", fmt.Errorf("demucs not found. Please install demucs: pip3 install -U demucs")
	}

	log.Printf("Using Demucs at: %s", demucsPath)

	// Create output directory for separated audio
	outputDir := filepath.Join(videoDir, "separated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create separated audio directory: %v", err)
	}

	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// Đảm bảo tên file separated là duy nhất (thêm timestamp)
	timestamp := time.Now().UnixNano()
	uniquePrefix := fmt.Sprintf("%d_%s", timestamp, fileNameWithoutExt)

	// Get audio duration using ffprobe
	durationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath)
	durationOutput, err := durationCmd.Output()
	if err != nil {
		log.Printf("Error getting audio duration: %v", err)
	} else {
		log.Printf("Input audio duration: %s seconds", string(durationOutput))
	}

	// Kiểm tra pretrained models
	modelsDir := "pretrained_models/2stems"
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		log.Printf("Warning: Pretrained models directory not found: %s", modelsDir)
		log.Printf("Demucs will download models automatically on first run")
	}

	// Use Demucs to separate audio với timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute) // 30 phút timeout
	defer cancel()

	// Set environment variables for Demucs
	cmd := exec.CommandContext(ctx, demucsPath,
		"-n", "htdemucs",
		"--two-stems", "vocals",
		"-o", outputDir,
		audioPath,
	)

	// Set PATH to include Python framework
	cmd.Env = append(os.Environ(), "PATH=/Library/Frameworks/Python.framework/Versions/3.11/bin:"+os.Getenv("PATH"))

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Demucs error output: %s", string(output))

		// Kiểm tra các lỗi cụ thể
		if strings.Contains(string(output), "CUDA") {
			return "", fmt.Errorf("CUDA error. Try running without GPU: demucs --cpu -n htdemucs")
		}
		if strings.Contains(string(output), "model") {
			return "", fmt.Errorf("model not found. Please download pretrained models")
		}
		if strings.Contains(string(output), "permission") {
			return "", fmt.Errorf("permission denied. Check file permissions")
		}

		return "", fmt.Errorf("failed to run demucs: %v, output: %s", err, string(output))
	}

	// Get the appropriate stem file (Demucs output)
	htdemucsDir := filepath.Join(outputDir, "htdemucs")
	subDirs, err := os.ReadDir(htdemucsDir)
	if err != nil || len(subDirs) == 0 {
		return "", fmt.Errorf("Demucs output folder not found: %v", err)
	}

	actualSubDir := subDirs[0].Name()
	stemPath := filepath.Join(htdemucsDir, actualSubDir, stemType+".wav")

	// Kiểm tra file stem tồn tại
	if _, err := os.Stat(stemPath); os.IsNotExist(err) {
		return "", fmt.Errorf("stem file not found: %s", stemPath)
	}

	log.Printf("%s extracted to: %s", stemType, stemPath)

	// Convert WAV to MP3 for better compatibility
	mp3Path := filepath.Join(outputDir, uniquePrefix+"_"+stemType+".mp3")
	ffmpegCmd := exec.Command("ffmpeg",
		"-i", stemPath,
		"-codec:a", "libmp3lame",
		"-qscale:a", "2",
		"-y", // Overwrite output file if exists
		mp3Path,
	)

	// Capture FFmpeg output
	ffmpegOutput, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg conversion error output: %s", string(ffmpegOutput))
		return "", fmt.Errorf("failed to convert %s to MP3: %v", stemType, err)
	}

	// Get output audio duration using ffprobe
	outputDurationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		mp3Path)
	outputDurationOutput, err := outputDurationCmd.Output()
	if err != nil {
		log.Printf("Error getting output audio duration: %v", err)
	} else {
		log.Printf("Output %s duration: %s seconds", stemType, string(outputDurationOutput))
	}

	log.Printf("%s converted to MP3: %s", stemType, mp3Path)

	// Clean up temporary files (giữ lại file MP3)
	// os.RemoveAll(filepath.Join(outputDir, fileNameWithoutExt))

	return mp3Path, nil
}

// ExtractBackgroundMusicAsync tách nhạc nền với error handling tốt hơn
func ExtractBackgroundMusicAsync(audioPath string, fileName string, videoDir string) (string, error) {
	return SeparateAudioAsync(audioPath, fileName, "no_vocals", videoDir)
}

// ExtractVocalsAsync tách giọng nói với error handling tốt hơn
func ExtractVocalsAsync(audioPath string, fileName string, videoDir string) (string, error) {
	return SeparateAudioAsync(audioPath, fileName, "vocals", videoDir)
}

// FallbackSeparateAudio tách audio bằng FFmpeg nếu Demucs thất bại
func FallbackSeparateAudio(audioPath string, fileName string, stemType string, videoDir string) (string, error) {
	log.Printf("Using FFmpeg fallback for %s separation", stemType)

	outputDir := filepath.Join(videoDir, "separated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create separated audio directory: %v", err)
	}

	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	timestamp := time.Now().UnixNano()
	uniquePrefix := fmt.Sprintf("%d_%s", timestamp, fileNameWithoutExt)

	// Sử dụng FFmpeg để tách audio (đơn giản hơn Demucs)
	var filter string
	if stemType == "vocals" {
		// Tách giọng nói bằng high-pass filter
		filter = "highpass=f=200,lowpass=f=3000"
	} else {
		// Tách nhạc nền bằng low-pass filter
		filter = "lowpass=f=200,highpass=f=3000"
	}

	mp3Path := filepath.Join(outputDir, uniquePrefix+"_"+stemType+".mp3")
	cmd := exec.Command("ffmpeg",
		"-i", audioPath,
		"-af", filter,
		"-codec:a", "libmp3lame",
		"-qscale:a", "2",
		"-y",
		mp3Path,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg fallback error: %s", string(output))
		return "", fmt.Errorf("ffmpeg fallback failed: %v", err)
	}

	log.Printf("FFmpeg fallback successful: %s", mp3Path)
	return mp3Path, nil
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
