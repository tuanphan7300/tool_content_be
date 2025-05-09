package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ProcessVideoToAudio(videoPath string) (string, error) {
	log.Printf("Starting to process video: %s", videoPath)

	// Create output directory if it doesn't exist
	outputDir := "./storage/audio"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename
	baseName := filepath.Base(videoPath)
	fileNameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	outputPath := filepath.Join(outputDir, fileNameWithoutExt+".mp3")

	log.Printf("Will save audio to: %s", outputPath)

	// Use FFmpeg to extract audio
	cmd := exec.Command("ffmpeg",
		"-i", videoPath, // Input file
		"-vn",                   // No video
		"-acodec", "libmp3lame", // Use MP3 codec
		"-q:a", "2", // High quality audio
		outputPath, // Output file
	)

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg error output: %s", string(output))
		return "", fmt.Errorf("failed to process video: %v", err)
	}

	log.Printf("Audio extracted successfully")
	return outputPath, nil
}

func separateAudio(audioPath string, fileName string, stemType string) (string, error) {
	log.Printf("Starting to separate %s from: %s", stemType, audioPath)

	// Create output directory for separated audio
	outputDir := "./storage/separated"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create separated audio directory: %v", err)
	}

	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Use Spleeter to separate audio
	spleeterPath := "/Users/phantuan/Library/Python/3.9/bin/spleeter"
	cmd := exec.Command(spleeterPath,
		"separate",
		"-p", "spleeter:2stems", // Separate into vocals and accompaniment
		"-o", outputDir,
		audioPath,
	)

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Spleeter error output: %s", string(output))
		return "", fmt.Errorf("failed to run spleeter: %v", err)
	}

	log.Printf("Spleeter output: %s", string(output))

	// Get the appropriate stem file
	stemPath := filepath.Join(outputDir, fileNameWithoutExt, stemType+".wav")
	log.Printf("%s extracted to: %s", stemType, stemPath)

	// Convert WAV to MP3 for better compatibility
	mp3Path := filepath.Join(outputDir, fileNameWithoutExt+"_"+stemType+".mp3")
	ffmpegCmd := exec.Command("ffmpeg",
		"-i", stemPath,
		"-codec:a", "libmp3lame",
		"-qscale:a", "2",
		mp3Path,
	)

	// Capture FFmpeg output
	ffmpegOutput, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg conversion error output: %s", string(ffmpegOutput))
		return "", fmt.Errorf("failed to convert %s to MP3: %v", stemType, err)
	}

	log.Printf("%s converted to MP3: %s", stemType, mp3Path)

	// Clean up temporary files
	os.RemoveAll(filepath.Join(outputDir, fileNameWithoutExt))

	return mp3Path, nil
}

func ExtractBackgroundMusic(audioPath string, fileName string) (string, error) {
	return separateAudio(audioPath, fileName, "accompaniment")
}

func ExtractVocals(audioPath string, fileName string) (string, error) {
	return separateAudio(audioPath, fileName, "vocals")
}
