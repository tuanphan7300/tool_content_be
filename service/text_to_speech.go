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
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(outputDir, fmt.Sprintf("tts_%s.mp3", timestamp))

	// Initialize Google TTS client
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		return "", fmt.Errorf("failed to create TTS client: %v", err)
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
		return "", fmt.Errorf("failed to synthesize speech: %v", err)
	}

	// Write the response to the output file
	if err := os.WriteFile(filename, resp.AudioContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write audio file: %v", err)
	}

	return filename, nil
}

// parseSRT parses SRT content and returns a slice of SRTEntry
func parseSRT(srtContent string) ([]SRTEntry, error) {
	var entries []SRTEntry
	blocks := strings.Split(srtContent, "\n\n")

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}

		// Parse index
		index, err := strconv.Atoi(lines[0])
		if err != nil {
			continue
		}

		// Parse timestamp
		timestamp := strings.Split(lines[1], " --> ")
		if len(timestamp) != 2 {
			continue
		}

		start, err := parseSRTTime(timestamp[0])
		if err != nil {
			continue
		}

		end, err := parseSRTTime(timestamp[1])
		if err != nil {
			continue
		}

		// Get text
		text := strings.Join(lines[2:], " ")

		entries = append(entries, SRTEntry{
			Index: index,
			Start: start,
			End:   end,
			Text:  text,
		})
	}

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

// ConvertSRTToSpeech converts SRT content to speech and returns the audio file path
func ConvertSRTToSpeech(srtContent string, videoDir string, speakingRate float64) (string, error) {
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
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		return "", fmt.Errorf("failed to create TTS client: %v", err)
	}
	defer client.Close()

	// Create a temporary directory for segment files
	tempDir, err := os.MkdirTemp("", "tts_segments")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Calculate total duration needed
	totalDuration := entries[len(entries)-1].End
	log.Printf("Total TTS duration needed: %.2f seconds", totalDuration)

	// Create a silent base audio of total duration with consistent format
	baseAudioFile := filepath.Join(tempDir, "base_silence.wav")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", totalDuration),
		"-ar", "44100",
		"-ac", "2",
		"-acodec", "pcm_s16le",
		"-y",
		baseAudioFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg base silence error: %s", string(output))
		return "", fmt.Errorf("failed to create base silence: %v", err)
	}

	// Use individual processing for all segments (batch processing causes audio overlap)
	var segmentFiles []string
	log.Printf("Using individual processing for %d segments", len(entries))
	segmentFiles, err = processIndividualTTS(client, ctx, entries, tempDir, speakingRate)
	if err != nil {
		return "", err
	}

	// Dynamic segment handling based on count
	totalSegments := len(segmentFiles)
	log.Printf("Processing %d segments for TTS", totalSegments)

	// For very large numbers of segments, use batch processing
	if totalSegments > 200 {
		log.Printf("Very large number of segments (%d), using batch processing", totalSegments)
		return processSegmentsInBatches(segmentFiles, entries, baseAudioFile, outputPath, tempDir, totalDuration)
	}

	// For large numbers of segments, use progressive mixing
	if totalSegments > 50 {
		log.Printf("Large number of segments (%d), using progressive mixing", totalSegments)
		return processSegmentsProgressive(segmentFiles, entries, baseAudioFile, outputPath, tempDir, totalDuration)
	}

	// For medium numbers of segments, use concat approach
	if totalSegments > 20 {
		log.Printf("Medium number of segments (%d), using concat approach", totalSegments)
		return processSegmentsConcat(segmentFiles, entries, baseAudioFile, outputPath, tempDir, totalDuration)
	}

	// For small numbers of segments (≤20), use individual adelay filters
	log.Printf("Small number of segments (%d), using individual adelay filters", totalSegments)

	// Create individual delayed files first to avoid filter complexity issues
	var delayedFiles []string
	for i, segmentFile := range segmentFiles {
		delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_%d.wav", i))

		// Create delayed segment using FFmpeg with volume boost
		cmd := exec.Command("ffmpeg",
			"-i", segmentFile,
			"-af", fmt.Sprintf("volume=3.0,adelay=%d|%d", int(entries[i].Start*1000), int(entries[i].Start*1000)),
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
			"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[out]", len(delayedFiles)+1),
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
				"-af", fmt.Sprintf("volume=3.0,adelay=%d|%d", int(batchEntries[j].Start*1000), int(batchEntries[j].Start*1000)),
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
				"-af", fmt.Sprintf("volume=3.0,adelay=%d|%d", int(batchEntries[j].Start*1000), int(batchEntries[j].Start*1000)),
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
			"-af", fmt.Sprintf("volume=3.0,adelay=%d|%d", int(entries[i].Start*1000), int(entries[i].Start*1000)),
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
func processBatchTTS(client *texttospeech.Client, ctx context.Context, entries []SRTEntry, tempDir string, speakingRate float64) ([]string, error) {
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

		// Single API call for the entire batch
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: combinedText},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: "vi-VN",
				Name:         "vi-VN-Wavenet-C",
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
			"-af", "volume=3.0",
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
func processIndividualTTS(client *texttospeech.Client, ctx context.Context, entries []SRTEntry, tempDir string, speakingRate float64) ([]string, error) {
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

		// Convert text to speech
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: entry.Text},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: "vi-VN",
				Name:         "vi-VN-Wavenet-C",
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

		// Convert to WAV with consistent format and boost volume
		wavSegmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.wav", i))
		cmd := exec.Command("ffmpeg",
			"-i", segmentFile,
			"-af", "volume=3.0",
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
