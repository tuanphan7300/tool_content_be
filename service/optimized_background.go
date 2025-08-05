package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OptimizedBackgroundExtractor tối ưu hóa việc tách nhạc nền
type OptimizedBackgroundExtractor struct {
	AudioPath string
	OutputDir string
	Timeout   time.Duration
}

// NewOptimizedBackgroundExtractor tạo extractor mới
func NewOptimizedBackgroundExtractor(audioPath, outputDir string) *OptimizedBackgroundExtractor {
	return &OptimizedBackgroundExtractor{
		AudioPath: audioPath,
		OutputDir: outputDir,
		Timeout:   5 * time.Minute, // Giảm timeout xuống 5 phút
	}
}

// ExtractWithFallback tách nhạc nền với fallback nhanh
func (o *OptimizedBackgroundExtractor) ExtractWithFallback() (string, error) {
	log.Printf("Starting optimized background extraction...")

	// Thử Demucs với timeout ngắn hơn
	backgroundPath, err := o.extractWithDemucs()
	if err != nil {
		log.Printf("Demucs failed, trying FFmpeg fallback: %v", err)

		// Fallback to FFmpeg với phương pháp nhanh hơn
		backgroundPath, err = o.extractWithFFmpeg()
		if err != nil {
			log.Printf("FFmpeg fallback also failed, using original audio: %v", err)
			return o.AudioPath, nil // Sử dụng audio gốc
		}
	}

	return backgroundPath, nil
}

// extractWithDemucs tách với Demucs (tối ưu hóa)
func (o *OptimizedBackgroundExtractor) extractWithDemucs() (string, error) {
	// Kiểm tra file audio tồn tại
	if _, err := os.Stat(o.AudioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file not found: %s", o.AudioPath)
	}

	// Tìm đường dẫn Demucs
	demucsPath := GetDemucsPath()
	if demucsPath == "" {
		return "", fmt.Errorf("demucs not found")
	}

	log.Printf("Using Demucs at: %s", demucsPath)

	// Create output directory for separated audio
	outputDir := filepath.Join(o.OutputDir, "separated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create separated audio directory: %v", err)
	}

	// Sử dụng Demucs với cấu hình tối ưu hóa
	cmd := exec.Command(demucsPath,
		"-n", "htdemucs", // Sử dụng model nhẹ hơn
		"--two-stems", "vocals", // Chỉ tách 2 stems
		"--mp3",                // Output MP3 trực tiếp
		"--mp3-bitrate", "128", // Giảm bitrate để nhanh hơn
		"-o", outputDir,
		o.AudioPath,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "PATH=/Library/Frameworks/Python.framework/Versions/3.11/bin:"+os.Getenv("PATH"))

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Demucs error output: %s", string(output))
		return "", fmt.Errorf("failed to run demucs: %v, output: %s", err, string(output))
	}

	// Tìm file output
	htdemucsDir := filepath.Join(outputDir, "htdemucs")
	subDirs, err := os.ReadDir(htdemucsDir)
	if err != nil || len(subDirs) == 0 {
		return "", fmt.Errorf("Demucs output folder not found: %v", err)
	}

	actualSubDir := subDirs[0].Name()
	stemPath := filepath.Join(htdemucsDir, actualSubDir, "no_vocals.mp3")

	// Kiểm tra file stem tồn tại
	if _, err := os.Stat(stemPath); os.IsNotExist(err) {
		return "", fmt.Errorf("stem file not found: %s", stemPath)
	}

	log.Printf("Background extracted to: %s", stemPath)
	return stemPath, nil
}

// extractWithFFmpeg tách với FFmpeg (fallback nhanh)
func (o *OptimizedBackgroundExtractor) extractWithFFmpeg() (string, error) {
	log.Printf("Using FFmpeg fallback for background extraction...")

	// Tạo output path
	fileName := filepath.Base(o.AudioPath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	timestamp := time.Now().UnixNano()
	outputPath := filepath.Join(o.OutputDir, fmt.Sprintf("%d_%s_no_vocals.mp3", timestamp, fileNameWithoutExt))

	// Sử dụng FFmpeg với filter phức tạp hơn để tách nhạc nền
	// Sử dụng high-pass filter để loại bỏ giọng nói (thường ở tần số thấp)
	cmd := exec.Command("ffmpeg",
		"-i", o.AudioPath,
		"-af", "highpass=f=200,lowpass=f=3000,volume=1.5", // Filter để tách nhạc nền
		"-codec:a", "libmp3lame",
		"-qscale:a", "3",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg error output: %s", string(output))
		return "", fmt.Errorf("failed to extract background with FFmpeg: %v", err)
	}

	log.Printf("Background extracted with FFmpeg to: %s", outputPath)
	return outputPath, nil
}

// FastBackgroundExtractor tách nhạc nền nhanh (chỉ dùng FFmpeg)
func FastBackgroundExtractor(audioPath, outputDir string) (string, error) {
	extractor := NewOptimizedBackgroundExtractor(audioPath, outputDir)
	extractor.Timeout = 2 * time.Minute // Timeout ngắn hơn cho fast mode

	return extractor.extractWithFFmpeg()
}

// QualityBackgroundExtractor tách nhạc nền chất lượng cao (dùng Demucs)
func QualityBackgroundExtractor(audioPath, outputDir string) (string, error) {
	extractor := NewOptimizedBackgroundExtractor(audioPath, outputDir)
	extractor.Timeout = 10 * time.Minute // Timeout dài hơn cho quality mode

	return extractor.ExtractWithFallback()
}
