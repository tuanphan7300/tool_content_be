package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// TranslateSRTFile translates an SRT file from Chinese to Vietnamese using Gemini
func TranslateSRTFile(srtFilePath, apiKey string) (string, error) {
	// Read the original SRT file
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Create the prompt for Gemini
	prompt := `Hãy dịch file SRT từ tiếng Trung sang tiếng Việt.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:

QUY TẮC QUAN TRỌNG NHẤT: Giữ nguyên 100% số thứ tự và dòng thời gian (timestamps) từ file gốc. TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Dòng thời gian phải được sao chép y hệt.

Về nội dung: Dịch tự nhiên, truyền cảm, phù hợp với văn nói. Rút gọn các câu quá dài để khớp với thời gian hiển thị.

Kiểm tra cuối cùng: Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn không có dòng thời gian nào bị sai lệch.

File SRT gốc:
` + string(srtContent)

	// Dùng model mặc định nếu không truyền vào
	modelName := "gemini-1.5-flash-latest"
	translatedContent, err := GenerateWithGemini(prompt, apiKey, modelName)
	if err != nil {
		return "", fmt.Errorf("failed to translate SRT with Gemini: %v", err)
	}

	// Clean up the response - remove any extra text that might be added by Gemini
	translatedContent = strings.TrimSpace(translatedContent)

	// If Gemini added any prefix or explanation, try to extract just the SRT content
	if strings.Contains(translatedContent, "1\n") {
		// Find the start of the SRT content
		startIndex := strings.Index(translatedContent, "1\n")
		if startIndex != -1 {
			translatedContent = translatedContent[startIndex:]
		}
	}

	return translatedContent, nil
}

// CreateSRTFromSegments creates an SRT file from segments and then translates it
func CreateSRTFromSegments(segments []Segment, outputPath string) error {
	var srtBuilder strings.Builder

	for i, segment := range segments {
		// SRT format: index, start --> end, text
		srtBuilder.WriteString(fmt.Sprintf("%d\n", i+1))
		srtBuilder.WriteString(fmt.Sprintf("%s --> %s\n",
			formatTime(segment.Start),
			formatTime(segment.End)))
		srtBuilder.WriteString(segment.Text + "\n\n")
	}

	// Write the original SRT file
	err := os.WriteFile(outputPath, []byte(srtBuilder.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write SRT file: %v", err)
	}

	return nil
}

// formatTime formats time in SRT format (HH:MM:SS,mmm)
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// TranslateAndCreateSRT creates SRT from segments, translates it, and saves both versions
func TranslateAndCreateSRT(segments []Segment, outputDir, filename string, apiKey string) (string, string, error) {
	// Create original SRT file
	originalSRTPath := fmt.Sprintf("%s/%s_original.srt", outputDir, filename)
	err := CreateSRTFromSegments(segments, originalSRTPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create original SRT: %v", err)
	}

	// Translate the SRT file
	translatedContent, err := TranslateSRTFile(originalSRTPath, apiKey)
	if err != nil {
		return originalSRTPath, "", fmt.Errorf("failed to translate SRT: %v", err)
	}

	// Save translated SRT file
	translatedSRTPath := fmt.Sprintf("%s/%s_vi.srt", outputDir, filename)
	err = os.WriteFile(translatedSRTPath, []byte(translatedContent), 0644)
	if err != nil {
		return originalSRTPath, "", fmt.Errorf("failed to save translated SRT: %v", err)
	}

	return originalSRTPath, translatedSRTPath, nil
}

// ParseSRTToSegments parses an SRT file and converts it back to segments
func ParseSRTToSegments(srtFilePath string) ([]Segment, error) {
	content, err := os.ReadFile(srtFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRT file: %v", err)
	}

	var segments []Segment
	lines := strings.Split(string(content), "\n")

	i := 0
	for i < len(lines) {
		// Skip empty lines
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) {
			break
		}

		// Parse index
		indexStr := strings.TrimSpace(lines[i])
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			i++
			continue
		}
		i++

		// Parse timestamp
		if i >= len(lines) {
			break
		}
		timestampLine := strings.TrimSpace(lines[i])
		timeParts := strings.Split(timestampLine, " --> ")
		if len(timeParts) != 2 {
			i++
			continue
		}

		startTime, err := parseSRTTime(timeParts[0])
		if err != nil {
			i++
			continue
		}

		endTime, err := parseSRTTime(timeParts[1])
		if err != nil {
			i++
			continue
		}
		i++

		// Parse text
		var textLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textLines = append(textLines, strings.TrimSpace(lines[i]))
			i++
		}
		text := strings.Join(textLines, " ")

		segments = append(segments, Segment{
			ID:    index,
			Start: startTime,
			End:   endTime,
			Text:  text,
		})
	}

	return segments, nil
}

// EstimateGeminiTokens estimates the number of tokens that will be used for SRT translation
func EstimateGeminiTokens(srtContent string, apiKey string) (int, error) {
	prompt := `Hãy dịch file SRT từ tiếng Trung sang tiếng Việt.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:

QUY TẮC QUAN TRỌNG NHẤT: Giữ nguyên 100% số thứ tự và dòng thời gian (timestamps) từ file gốc. TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Dòng thời gian phải được sao chép y hệt.

Về nội dung: Dịch tự nhiên, truyền cảm, phù hợp với văn nói. Rút gọn các câu quá dài để khớp với thời gian hiển thị.

Kiểm tra cuối cùng: Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn không có dòng thời gian nào bị sai lệch.

File SRT gốc:
` + srtContent
	modelName := "gemini-1.5-flash-latest"
	tokens, err := CountTokens(prompt, apiKey, modelName)
	if err != nil {
		tokens = int(float64(len(srtContent))/62.5 + 0.9999)
		if tokens < 1 {
			tokens = 1
		}
	}
	return tokens, nil
}

// TranslateSRTFileWithModel dịch SRT với modelName động
func TranslateSRTFileWithModel(srtFilePath, apiKey, modelName string) (string, error) {
	// Read the original SRT file
	log.Infof("sử dụng model gemini %s", modelName)
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Create the prompt for Gemini
	prompt := `Hãy dịch file SRT từ tiếng Trung sang tiếng Việt.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:

QUY TẮC QUAN TRỌNG NHẤT: Giữ nguyên 100% số thứ tự và dòng thời gian (timestamps) từ file gốc. TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Dòng thời gian phải được sao chép y hệt.

Về nội dung: Dịch tự nhiên, truyền cảm, phù hợp với văn nói. Rút gọn các câu quá dài để khớp với thời gian hiển thị.

Kiểm tra cuối cùng: Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn không có dòng thời gian nào bị sai lệch.

File SRT gốc:
` + string(srtContent)

	// Call Gemini API
	translatedContent, err := GenerateWithGemini(prompt, apiKey, modelName)
	if err != nil {
		return "", fmt.Errorf("failed to translate SRT with Gemini: %v", err)
	}

	// Clean up the response - remove any extra text that might be added by Gemini
	translatedContent = strings.TrimSpace(translatedContent)

	// If Gemini added any prefix or explanation, try to extract just the SRT content
	if strings.Contains(translatedContent, "1\n") {
		// Find the start of the SRT content
		startIndex := strings.Index(translatedContent, "1\n")
		if startIndex != -1 {
			translatedContent = translatedContent[startIndex:]
		}
	}

	return translatedContent, nil
}

// TranslateSRTFileWithModelAndLanguage dịch SRT với modelName động và ngôn ngữ đích
func TranslateSRTFileWithModelAndLanguage(srtFilePath, apiKey, modelName, targetLanguage string) (string, error) {
	// Read the original SRT file
	log.Infof("sử dụng model gemini %s để dịch sang %s", modelName, targetLanguage)
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Map language codes to language names
	languageNames := map[string]string{
		"vi": "tiếng Việt",
		"en": "tiếng Anh",
		"ja": "tiếng Nhật",
		"ko": "tiếng Hàn",
		"zh": "tiếng Trung",
		"fr": "tiếng Pháp",
		"de": "tiếng Đức",
		"es": "tiếng Tây Ban Nha",
	}

	targetLangName := languageNames[targetLanguage]
	if targetLangName == "" {
		targetLangName = "tiếng Việt" // Default to Vietnamese
	}

	// Create the prompt for Gemini with dynamic target language
	prompt := fmt.Sprintf(`Hãy dịch file SRT sang %s.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:

QUY TẮC QUAN TRỌNG NHẤT: Giữ nguyên 100%% số thứ tự và dòng thời gian (timestamps) từ file gốc. TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Dòng thời gian phải được sao chép y hệt.

Về nội dung: Dịch tự nhiên, truyền cảm, phù hợp với văn nói. Rút gọn các câu quá dài để khớp với thời gian hiển thị.

Kiểm tra cuối cùng: Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn không có dòng thời gian nào bị sai lệch.

File SRT gốc:
%s`, targetLangName, string(srtContent))

	// Call Gemini API
	translatedContent, err := GenerateWithGemini(prompt, apiKey, modelName)
	if err != nil {
		return "", fmt.Errorf("failed to translate SRT with Gemini: %v", err)
	}

	// Clean up the response - remove any extra text that might be added by Gemini
	translatedContent = strings.TrimSpace(translatedContent)

	// If Gemini added any prefix or explanation, try to extract just the SRT content
	if strings.Contains(translatedContent, "1\n") {
		// Find the start of the SRT content
		startIndex := strings.Index(translatedContent, "1\n")
		if startIndex != -1 {
			translatedContent = translatedContent[startIndex:]
		}
	}

	return translatedContent, nil
}
