package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// TranslateSRTFile translates an SRT file from Chinese to Vietnamese using Gemini
func TranslateSRTFile(srtFilePath, apiKey, targetLanguage, modelName string) (string, error) {
	// Read the original SRT file
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Create the prompt for Gemini
	prompt := fmt.Sprintf(`Hãy dịch file SRT sang %s.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:

QUY TẮC QUAN TRỌNG NHẤT: Giữ nguyên 100%% số thứ tự và dòng thời gian (timestamps) từ file gốc. TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Dòng thời gian phải được sao chép y hệt.

Về nội dung: Dịch tự nhiên, truyền cảm, phù hợp với văn nói. Rút gọn các câu quá dài để khớp với thời gian hiển thị.

Kiểm tra cuối cùng: Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn không có dòng thời gian nào bị sai lệch.

File SRT gốc:
%s`, targetLanguage, string(srtContent))

	// Sử dụng model được truyền vào, nếu không có thì dùng default
	if modelName == "" {
		modelName = "gemini-1.5-flash-latest"
	}
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
	translatedContent, err := TranslateSRTFile(originalSRTPath, apiKey, "vi", "")
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
		return "", fmt.Errorf("Lỗi dịch thuật: %v", err)
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

// TranslateSRTFileWithGPT translates an SRT file using GPT
// This is a wrapper function that calls the actual implementation in gpt.go
func TranslateSRTFileWithGPT(srtFilePath, apiKey, modelName, targetLanguage string) (string, error) {
	return TranslateSRTWithGPT(srtFilePath, apiKey, modelName, targetLanguage)
}

// DetectSRTLanguage detects the language of SRT content using heuristic approach
func DetectSRTLanguage(srtContent string) string {
	// Extract text content from SRT (remove timestamps and numbers)
	textContent := extractTextFromSRT(srtContent)

	// Simple language detection based on character patterns and common words
	language := detectLanguageFromText(textContent)

	log.Printf("Detected language: %s for SRT content", language)
	return language
}

// extractTextFromSRT extracts only the text content from SRT, removing timestamps and numbers
func extractTextFromSRT(srtContent string) string {
	lines := strings.Split(srtContent, "\n")
	var textLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines, numbers (index), and timestamp lines
		if line == "" || isNumeric(line) || isTimestampLine(line) {
			continue
		}

		textLines = append(textLines, line)
	}

	return strings.Join(textLines, " ")
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isTimestampLine checks if a line is a timestamp line (contains -->)
func isTimestampLine(line string) bool {
	return strings.Contains(line, "-->")
}

// detectLanguageFromText detects language using character patterns and common words
func detectLanguageFromText(text string) string {
	text = strings.ToLower(text)

	// Vietnamese detection
	if containsVietnameseChars(text) {
		return "vi"
	}

	// Chinese detection (simplified and traditional)
	if containsChineseChars(text) {
		return "zh"
	}

	// Japanese detection
	if containsJapaneseChars(text) {
		return "ja"
	}

	// Korean detection
	if containsKoreanChars(text) {
		return "ko"
	}

	// English detection
	if containsEnglishWords(text) {
		return "en"
	}

	// French detection
	if containsFrenchWords(text) {
		return "fr"
	}

	// German detection
	if containsGermanWords(text) {
		return "de"
	}

	// Spanish detection
	if containsSpanishWords(text) {
		return "es"
	}

	// Default to Vietnamese if no clear pattern detected
	return "vi"
}

// containsVietnameseChars checks for Vietnamese diacritics
func containsVietnameseChars(text string) bool {
	vietnameseChars := []rune{'à', 'á', 'ạ', 'ả', 'ã', 'â', 'ầ', 'ấ', 'ậ', 'ẩ', 'ẫ', 'ă', 'ằ', 'ắ', 'ặ', 'ẳ', 'ẵ',
		'è', 'é', 'ẹ', 'ẻ', 'ẽ', 'ê', 'ề', 'ế', 'ệ', 'ể', 'ễ',
		'ì', 'í', 'ị', 'ỉ', 'ĩ',
		'ò', 'ó', 'ọ', 'ỏ', 'õ', 'ô', 'ồ', 'ố', 'ộ', 'ổ', 'ỗ', 'ơ', 'ờ', 'ớ', 'ợ', 'ở', 'ỡ',
		'ù', 'ú', 'ụ', 'ủ', 'ũ', 'ư', 'ừ', 'ứ', 'ự', 'ử', 'ữ',
		'ỳ', 'ý', 'ỵ', 'ỷ', 'ỹ',
		'đ'}

	for _, char := range vietnameseChars {
		if strings.ContainsRune(text, char) {
			return true
		}
	}
	return false
}

// containsChineseChars checks for Chinese characters
func containsChineseChars(text string) bool {
	for _, r := range text {
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
			(r >= 0x20000 && r <= 0x2A6DF) { // CJK Unified Ideographs Extension B
			return true
		}
	}
	return false
}

// containsJapaneseChars checks for Japanese characters
func containsJapaneseChars(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FFF) { // Kanji
			return true
		}
	}
	return false
}

// containsKoreanChars checks for Korean characters
func containsKoreanChars(text string) bool {
	for _, r := range text {
		if r >= 0xAC00 && r <= 0xD7AF { // Hangul Syllables
			return true
		}
	}
	return false
}

// containsEnglishWords checks for common English words
func containsEnglishWords(text string) bool {
	englishWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by", "is", "are", "was", "were", "be", "been", "have", "has", "had", "do", "does", "did", "will", "would", "could", "should", "may", "might", "can", "this", "that", "these", "those", "i", "you", "he", "she", "it", "we", "they", "me", "him", "her", "us", "them"}

	for _, word := range englishWords {
		if strings.Contains(text, " "+word+" ") || strings.HasPrefix(text, word+" ") || strings.HasSuffix(text, " "+word) {
			return true
		}
	}
	return false
}

// containsFrenchWords checks for common French words
func containsFrenchWords(text string) bool {
	frenchWords := []string{"le", "la", "les", "un", "une", "des", "et", "ou", "mais", "dans", "sur", "avec", "sans", "pour", "par", "de", "du", "des", "est", "sont", "était", "étaient", "être", "avoir", "faire", "aller", "venir", "voir", "savoir", "pouvoir", "vouloir", "devoir", "je", "tu", "il", "elle", "nous", "vous", "ils", "elles"}

	for _, word := range frenchWords {
		if strings.Contains(text, " "+word+" ") || strings.HasPrefix(text, word+" ") || strings.HasSuffix(text, " "+word) {
			return true
		}
	}
	return false
}

// containsGermanWords checks for common German words
func containsGermanWords(text string) bool {
	germanWords := []string{"der", "die", "das", "ein", "eine", "und", "oder", "aber", "in", "auf", "mit", "ohne", "für", "von", "zu", "ist", "sind", "war", "waren", "sein", "haben", "machen", "gehen", "kommen", "sehen", "wissen", "können", "wollen", "müssen", "ich", "du", "er", "sie", "es", "wir", "ihr", "sie"}

	for _, word := range germanWords {
		if strings.Contains(text, " "+word+" ") || strings.HasPrefix(text, word+" ") || strings.HasSuffix(text, " "+word) {
			return true
		}
	}
	return false
}

// containsSpanishWords checks for common Spanish words
func containsSpanishWords(text string) bool {
	spanishWords := []string{"el", "la", "los", "las", "un", "una", "unos", "unas", "y", "o", "pero", "en", "sobre", "con", "sin", "para", "por", "de", "del", "es", "son", "era", "eran", "ser", "estar", "tener", "hacer", "ir", "venir", "ver", "saber", "poder", "querer", "deber", "yo", "tú", "él", "ella", "nosotros", "vosotros", "ellos", "ellas"}

	for _, word := range spanishWords {
		if strings.Contains(text, " "+word+" ") || strings.HasPrefix(text, word+" ") || strings.HasSuffix(text, " "+word) {
			return true
		}
	}
	return false
}

// TestDetectSRTLanguage is a simple test function to verify language detection
func TestDetectSRTLanguage() {
	testCases := []struct {
		name       string
		srtContent string
		expected   string
	}{
		{
			name: "Vietnamese SRT",
			srtContent: `1
00:00:01,000 --> 00:00:04,000
Xin chào các bạn

2
00:00:04,000 --> 00:00:07,000
Tôi là người Việt Nam`,
			expected: "vi",
		},
		{
			name: "English SRT",
			srtContent: `1
00:00:01,000 --> 00:00:04,000
Hello everyone

2
00:00:04,000 --> 00:00:07,000
I am from the United States`,
			expected: "en",
		},
		{
			name: "Chinese SRT",
			srtContent: `1
00:00:01,000 --> 00:00:04,000
大家好

2
00:00:04,000 --> 00:00:07,000
我是中国人`,
			expected: "zh",
		},
	}

	for _, tc := range testCases {
		result := DetectSRTLanguage(tc.srtContent)
		if result == tc.expected {
			log.Printf("✓ %s: detected %s (expected %s)", tc.name, result, tc.expected)
		} else {
			log.Printf("✗ %s: detected %s (expected %s)", tc.name, result, tc.expected)
		}
	}
}
