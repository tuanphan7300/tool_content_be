package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"log"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
)

type TTSOptions struct {
	Text      string
	VoiceName string
	Speed     float64
	Pitch     float64
}

type SRTEntry struct {
	Index int
	Start float64
	End   float64
	Text  string
}

// TextToSpeech converts text to speech and returns the audio content
func TextToSpeech(text string, options TTSOptions) (string, error) {
	// Create output directory if it doesn't exist
	outputDir := "./storage/tts"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("Hệ thống đang gặp sự cố, vui lòng thử lại sau")
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(outputDir, fmt.Sprintf("tts_%s.mp3", timestamp))

	// Initialize Google TTS client
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		return "", fmt.Errorf("Hệ thống đang gặp sự cố, vui lòng thử lại sau")
	}
	defer client.Close()

	// Set the text input to be synthesized
	input := &texttospeechpb.SynthesisInput{
		InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
	}

	// Build the voice request
	voice := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: "vi-VN",
		Name:         options.VoiceName,
	}

	// Select the type of audio file
	audioConfig := &texttospeechpb.AudioConfig{
		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		SpeakingRate:  options.Speed,
		Pitch:         options.Pitch,
	}

	// Perform the text-to-speech request
	resp, err := client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
		Input:       input,
		Voice:       voice,
		AudioConfig: audioConfig,
	})
	if err != nil {
		return "", fmt.Errorf("Hệ thống đang gặp sự cố, vui lòng thử lại sau")
	}

	// Write the response to the output file
	if err := os.WriteFile(filename, resp.AudioContent, 0644); err != nil {
		return "", fmt.Errorf("Hệ thống đang gặp sự cố, vui lòng thử lại sau")
	}

	return filename, nil
}

// parseSRT parses SRT content and returns a slice of SRTEntry
func parseSRT(srtContent string) ([]SRTEntry, error) {
	var entries []SRTEntry

	// Log first 200 characters for debugging
	preview := srtContent
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	blocks := strings.Split(srtContent, "\n\n")
	for i, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) < 3 {
			log.Printf("Block %d has insufficient lines: %d", i, len(lines))
			continue
		}

		// Parse index
		index, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			log.Printf("Block %d: failed to parse index '%s'", i, lines[0])
			continue
		}

		// Parse timestamp
		timestamp := strings.Split(strings.TrimSpace(lines[1]), " --> ")
		if len(timestamp) != 2 {
			log.Printf("Block %d: invalid timestamp format '%s'", i, lines[1])
			continue
		}

		start, err := parseSRTTime(timestamp[0])
		if err != nil {
			log.Printf("Block %d: failed to parse start time '%s': %v", i, timestamp[0], err)
			continue
		}

		end, err := parseSRTTime(timestamp[1])
		if err != nil {
			log.Printf("Block %d: failed to parse end time '%s': %v", i, timestamp[1], err)
			continue
		}

		// Get text - only lines after timestamp
		var textLines []string
		for j := 2; j < len(lines); j++ {
			line := strings.TrimSpace(lines[j])
			if line != "" {
				textLines = append(textLines, line)
			}
		}
		text := strings.Join(textLines, " ")
		text = strings.TrimSpace(text)

		// Skip if text is empty or contains only numbers/timestamps
		if text == "" || isOnlyNumbersOrTimestamps(text) {
			log.Printf("Block %d: Skipping invalid text: '%s'", i, text)
			continue
		}

		// Debug: Log parsed text
		log.Printf("Block %d: Parsed text: '%s'", i, text)

		entries = append(entries, SRTEntry{
			Index: index,
			Start: start,
			End:   end,
			Text:  text,
		})
	}

	log.Printf("Successfully parsed %d entries", len(entries))
	return entries, nil
}

// parseSRTTime converts SRT time format (HH:MM:SS,mmm) to seconds
func parseSRTTime(timeStr string) (float64, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format")
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	secondsParts := strings.Split(parts[2], ",")
	if len(secondsParts) != 2 {
		return 0, fmt.Errorf("invalid seconds format")
	}

	seconds, err := strconv.Atoi(secondsParts[0])
	if err != nil {
		return 0, err
	}

	milliseconds, err := strconv.Atoi(secondsParts[1])
	if err != nil {
		return 0, err
	}

	return float64(hours*3600+minutes*60+seconds) + float64(milliseconds)/1000, nil
}

// cleanSRTContent cleans SRT content to ensure proper parsing
func cleanSRTContent(srtContent string) string {
	// Normalize line endings
	srtContent = strings.ReplaceAll(srtContent, "\r\n", "\n")
	srtContent = strings.ReplaceAll(srtContent, "\r", "\n")

	// Remove any BOM or special characters
	srtContent = strings.TrimPrefix(srtContent, "\uFEFF") // Remove BOM

	// Ensure proper spacing between blocks
	srtContent = strings.ReplaceAll(srtContent, "\n\n\n", "\n\n")

	return strings.TrimSpace(srtContent)
}

// isOnlyNumbersOrTimestamps checks if text contains only numbers or timestamp patterns
func isOnlyNumbersOrTimestamps(text string) bool {
	// Remove common punctuation and spaces
	cleaned := strings.ReplaceAll(text, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "-->", "")

	// Check if it's only digits
	for _, r := range cleaned {
		if r < '0' || r > '9' {
			return false
		}
	}

	return len(cleaned) > 0
}

// getVoiceForLanguage maps language codes to Google TTS voice settings
// This function ensures that the appropriate voice is used for each target language
// Vietnamese: vi-VN-Wavenet-C (female voice, natural for Vietnamese)
// English: en-US-Wavenet-F (female voice, natural for English)
// Japanese: ja-JP-Wavenet-A (female voice, natural for Japanese)
// Korean: ko-KR-Wavenet-A (female voice, natural for Korean)
// Chinese: cmn-CN-Wavenet-A (female voice, natural for Chinese)
// French: fr-FR-Wavenet-A (female voice, natural for French)
// German: de-DE-Wavenet-A (female voice, natural for German)
// Spanish: es-ES-Wavenet-A (female voice, natural for Spanish)
func getVoiceForLanguage(languageCode string) (string, string) {
	voiceMap := map[string]struct {
		LanguageCode string
		VoiceName    string
	}{
		"vi": {"vi-VN", "vi-VN-Wavenet-C"},
		"en": {"en-US", "en-US-Wavenet-F"},
		"ja": {"ja-JP", "ja-JP-Wavenet-A"},
		"ko": {"ko-KR", "ko-KR-Wavenet-A"},
		"zh": {"cmn-CN", "cmn-CN-Wavenet-A"},
		"fr": {"fr-FR", "fr-FR-Wavenet-A"},
		"de": {"de-DE", "de-DE-Wavenet-A"},
		"es": {"es-ES", "es-ES-Wavenet-A"},
	}

	if voice, exists := voiceMap[languageCode]; exists {
		return voice.LanguageCode, voice.VoiceName
	}

	// Default to Vietnamese
	return "vi-VN", "vi-VN-Wavenet-C"
}

// ConvertSRTToSpeechWithLanguage converts SRT content to speech with specified language
func ConvertSRTToSpeechWithLanguage(srtContent string, videoDir string, speakingRate float64, targetLanguage string) (string, error) {
	// Clean SRT content first
	srtContent = cleanSRTContent(srtContent)

	// Parse SRT content
	entries, err := parseSRT(srtContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse SRT: %v", err)
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no entries found in SRT content")
	}

	// Create output file path
	outputPath := filepath.Join(videoDir, "tts_output.mp3")

	// Initialize Google TTS client
	ctx := context.Background()
	log.Printf("Creating Google TTS client...")
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		log.Printf("Failed to create TTS client: %v", err)
		return "", fmt.Errorf("failed to create TTS client: %v", err)
	}
	defer client.Close()
	log.Printf("Google TTS client created successfully")

	// Get voice settings for target language
	languageCode, voiceName := getVoiceForLanguage(targetLanguage)
	log.Printf("Using voice: %s (%s) for language: %s", voiceName, languageCode, targetLanguage)

	// Create a temporary directory for segment files
	tempDir, err := os.MkdirTemp("", "tts_segments")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// TTS từng đoạn, căn chỉnh duration, adelay
	var delayedFiles []string
	log.Printf("Processing %d SRT entries for TTS", len(entries))
	for i, entry := range entries {
		log.Printf("Processing segment %d: '%s' (%.2f -> %.2f)", i, entry.Text, entry.Start, entry.End)

		// Clean text trước khi gửi lên TTS
		cleanText := strings.TrimSpace(entry.Text)
		if cleanText == "" {
			log.Printf("Segment %d: Empty text, skipping", i)
			continue
		}

		log.Printf("Segment %d: Sending to TTS: '%s'", i, cleanText)

		// TTS đoạn
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: cleanText},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: languageCode,
				Name:         voiceName,
			},
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
				SpeakingRate:    speakingRate,
				SampleRateHertz: 44100,
			},
		}
		resp, err := client.SynthesizeSpeech(ctx, &req)
		if err != nil {
			return "", fmt.Errorf("failed to synthesize speech for segment %d: %v", i, err)
		}
		segmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.mp3", i))
		if err := os.WriteFile(segmentFile, resp.AudioContent, 0644); err != nil {
			return "", fmt.Errorf("failed to save segment %d: %v", i, err)
		}
		// Convert to WAV
		wavSegmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.wav", i))
		cmd := exec.Command("ffmpeg",
			"-i", segmentFile,
			"-af", "volume=2.0",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			wavSegmentFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg segment conversion error: %s", string(output))
			return "", fmt.Errorf("failed to convert segment %d to WAV: %v", i, err)
		}
		// Get actual duration
		cmd = exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", wavSegmentFile)
		durationStr, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get audio duration for segment %d: %v", i, err)
		}
		actualDuration, err := strconv.ParseFloat(strings.TrimSpace(string(durationStr)), 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse audio duration for segment %d: %v", i, err)
		}
		expectedDuration := entry.End - entry.Start
		adjustedWavFile := wavSegmentFile
		if math.Abs(actualDuration-expectedDuration) > 0.05 {
			adjustedWavFile = filepath.Join(tempDir, fmt.Sprintf("adjusted_segment_%d.wav", i))
			if actualDuration < expectedDuration {
				// Add silence
				silenceDuration := expectedDuration - actualDuration
				silenceFile := filepath.Join(tempDir, fmt.Sprintf("silence_%d.wav", i))
				cmd := exec.Command("ffmpeg",
					"-f", "lavfi",
					"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", silenceDuration),
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					silenceFile)
				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg silence creation error: %s", string(output))
					return "", fmt.Errorf("failed to create silence for segment %d: %v", i, err)
				}
				// Concatenate segment with silence
				cmd = exec.Command("ffmpeg",
					"-i", wavSegmentFile,
					"-i", silenceFile,
					"-filter_complex", "[0:a][1:a]concat=n=2:v=0:a=1[a]",
					"-map", "[a]",
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					adjustedWavFile)
				output, err = cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg concat error: %s", string(output))
					return "", fmt.Errorf("failed to concat segment %d with silence: %v", i, err)
				}
			} else {
				// Trim segment
				cmd := exec.Command("ffmpeg",
					"-i", wavSegmentFile,
					"-t", fmt.Sprintf("%f", expectedDuration),
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					adjustedWavFile)
				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg trim error: %s", string(output))
					return "", fmt.Errorf("failed to trim segment %d: %v", i, err)
				}
			}
		}
		// Dùng adelay để căn đúng thời điểm bắt đầu
		delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_%d.wav", i))
		cmd = exec.Command("ffmpeg",
			"-i", adjustedWavFile,
			"-af", fmt.Sprintf("adelay=%d|%d", int(entry.Start*1000), int(entry.Start*1000)),
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			delayedFile)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg delay error for segment %d: %s", i, string(output))
			return "", fmt.Errorf("failed to delay segment %d: %v", i, err)
		}
		delayedFiles = append(delayedFiles, delayedFile)
	}

	// Mix all delayed files
	if len(delayedFiles) > 0 {
		args := []string{"-i"}
		args = append(args, delayedFiles[0])
		for _, delayedFile := range delayedFiles[1:] {
			args = append(args, "-i", delayedFile)
		}
		filter := fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(delayedFiles))
		args = append(args,
			"-filter_complex", filter,
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)
		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg final mix error: %s", string(output))
			return "", fmt.Errorf("failed to create final TTS audio: %v", err)
		}
		log.Printf("TTS audio created successfully: %s", outputPath)
		return outputPath, nil
	} else {
		return "", fmt.Errorf("no valid segments processed")
	}
}

// processSegmentsProgressive xử lý segments lớn bằng cách mix từng phần
func processSegmentsProgressive(segmentFiles []string, entries []SRTEntry, baseAudioFile, outputPath, tempDir string, totalDuration float64) (string, error) {
	log.Printf("Starting progressive mixing for %d segments", len(segmentFiles))

	// Tạo base silence
	baseSilence := filepath.Join(tempDir, "base_silence.wav")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", totalDuration),
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "pcm_s16le",
		"-y",
		baseSilence)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create base silence: %v", err)
	}

	// Xử lý từng batch 50 segments
	batchSize := 50
	currentMixFile := baseSilence

	for i := 0; i < len(segmentFiles); i += batchSize {
		end := i + batchSize
		if end > len(segmentFiles) {
			end = len(segmentFiles)
		}

		batchSegments := segmentFiles[i:end]
		batchEntries := entries[i:end]

		log.Printf("Processing batch %d-%d (%d segments)", i+1, end, len(batchSegments))

		// Tạo delayed files cho batch này
		var delayedFiles []string
		for j, segmentFile := range batchSegments {
			delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_batch_%d_%d.wav", i, j))

			cmd := exec.Command("ffmpeg",
				"-i", segmentFile,
				"-af", fmt.Sprintf("volume=2.0,adelay=%d|%d", int(batchEntries[j].Start*1000), int(batchEntries[j].Start*1000)),
				"-ar", "44100",
				"-ac", "2",
				"-acodec", "pcm_s16le",
				"-y",
				delayedFile)

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("FFmpeg delay error for batch segment %d: %s", j, string(output))
				continue
			}
			delayedFiles = append(delayedFiles, delayedFile)
		}

		if len(delayedFiles) == 0 {
			continue
		}

		// Mix batch với current mix
		batchMixFile := filepath.Join(tempDir, fmt.Sprintf("batch_mix_%d.wav", i))
		args := []string{"-i", currentMixFile}
		for _, delayedFile := range delayedFiles {
			args = append(args, "-i", delayedFile)
		}
		args = append(args,
			"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(delayedFiles)+1),
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			batchMixFile)

		cmd = exec.Command("ffmpeg", args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg batch mix error: %s", string(output))
			continue
		}

		currentMixFile = batchMixFile
	}

	// Convert final mix to MP3
	cmd = exec.Command("ffmpeg",
		"-i", currentMixFile,
		"-acodec", "libmp3lame",
		"-b:a", "192k",
		"-y",
		outputPath)

	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert final mix to MP3: %v", err)
	}

	log.Printf("Progressive mixing completed successfully: %s", outputPath)
	return outputPath, nil
}

// processSegmentsInBatches xử lý segments rất lớn bằng cách xử lý theo batch
func processSegmentsInBatches(segmentFiles []string, entries []SRTEntry, baseAudioFile, outputPath, tempDir string, totalDuration float64) (string, error) {
	log.Printf("Starting batch processing for %d segments", len(segmentFiles))

	// Tạo base silence
	baseSilence := filepath.Join(tempDir, "base_silence.wav")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", totalDuration),
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "pcm_s16le",
		"-y",
		baseSilence)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create base silence: %v", err)
	}

	// Xử lý theo batch 30 segments
	batchSize := 30
	var batchFiles []string

	for i := 0; i < len(segmentFiles); i += batchSize {
		end := i + batchSize
		if end > len(segmentFiles) {
			end = len(segmentFiles)
		}

		batchSegments := segmentFiles[i:end]
		batchEntries := entries[i:end]

		log.Printf("Processing batch %d-%d (%d segments)", i+1, end, len(batchSegments))

		// Tạo batch file
		batchFile := filepath.Join(tempDir, fmt.Sprintf("batch_%d.wav", i))

		// Tạo delayed files cho batch này
		var delayedFiles []string
		for j, segmentFile := range batchSegments {
			delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_batch_%d_%d.wav", i, j))

			cmd := exec.Command("ffmpeg",
				"-i", segmentFile,
				"-af", fmt.Sprintf("volume=2.0,adelay=%d|%d", int(batchEntries[j].Start*1000), int(batchEntries[j].Start*1000)),
				"-ar", "44100",
				"-ac", "2",
				"-acodec", "pcm_s16le",
				"-y",
				delayedFile)

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("FFmpeg delay error for batch segment %d: %s", j, string(output))
				continue
			}
			delayedFiles = append(delayedFiles, delayedFile)
		}

		if len(delayedFiles) == 0 {
			continue
		}

		// Mix batch
		args := []string{"-i", baseSilence}
		for _, delayedFile := range delayedFiles {
			args = append(args, "-i", delayedFile)
		}
		args = append(args,
			"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest[out]", len(delayedFiles)+1),
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			batchFile)

		cmd = exec.Command("ffmpeg", args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg batch mix error: %s", string(output))
			continue
		}

		batchFiles = append(batchFiles, batchFile)
	}

	// Mix tất cả batch files
	if len(batchFiles) == 0 {
		// Nếu không có batch nào, copy base silence
		cmd = exec.Command("ffmpeg",
			"-i", baseSilence,
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)

		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to copy base audio: %v", err)
		}
	} else {
		// Mix tất cả batches
		args := []string{"-i", baseSilence}
		for _, batchFile := range batchFiles {
			args = append(args, "-i", batchFile)
		}
		args = append(args,
			"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest[out]", len(batchFiles)+1),
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)

		cmd = exec.Command("ffmpeg", args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to mix all batches: %v", err)
		}
	}

	log.Printf("Batch processing completed successfully: %s", outputPath)
	return outputPath, nil
}

// processSegmentsConcat xử lý segments trung bình bằng cách tạo delayed files riêng biệt
func processSegmentsConcat(segmentFiles []string, entries []SRTEntry, baseAudioFile, outputPath, tempDir string, totalDuration float64) (string, error) {
	log.Printf("Starting concat approach for %d segments", len(segmentFiles))

	// Create individual delayed files first
	var delayedFiles []string
	for i, segmentFile := range segmentFiles {
		delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_%d.wav", i))

		// Create delayed segment using FFmpeg with volume boost
		cmd := exec.Command("ffmpeg",
			"-i", segmentFile,
			"-af", fmt.Sprintf("volume=2.0,adelay=%d|%d", int(entries[i].Start*1000), int(entries[i].Start*1000)),
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			delayedFile)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg delay error for segment %d: %s", i, string(output))
			continue
		}
		delayedFiles = append(delayedFiles, delayedFile)
	}

	// Mix all delayed files with base silence
	if len(delayedFiles) > 0 {
		args := []string{"-i", baseAudioFile}
		for _, delayedFile := range delayedFiles {
			args = append(args, "-i", delayedFile)
		}
		args = append(args,
			"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest[out]", len(delayedFiles)+1),
			"-map", "[out]",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)

		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg final mix error: %s", string(output))
			return "", fmt.Errorf("failed to create final TTS audio: %v", err)
		}

		log.Printf("TTS audio created successfully: %s", outputPath)
		return outputPath, nil
	} else {
		// If no delayed files, just copy base silence
		cmd := exec.Command("ffmpeg",
			"-i", baseAudioFile,
			"-acodec", "libmp3lame",
			"-b:a", "192k",
			"-y",
			outputPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg copy error: %s", string(output))
			return "", fmt.Errorf("failed to copy base audio: %v", err)
		}

		log.Printf("TTS audio created successfully (base only): %s", outputPath)
		return outputPath, nil
	}
}

// BatchTTSRequest combines multiple text segments into a single API call
type BatchTTSRequest struct {
	Texts      []string
	StartTimes []float64
	EndTimes   []float64
}

// processBatchTTS processes multiple text segments in a single API call when possible
func processBatchTTS(client *texttospeech.Client, ctx context.Context, entries []SRTEntry, tempDir string, speakingRate float64, targetLanguage string) ([]string, error) {
	var segmentFiles []string

	// Group entries into batches (max 10 segments per batch to avoid API limits)
	batchSize := 10
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}

		batch := entries[i:end]
		log.Printf("Processing batch %d/%d: segments %d-%d", (i/batchSize)+1, (len(entries)+batchSize-1)/batchSize, i+1, end)

		// Combine all texts in this batch
		var combinedText string
		for j, entry := range batch {
			if j > 0 {
				combinedText += " [PAUSE] " // Add pause marker between segments
			}
			combinedText += entry.Text
		}

		// Get voice settings for target language
		languageCode, voiceName := getVoiceForLanguage(targetLanguage)

		// Single API call for the entire batch
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: combinedText},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: languageCode,
				Name:         voiceName,
			},
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
				SpeakingRate:    speakingRate,
				SampleRateHertz: 44100,
			},
		}

		resp, err := client.SynthesizeSpeech(ctx, &req)
		if err != nil {
			return nil, fmt.Errorf("failed to synthesize speech for batch %d: %v", (i/batchSize)+1, err)
		}

		// Save batch audio
		batchFile := filepath.Join(tempDir, fmt.Sprintf("batch_%d.mp3", i/batchSize))
		if err := os.WriteFile(batchFile, resp.AudioContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to save batch %d: %v", (i/batchSize)+1, err)
		}

		// Convert to WAV with volume boost
		wavBatchFile := filepath.Join(tempDir, fmt.Sprintf("batch_%d.wav", i/batchSize))
		cmd := exec.Command("ffmpeg",
			"-i", batchFile,
			"-af", "volume=2.0",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			wavBatchFile)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg batch conversion error: %s", string(output))
			return nil, fmt.Errorf("failed to convert batch %d to WAV: %v", (i/batchSize)+1, err)
		}

		segmentFiles = append(segmentFiles, wavBatchFile)
	}

	return segmentFiles, nil
}

// processIndividualTTS processes each text segment individually (optimized version)
func processIndividualTTS(client *texttospeech.Client, ctx context.Context, entries []SRTEntry, tempDir string, speakingRate float64, targetLanguage string) ([]string, error) {
	var segmentFiles []string

	// Group consecutive short segments to reduce API calls
	var groupedEntries []SRTEntry
	var currentGroup []SRTEntry

	for i, entry := range entries {
		// If current segment is short (< 2 seconds) and close to previous segment (< 0.5s gap)
		shouldGroup := len(entry.Text) < 50 && (entry.End-entry.Start) < 2.0

		if i > 0 {
			prevEntry := entries[i-1]
			gap := entry.Start - prevEntry.End
			shouldGroup = shouldGroup && gap < 0.5
		}

		if shouldGroup && len(currentGroup) < 3 { // Max 3 segments per group
			currentGroup = append(currentGroup, entry)
		} else {
			// Process current group if exists
			if len(currentGroup) > 0 {
				groupedEntry := mergeSegments(currentGroup)
				groupedEntries = append(groupedEntries, groupedEntry)
				currentGroup = nil
			}

			// Start new group or add as individual
			if shouldGroup {
				currentGroup = []SRTEntry{entry}
			} else {
				groupedEntries = append(groupedEntries, entry)
			}
		}
	}

	// Process remaining group
	if len(currentGroup) > 0 {
		groupedEntry := mergeSegments(currentGroup)
		groupedEntries = append(groupedEntries, groupedEntry)
	}

	log.Printf("Reduced from %d to %d segments by grouping", len(entries), len(groupedEntries))

	// Process each entry (individual or grouped)
	for i, entry := range groupedEntries {
		log.Printf("Processing segment %d/%d: %.2f - %.2f", i+1, len(groupedEntries), entry.Start, entry.End)

		// Get voice settings for target language
		languageCode, voiceName := getVoiceForLanguage(targetLanguage)

		// Convert text to speech
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: entry.Text},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: languageCode,
				Name:         voiceName,
			},
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
				SpeakingRate:    speakingRate,
				SampleRateHertz: 44100,
			},
		}

		resp, err := client.SynthesizeSpeech(ctx, &req)
		if err != nil {
			return nil, fmt.Errorf("failed to synthesize speech for segment %d: %v", i, err)
		}

		// Save segment to temporary file
		segmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.mp3", i))
		if err := os.WriteFile(segmentFile, resp.AudioContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to save segment %d: %v", i, err)
		}
		// Log đường dẫn, kích thước, thời lượng file mp3 gốc Google trả về
		_, _ = os.Stat(segmentFile)
		// Convert to WAV with consistent format and boost volume
		wavSegmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.wav", i))
		cmd := exec.Command("ffmpeg",
			"-i", segmentFile,
			"-af", "volume=2.0",
			"-ar", "44100",
			"-ac", "2",
			"-acodec", "pcm_s16le",
			"-y",
			wavSegmentFile)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("FFmpeg segment conversion error: %s", string(output))
			return nil, fmt.Errorf("failed to convert segment %d to WAV: %v", i, err)
		}

		// Get actual duration of generated audio
		cmd = exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", wavSegmentFile)
		durationStr, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get audio duration for segment %d: %v", i, err)
		}
		actualDuration, err := strconv.ParseFloat(strings.TrimSpace(string(durationStr)), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse audio duration for segment %d: %v", i, err)
		}

		expectedDuration := entry.End - entry.Start
		log.Printf("Segment %d: expected %.2fs, actual %.2fs", i+1, expectedDuration, actualDuration)

		// Adjust segment duration to match SRT timing
		adjustedWavFile := wavSegmentFile
		if math.Abs(actualDuration-expectedDuration) > 0.05 { // 50ms tolerance
			adjustedWavFile = filepath.Join(tempDir, fmt.Sprintf("adjusted_segment_%d.wav", i))

			if actualDuration < expectedDuration {
				// Add silence to match expected duration
				silenceDuration := expectedDuration - actualDuration
				silenceFile := filepath.Join(tempDir, fmt.Sprintf("silence_%d.wav", i))
				cmd := exec.Command("ffmpeg",
					"-f", "lavfi",
					"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", silenceDuration),
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					silenceFile)

				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg silence creation error: %s", string(output))
					return nil, fmt.Errorf("failed to create silence for segment %d: %v", i, err)
				}

				// Concatenate segment with silence using filter_complex
				cmd = exec.Command("ffmpeg",
					"-i", wavSegmentFile,
					"-i", silenceFile,
					"-filter_complex", "[0:a][1:a]concat=n=2:v=0:a=1[a]",
					"-map", "[a]",
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					adjustedWavFile)

				output, err = cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg concat error: %s", string(output))
					return nil, fmt.Errorf("failed to concat segment %d with silence: %v", i, err)
				}
			} else {
				// Trim segment to match expected duration
				cmd := exec.Command("ffmpeg",
					"-i", wavSegmentFile,
					"-t", fmt.Sprintf("%f", expectedDuration),
					"-ar", "44100",
					"-ac", "2",
					"-acodec", "pcm_s16le",
					"-y",
					adjustedWavFile)

				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("FFmpeg trim error: %s", string(output))
					return nil, fmt.Errorf("failed to trim segment %d: %v", i, err)
				}
			}
		}

		segmentFiles = append(segmentFiles, adjustedWavFile)
	}

	return segmentFiles, nil
}

// mergeSegments combines multiple short segments into one
func mergeSegments(segments []SRTEntry) SRTEntry {
	if len(segments) == 0 {
		return SRTEntry{}
	}

	if len(segments) == 1 {
		return segments[0]
	}

	// Combine text with small pauses
	var combinedText string
	for i, segment := range segments {
		if i > 0 {
			combinedText += " " // Small pause between segments
		}
		combinedText += segment.Text
	}

	// Use timing from first and last segment
	return SRTEntry{
		Index: segments[0].Index,
		Start: segments[0].Start,
		End:   segments[len(segments)-1].End,
		Text:  combinedText,
	}
}

// ConvertSRTToSpeechWithService wrapper function that uses service_config to determine which TTS service to use
func ConvertSRTToSpeechWithService(srtContent string, videoDir string, speakingRate float64, targetLanguage string, serviceName string, modelAPIName string) (string, error) {
	// Currently only Google TTS (tts_wavenet) is supported for text_to_speech
	if serviceName == "tts_wavenet" {
		return ConvertSRTToSpeechWithLanguage(srtContent, videoDir, speakingRate, targetLanguage)
	}

	// For future services, we can add more conditions here
	// Example: if serviceName == "azure_tts" { return ConvertSRTToSpeechWithAzure(srtContent, videoDir, speakingRate, targetLanguage, modelAPIName) }

	return "", fmt.Errorf("unsupported text-to-speech service: %s", serviceName)
}
