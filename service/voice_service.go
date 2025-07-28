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

func ProcessVideoToAudio(videoPath string, videoDir string) (string, error) {
	log.Printf("Starting to process video: %s", videoPath)

	// Create output directory if it doesn't exist
	outputDir := filepath.Join(videoDir, "audio")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename (unique)
	baseName := filepath.Base(videoPath)
	fileNameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	// Thêm timestamp vào tên file audio để đảm bảo duy nhất
	timestamp := time.Now().UnixNano()
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%d_%s.mp3", timestamp, fileNameWithoutExt))

	log.Printf("Will save audio to: %s", outputPath)

	// Get video duration using ffprobe
	durationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath)
	durationOutput, err := durationCmd.Output()
	if err != nil {
		log.Printf("Error getting video duration: %v", err)
	} else {
		log.Printf("Video duration: %s seconds", string(durationOutput))
	}

	// Use FFmpeg to extract audio
	cmd := exec.Command("ffmpeg",
		"-i", videoPath, // Input file
		"-vn",                   // No video
		"-acodec", "libmp3lame", // Use MP3 codec
		"-q:a", "2", // High quality audio
		"-y",       // Overwrite output file if exists
		outputPath, // Output file
	)

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg error output: %s", string(output))
		return "", fmt.Errorf("failed to process video: %v", err)
	}

	// Get audio duration using ffprobe
	audioDurationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		outputPath)
	audioDurationOutput, err := audioDurationCmd.Output()
	if err != nil {
		log.Printf("Error getting audio duration: %v", err)
	} else {
		log.Printf("Audio duration: %s seconds", string(audioDurationOutput))
	}

	log.Printf("Audio extracted successfully")
	return outputPath, nil
}

func separateAudio(audioPath string, fileName string, stemType string, videoDir string) (string, error) {
	log.Printf("Starting to separate %s from: %s", audioPath, stemType)

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

	// Tìm đường dẫn Demucs
	demucsPath := GetDemucsPath()
	if demucsPath == "" {
		return "", fmt.Errorf("demucs not found. Please install demucs: pip3 install -U demucs")
	}

	log.Printf("Using Demucs at: %s", demucsPath)

	// Use Demucs to separate audio
	cmd := exec.Command(demucsPath,
		"-n", "htdemucs",
		"--two-stems", "vocals",
		"-o", outputDir,
		audioPath,
	)

	// Capture command output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Demucs error output: %s", string(output))
		return "", fmt.Errorf("failed to run demucs: %v", err)
	}

	log.Printf("Demucs output: %s", string(output))

	// Get the appropriate stem file (Demucs output)
	htdemucsDir := filepath.Join(outputDir, "htdemucs")
	subDirs, err := os.ReadDir(htdemucsDir)
	if err != nil || len(subDirs) == 0 {
		return "", fmt.Errorf("Demucs output folder not found: %v", err)
	}
	actualSubDir := subDirs[0].Name()
	stemPath := filepath.Join(htdemucsDir, actualSubDir, stemType+".wav")
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

	// Clean up temporary files
	os.RemoveAll(filepath.Join(outputDir, fileNameWithoutExt))

	return mp3Path, nil
}

func ExtractBackgroundMusic(audioPath string, fileName string, videoDir string) (string, error) {
	return separateAudio(audioPath, fileName, "no_vocals", videoDir)
}

func ExtractVocals(audioPath string, fileName string, videoDir string) (string, error) {
	return separateAudio(audioPath, fileName, "vocals", videoDir)
}

// MergeVideoWithAudio merges a video with background music and TTS audio
func MergeVideoWithAudio(videoPath, backgroundMusicPath, ttsPath, videoDir string, backgroundVolume, ttsVolume float64) (string, error) {
	log.Printf("MergeVideoWithAudio called with volumes - background: %.2f, tts: %.2f", backgroundVolume, ttsVolume)

	// Create output directory if it doesn't exist
	outputDir := filepath.Join(videoDir, "merged")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("merged_%s.mp4", timestamp))

	// Get video duration first
	videoDurationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath)
	videoDurationOutput, err := videoDurationCmd.Output()
	if err != nil {
		log.Printf("Error getting video duration: %v", err)
	} else {
		log.Printf("Video duration: %s seconds", string(videoDurationOutput))
	}

	// Get TTS duration
	ttsDurationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		ttsPath)
	ttsDurationOutput, err := ttsDurationCmd.Output()
	if err != nil {
		log.Printf("Error getting TTS duration: %v", err)
	} else {
		log.Printf("TTS duration: %s seconds", string(ttsDurationOutput))
	}

	// Create a complex filter để mix audio với volume tuỳ chỉnh và đảm bảo đồng bộ
	// Sử dụng apad để đảm bảo background music có đủ độ dài
	// Thêm normalize=0 để tránh amix tự động normalize volume
	filterComplex := fmt.Sprintf(
		"[1:a]volume=%.2f,apad=whole_dur=%s[bg];[2:a]volume=%.2f[tts];[bg][tts]amix=inputs=2:duration=longest:normalize=0[audio]",
		backgroundVolume, strings.TrimSpace(string(videoDurationOutput)), ttsVolume,
	)

	log.Printf("FFmpeg filter complex: %s", filterComplex)

	// Merge video with adjusted audio - sử dụng -shortest để đảm bảo đồng bộ
	cmd := exec.Command("ffmpeg",
		"-i", videoPath, // Input video
		"-i", backgroundMusicPath, // Background music
		"-i", ttsPath, // TTS audio
		"-filter_complex", filterComplex, // Apply audio filters
		"-map", "0:v", // Map video stream
		"-map", "[audio]", // Map mixed audio
		"-c:v", "copy", // Copy video codec
		"-c:a", "aac", // Use AAC for audio
		"-b:a", "192k", // Set audio bitrate
		"-shortest",                       // End when shortest input ends - quan trọng cho đồng bộ
		"-avoid_negative_ts", "make_zero", // Tránh timestamp âm
		"-fflags", "+genpts", // Generate presentation timestamps
		"-y", // Overwrite output file
		outputPath,
	)

	// Capture command output for better error handling
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg merge error output: %s", string(output))
		return "", fmt.Errorf("failed to merge video with audio: %v, output: %s", err, string(output))
	}

	log.Printf("FFmpeg merge output: %s", string(output))

	// Verify final video duration
	finalDurationCmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		outputPath)
	finalDurationOutput, err := finalDurationCmd.Output()
	if err != nil {
		log.Printf("Warning: failed to get final video duration: %v", err)
	} else {
		log.Printf("Final merged video duration: %s seconds", string(finalDurationOutput))
	}

	return outputPath, nil
}

// BurnSubtitleWithBackground burns subtitle into video with black background
func BurnSubtitleWithBackground(videoPath, srtPath, outputDir string) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("burned_%s.mp4", timestamp))

	// FFmpeg command to burn subtitle with black background
	// -vf "subtitles=subtitle.srt:force_style='Fontsize=24,PrimaryColour=&Hffffff,OutlineColour=&H000000,BackColour=&H000000,Outline=2,Shadow=1'"
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("subtitles=%s:force_style='Fontsize=24,PrimaryColour=&Hffffff,OutlineColour=&H000000,BackColour=&H000000,Outline=2,Shadow=1'", srtPath),
		"-c:a", "copy", // Copy audio without re-encoding
		"-y", // Overwrite output file
		outputPath,
	)

	// Capture command output for better error handling
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg burn subtitle error: %s", string(output))
		return "", fmt.Errorf("failed to burn subtitle: %v, output: %s", err, string(output))
	}

	log.Printf("FFmpeg burn subtitle output: %s", string(output))
	return outputPath, nil
}


