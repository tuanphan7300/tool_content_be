package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	// Initialize the client with credentials file
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
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

	// Create output directory if it doesn't exist
	outputDir := filepath.Join(videoDir, "tts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate output filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("tts_%s.mp3", timestamp))

	// Initialize text-to-speech client
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

	// Process each segment
	var segmentFiles []string

	for i, entry := range entries {
		// Convert text to speech first
		req := texttospeechpb.SynthesizeSpeechRequest{
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: entry.Text},
			},
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: "vi-VN",
				Name:         "vi-VN-Wavenet-C",
			},
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding: texttospeechpb.AudioEncoding_MP3,
				SpeakingRate:  speakingRate,
			},
		}

		resp, err := client.SynthesizeSpeech(ctx, &req)
		if err != nil {
			return "", fmt.Errorf("failed to synthesize speech: %v", err)
		}

		// Save segment to temporary file
		segmentFile := filepath.Join(tempDir, fmt.Sprintf("segment_%d.mp3", i))
		if err := os.WriteFile(segmentFile, resp.AudioContent, 0644); err != nil {
			return "", fmt.Errorf("failed to save segment: %v", err)
		}

		// Get the actual duration of the generated audio
		cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", segmentFile)
		durationStr, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get audio duration: %v", err)
		}
		actualDuration, err := strconv.ParseFloat(strings.TrimSpace(string(durationStr)), 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse audio duration: %v", err)
		}

		expectedDuration := entry.End - entry.Start
		adjustedSegmentFile := segmentFile

		// Nếu audio TTS ngắn hơn duration SRT, chèn silence vào cuối cho đủ
		if actualDuration < expectedDuration-0.05 {
			silenceFile := filepath.Join(tempDir, fmt.Sprintf("silence_end_%d.mp3", i))
			silenceDuration := expectedDuration - actualDuration
			cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", silenceDuration), "-q:a", "0", "-map", "0:a", silenceFile)
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to create end silence: %v", err)
			}
			// Ghép segmentFile + silenceFile thành adjustedSegmentFile
			concatList := filepath.Join(tempDir, fmt.Sprintf("concat_%d.txt", i))
			concatContent := fmt.Sprintf("file '%s'\nfile '%s'\n", segmentFile, silenceFile)
			if err := os.WriteFile(concatList, []byte(concatContent), 0644); err != nil {
				return "", fmt.Errorf("failed to create concat list: %v", err)
			}
			adjustedSegmentFile = filepath.Join(tempDir, fmt.Sprintf("adjusted_segment_%d.mp3", i))
			cmd = exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatList, "-c", "copy", adjustedSegmentFile)
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to concat segment and silence: %v", err)
			}
		} else if actualDuration > expectedDuration+0.05 {
			// Nếu audio TTS dài hơn duration SRT, cắt bớt
			trimmedFile := filepath.Join(tempDir, fmt.Sprintf("trimmed_segment_%d.mp3", i))
			cmd := exec.Command("ffmpeg", "-i", segmentFile, "-t", fmt.Sprintf("%f", expectedDuration), "-c", "copy", trimmedFile)
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to trim segment: %v", err)
			}
			adjustedSegmentFile = trimmedFile
		}

		// Chèn silence trước nếu cần để bắt đầu đúng tại entry.Start
		var silenceBefore float64
		if entry.Start > 0 {
			if i == 0 {
				silenceBefore = entry.Start
			} else {
				prevEnd := entries[i-1].End
				silenceBefore = entry.Start - prevEnd
				if silenceBefore < 0 {
					silenceBefore = 0
				}
			}
		}
		if silenceBefore > 0 {
			silenceFile := filepath.Join(tempDir, fmt.Sprintf("silence_%d.mp3", i))
			cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", fmt.Sprintf("anullsrc=r=44100:cl=stereo:d=%f", silenceBefore), "-q:a", "0", "-map", "0:a", silenceFile)
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to create silence: %v", err)
			}
			segmentFiles = append(segmentFiles, silenceFile)
		}
		segmentFiles = append(segmentFiles, adjustedSegmentFile)
	}

	// Combine all segments
	if len(segmentFiles) > 0 {
		// Create a file list for ffmpeg
		listFile := filepath.Join(tempDir, "filelist.txt")
		var fileList strings.Builder
		for _, file := range segmentFiles {
			fileList.WriteString(fmt.Sprintf("file '%s'\n", file))
		}
		if err := os.WriteFile(listFile, []byte(fileList.String()), 0644); err != nil {
			return "", fmt.Errorf("failed to create file list: %v", err)
		}

		// Combine all segments
		cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to combine segments: %v", err)
		}
	}

	return outputPath, nil
}
