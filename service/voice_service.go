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

	// Get the appropriate stem file (Demucs output)
	htdemucsDir := filepath.Join(outputDir, "htdemucs")
	subDirs, err := os.ReadDir(htdemucsDir)
	if err != nil || len(subDirs) == 0 {
		return "", fmt.Errorf("demucs output folder not found: %v", err)
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

// BurnSubtitleWithBackground burns subtitle into video with solid background box
func BurnSubtitleWithBackground(videoPath, srtPath, outputDir string, textColor, bgColor string) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Check if SRT file exists
	if _, err := os.Stat(srtPath); os.IsNotExist(err) {
		return "", fmt.Errorf("srt file not found: %s", srtPath)
	}

	// Check if video file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("video file not found: %s", videoPath)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("burned_%s.mp4", timestamp))

	// Convert hex colors to ASS format
	textColorASS := convertHexToASSColor(textColor)
	bgColorASS := convertHexToASSColor(bgColor)

	// Use absolute path for SRT file to avoid path issues
	absSrtPath, err := filepath.Abs(srtPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for SRT: %v", err)
	}

	// Detect language from SRT content
	srtContentBytes, err := os.ReadFile(absSrtPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file for language detection: %v", err)
	}
	srtContent := string(srtContentBytes)
	lang := DetectSRTLanguage(srtContent)

	// Map language to font
	fontMap := map[string]string{
		"zh": "Noto Sans CJK SC",
		"ja": "Noto Sans CJK JP",
		"ko": "Noto Sans CJK KR",
		"vi": "Arial",
		"en": "Arial",
		"fr": "DejaVu Sans",
		"de": "DejaVu Sans",
		"es": "DejaVu Sans",
	}
	fontName, ok := fontMap[lang]
	if !ok {
		fontName = "Arial Unicode MS" // fallback
	}

	log.Printf("Burning subtitle: video=%s, srt=%s, textColor=%s, bgColor=%s, lang=%s, font=%s", videoPath, absSrtPath, textColor, bgColor, lang, fontName)

	// FFmpeg command to burn subtitle with solid background box
	// Use absolute path and escape special characters
	escapedSrtPath := strings.ReplaceAll(absSrtPath, "'", "\\'")
	forceStyle := fmt.Sprintf("Fontname=%s,Fontsize=24,PrimaryColour=%s,BackColour=%s,Outline=2,Shadow=0,BorderStyle=3", fontName, textColorASS, bgColorASS)
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("subtitles='%s':force_style='%s'", escapedSrtPath, forceStyle),
		"-c:a", "copy", // Copy audio without re-encoding
		"-y", // Overwrite output file
		outputPath,
	)

	// Capture command output for better error handling
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg burn subtitle error: %s", string(output))
		log.Printf("FFmpeg command: %s", strings.Join(cmd.Args, " "))
		return "", fmt.Errorf("failed to burn subtitle: %v, output: %s", err, string(output))
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("output file was not created: %s", outputPath)
	}

	log.Printf("Successfully burned subtitle to: %s", outputPath)
	return outputPath, nil
}

// BurnSubtitleWithASS burns subtitle using ASS format (alternative method)
func BurnSubtitleWithASS(videoPath, srtPath, outputDir string, textColor, bgColor string) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("burned_ass_%s.mp4", timestamp))

	// Convert SRT to ASS format
	assPath := strings.Replace(srtPath, ".srt", ".ass", 1)
	if err := convertSRTtoASS(srtPath, assPath, textColor, bgColor); err != nil {
		return "", fmt.Errorf("failed to convert SRT to ASS: %v", err)
	}

	// Use absolute path for ASS file
	absAssPath, err := filepath.Abs(assPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for ASS: %v", err)
	}

	log.Printf("Burning subtitle with ASS: video=%s, ass=%s", videoPath, absAssPath)

	// FFmpeg command using ASS format
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("ass=%s", absAssPath),
		"-c:a", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg ASS burn subtitle error: %s", string(output))
		return "", fmt.Errorf("failed to burn subtitle with ASS: %v, output: %s", err, string(output))
	}

	// Clean up temporary ASS file
	os.Remove(assPath)

	return outputPath, nil
}

// BurnSubtitleWithBackgroundFont giống BurnSubtitleWithBackground nhưng cho phép truyền fontName tuỳ ý
func BurnSubtitleWithBackgroundFont(videoPath, srtPath, outputDir string, textColor, bgColor, fontName string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}
	if _, err := os.Stat(srtPath); os.IsNotExist(err) {
		return "", fmt.Errorf("srt file not found: %s", srtPath)
	}
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("video file not found: %s", videoPath)
	}
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("burned_%s.mp4", timestamp))
	textColorASS := convertHexToASSColor(textColor)
	bgColorASS := convertHexToASSColor(bgColor)
	absSrtPath, err := filepath.Abs(srtPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for SRT: %v", err)
	}
	log.Printf("Burning subtitle: video=%s, srt=%s, textColor=%s, bgColor=%s, font=%s", videoPath, absSrtPath, textColor, bgColor, fontName)
	escapedSrtPath := strings.ReplaceAll(absSrtPath, "'", "\\'")
	forceStyle := fmt.Sprintf("Fontname=%s,Fontsize=24,PrimaryColour=%s,BackColour=%s,Outline=2,Shadow=0,BorderStyle=3", fontName, textColorASS, bgColorASS)
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("subtitles='%s':force_style='%s'", escapedSrtPath, forceStyle),
		"-c:a", "copy",
		"-y",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg burn subtitle error: %s", string(output))
		log.Printf("FFmpeg command: %s", strings.Join(cmd.Args, " "))
		return "", fmt.Errorf("failed to burn subtitle: %v, output: %s", err, string(output))
	}
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("output file was not created: %s", outputPath)
	}
	log.Printf("Successfully burned subtitle to: %s", outputPath)
	return outputPath, nil
}

// convertHexToASSColor converts hex color to ASS format
func convertHexToASSColor(hex string) string {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	// Ensure hex is 6 characters
	if len(hex) != 6 {
		return "&H000000" // Default black
	}

	// Convert hex to ASS format: &H00BBGGRR
	// hex: RRGGBB
	bb := hex[4:6] // Blue
	gg := hex[2:4] // Green
	rr := hex[0:2] // Red
	return fmt.Sprintf("&H00%s%s%s", bb, gg, rr)
}

// convertSRTtoASS converts SRT subtitle to ASS format with custom colors
func convertSRTtoASS(srtPath, assPath, textColor, bgColor string) error {
	// Read SRT file
	srtContent, err := os.ReadFile(srtPath)
	if err != nil {
		return err
	}

	// Convert colors to ASS format
	textColorASS := convertHexToASSColor(textColor)
	bgColorASS := convertHexToASSColor(bgColor)

	// Create ASS header with styling
	assHeader := fmt.Sprintf(`[Script Info]
Title: Converted from SRT
ScriptType: v4.00+
WrapStyle: 1
ScaledBorderAndShadow: yes
YCbCr Matrix: TV.601

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,24,%s,%s,%s,%s,0,0,0,0,100,100,0,0,3,2,0,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`, textColorASS, textColorASS, textColorASS, bgColorASS)

	// Parse SRT and convert to ASS
	lines := strings.Split(string(srtContent), "\n")
	var assEvents strings.Builder
	assEvents.WriteString(assHeader)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "1" || line == "2" || line == "3" {
			continue // Skip SRT index numbers
		}

		// Check if this is a timestamp line
		if strings.Contains(line, "-->") {
			// Parse timestamp
			parts := strings.Split(line, " --> ")
			if len(parts) == 2 {
				startTime := convertTimeToASS(parts[0])
				endTime := convertTimeToASS(parts[1])

				// Get subtitle text from next line
				i++
				if i < len(lines) {
					text := strings.TrimSpace(lines[i])
					if text != "" {
						// Create ASS event
						assEvent := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startTime, endTime, text)
						assEvents.WriteString(assEvent)
					}
				}
			}
		}
	}

	// Write ASS file
	return os.WriteFile(assPath, []byte(assEvents.String()), 0644)
}

// convertTimeToASS converts SRT time format to ASS time format
func convertTimeToASS(srtTime string) string {
	// SRT format: 00:00:01,000
	// ASS format: 0:00:01.00
	srtTime = strings.TrimSpace(srtTime)
	srtTime = strings.Replace(srtTime, ",", ".", 1)

	// Remove leading zeros from hours
	if strings.HasPrefix(srtTime, "00:") {
		srtTime = "0:" + srtTime[3:]
	}

	return srtTime
}
