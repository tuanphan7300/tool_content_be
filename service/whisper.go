package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

type Segment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type WhisperUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	DurationSeconds  float64 `json:"duration"`
}

type WhisperResponse struct {
	Text     string        `json:"text"`
	Segments []Segment     `json:"segments"`
	Usage    *WhisperUsage `json:"usage,omitempty"`
	Duration float64       `json:"duration,omitempty"`
}

// SplitLongSegments tách các segment dài thành các segment ngắn như CapCut
func SplitLongSegments(segments []Segment) []Segment {
	var result []Segment
	segmentID := 1

	// Bước 1: Clean và validate tất cả segments
	var validSegments []Segment
	for _, segment := range segments {
		// Clean text triệt để
		cleanText := cleanTextThoroughly(segment.Text)
		if cleanText == "" {
			continue
		}

		// Validate timing
		if segment.End <= segment.Start || segment.End-segment.Start < 0.1 {
			continue
		}

		validSegments = append(validSegments, Segment{
			ID:    segment.ID,
			Start: segment.Start,
			End:   segment.End,
			Text:  cleanText,
		})
	}

	// Bước 2: Merge các segments quá ngắn hoặc liên tiếp
	mergedSegments := mergeShortSegments(validSegments)

	// Bước 3: Split segments dài
	for _, segment := range mergedSegments {
		// Nếu segment ngắn hơn 3 giây, giữ nguyên
		if segment.End-segment.Start <= 3.0 {
			segment.ID = segmentID
			result = append(result, segment)
			segmentID++
			continue
		}

		// Tách segment dài thành các phần nhỏ
		splitSegments := splitSegmentIntelligently(segment, segmentID)
		result = append(result, splitSegments...)
		segmentID += len(splitSegments)
	}

	return result
}

// cleanTextThoroughly - Giải pháp triệt để để clean text
func cleanTextThoroughly(text string) string {
	if text == "" {
		return ""
	}

	// 1. Loại bỏ tất cả ký tự Unicode không hợp lệ
	text = strings.ReplaceAll(text, "\ufffd", "")
	text = strings.ReplaceAll(text, "\u0000", "")

	// 2. Loại bỏ các ký tự control và whitespace không cần thiết
	var result strings.Builder
	for _, r := range text {
		// Chỉ giữ lại ký tự có thể in được, dấu câu, và whitespace cơ bản
		if (r >= 32 && r <= 126) || // ASCII printable
			(r >= 0x4E00 && r <= 0x9FFF) || // Chinese characters
			(r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0xAC00 && r <= 0xD7AF) || // Korean Hangul
			(r >= 0x0E00 && r <= 0x0E7F) || // Thai
			r == '\n' || r == '\t' || r == ' ' {
			result.WriteRune(r)
		}
	}

	// 3. Normalize whitespace
	cleaned := strings.TrimSpace(result.String())
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// 4. Loại bỏ segments chỉ có dấu câu
	if isOnlyPunctuation(cleaned) {
		return ""
	}

	return cleaned
}

// isOnlyPunctuation kiểm tra xem text có chỉ chứa dấu câu không
func isOnlyPunctuation(text string) bool {
	punctuation := []rune{'.', '。', '！', '!', '？', '?', '，', ',', '；', ';', '：', ':', '、', '…', '—', '(', ')', '[', ']', '{', '}', '"', '"', '\'', '\'', '-', '_'}

	for _, r := range text {
		isPunct := false
		for _, p := range punctuation {
			if r == p {
				isPunct = true
				break
			}
		}
		if !isPunct && r != ' ' && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}

// mergeShortSegments merge các segments quá ngắn hoặc liên tiếp
func mergeShortSegments(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}

	var result []Segment
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		next := segments[i]

		// Merge nếu:
		// 1. Segment hiện tại quá ngắn (< 1 giây)
		// 2. Khoảng cách giữa 2 segments quá nhỏ (< 0.5 giây)
		// 3. Cả 2 segments đều ngắn
		shouldMerge := (current.End-current.Start < 1.0) ||
			(next.End-next.Start < 1.0) ||
			(next.Start-current.End < 0.5)

		if shouldMerge {
			// Merge segments
			current.End = next.End
			current.Text = strings.TrimSpace(current.Text + " " + next.Text)
		} else {
			// Thêm segment hiện tại vào kết quả
			result = append(result, current)
			current = next
		}
	}

	// Thêm segment cuối
	result = append(result, current)

	return result
}

// splitSegmentIntelligently - Tách segment thông minh
func splitSegmentIntelligently(segment Segment, startID int) []Segment {
	startTime := segment.Start
	endTime := segment.End
	duration := endTime - startTime

	// Nếu duration quá ngắn, không tách
	if duration <= 3.0 {
		return []Segment{segment}
	}

	// Tách theo dấu câu trước
	punctuationSplits := splitByPunctuation(segment, startID)
	if len(punctuationSplits) > 1 {
		return punctuationSplits
	}

	// Nếu không tách được theo dấu câu, tách theo độ dài
	return splitByLength(segment, startID)
}

// splitByPunctuation tách segment dựa trên dấu câu
func splitByPunctuation(segment Segment, startID int) []Segment {
	var result []Segment

	// Dấu câu để tách câu (ưu tiên theo thứ tự)
	punctuationMarks := []string{".", "。", "！", "!", "？", "?", "，", ",", "；", ";", "：", ":", "、", "…", "—"}

	text := segment.Text
	startTime := segment.Start
	endTime := segment.End
	duration := endTime - startTime

	// Tìm vị trí các dấu câu
	var splitPoints []int
	for _, mark := range punctuationMarks {
		positions := findAllOccurrences(text, mark)
		for _, pos := range positions {
			splitPoints = append(splitPoints, pos)
		}
	}

	// Sắp xếp và loại bỏ trùng lặp
	splitPoints = sortAndDeduplicate(splitPoints)

	// Nếu không có đủ dấu câu để tách, return segment gốc
	if len(splitPoints) <= 1 {
		return []Segment{segment}
	}

	// Tách theo dấu câu
	currentPos := 0
	currentID := startID

	for i, splitPoint := range splitPoints {
		if splitPoint <= currentPos {
			continue
		}

		// Tính thời gian dựa trên số ký tự
		partText := text[currentPos : splitPoint+1]
		partRuneCount := len([]rune(partText))
		totalRuneCount := len([]rune(text))
		partDuration := (float64(partRuneCount) / float64(totalRuneCount)) * duration

		partStart := startTime
		if i > 0 {
			processedRunes := len([]rune(text[:currentPos]))
			partStart = startTime + (float64(processedRunes)/float64(totalRuneCount))*duration
		}
		partEnd := partStart + partDuration

		if partEnd > endTime {
			partEnd = endTime
		}

		partText = strings.TrimSpace(partText)
		if partText != "" && !isOnlyPunctuation(partText) {
			result = append(result, Segment{
				ID:    currentID,
				Start: partStart,
				End:   partEnd,
				Text:  partText,
			})
			currentID++
		}

		currentPos = splitPoint + 1
	}

	// Thêm phần còn lại
	if currentPos < len(text) {
		remainingText := strings.TrimSpace(text[currentPos:])
		if remainingText != "" && !isOnlyPunctuation(remainingText) {
			processedRunes := len([]rune(text[:currentPos]))
			totalRuneCount := len([]rune(text))
			result = append(result, Segment{
				ID:    currentID,
				Start: startTime + (float64(processedRunes)/float64(totalRuneCount))*duration,
				End:   endTime,
				Text:  remainingText,
			})
		}
	}

	// Nếu không tách được, return segment gốc
	if len(result) == 0 {
		return []Segment{segment}
	}

	return result
}

// splitByLength tách segment dựa trên độ dài text
func splitByLength(segment Segment, startID int) []Segment {
	var result []Segment

	text := segment.Text
	startTime := segment.Start
	endTime := segment.End
	duration := endTime - startTime

	// Sử dụng số ký tự (rune) thay vì byte
	runeText := []rune(text)
	totalRunes := len(runeText)

	// Nếu text quá ngắn, không tách
	if totalRunes <= 25 {
		return []Segment{segment}
	}

	// Tách thành các phần có độ dài tối đa 25 ký tự
	maxRunesPerSegment := 25
	numSegments := (totalRunes + maxRunesPerSegment - 1) / maxRunesPerSegment

	for i := 0; i < numSegments; i++ {
		startRune := i * maxRunesPerSegment
		endRune := startRune + maxRunesPerSegment
		if endRune > totalRunes {
			endRune = totalRunes
		}

		// Tính thời gian dựa trên số ký tự
		segmentStart := startTime + (float64(startRune)/float64(totalRunes))*duration
		segmentEnd := startTime + (float64(endRune)/float64(totalRunes))*duration

		if segmentEnd > endTime {
			segmentEnd = endTime
		}

		segmentText := string(runeText[startRune:endRune])
		segmentText = strings.TrimSpace(segmentText)

		if segmentText != "" && !isOnlyPunctuation(segmentText) {
			result = append(result, Segment{
				ID:    startID + i,
				Start: segmentStart,
				End:   segmentEnd,
				Text:  segmentText,
			})
		}
	}

	// Nếu không tách được, return segment gốc
	if len(result) == 0 {
		return []Segment{segment}
	}

	return result
}

// findAllOccurrences tìm tất cả vị trí xuất hiện của một ký tự
func findAllOccurrences(text, char string) []int {
	var positions []int
	for i, r := range text {
		if string(r) == char {
			positions = append(positions, i)
		}
	}
	return positions
}

// sortAndDeduplicate sắp xếp và loại bỏ trùng lặp
func sortAndDeduplicate(positions []int) []int {
	if len(positions) == 0 {
		return positions
	}

	// Sắp xếp
	for i := 0; i < len(positions)-1; i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[i] > positions[j] {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}

	// Loại bỏ trùng lặp
	var result []int
	result = append(result, positions[0])
	for i := 1; i < len(positions); i++ {
		if positions[i] != positions[i-1] {
			result = append(result, positions[i])
		}
	}

	return result
}

func TranscribeWhisperOpenAI(filePath, apiKey string) (string, []Segment, *WhisperUsage, error) {

	url := "https://api.openai.com/v1/audio/transcriptions"

	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Thêm file
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return "", nil, nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", nil, nil, err
	}

	// Thêm model + format
	writer.WriteField("model", "whisper-1")
	writer.WriteField("response_format", "verbose_json") // 🔥 Đổi thành verbose_json
	writer.Close()

	// Tạo request
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return "", nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Gửi request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()

	// Đọc response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, nil, fmt.Errorf("OpenAI error: %s", string(body))
	}

	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse whisper response: %v", err)
	}

	// Giữ nguyên segments gốc từ Whisper để đảm bảo thời gian chính xác
	// Chỉ clean text, không thay đổi timing
	var cleanedSegments []Segment
	for _, segment := range whisperResp.Segments {
		cleanText := cleanTextThoroughly(segment.Text)
		if cleanText != "" {
			cleanedSegments = append(cleanedSegments, Segment{
				ID:    segment.ID,
				Start: segment.Start,
				End:   segment.End,
				Text:  cleanText,
			})
		}
	}

	// Tạo lại text từ các segment đã clean
	var splitText strings.Builder
	for _, segment := range cleanedSegments {
		splitText.WriteString(segment.Text)
		splitText.WriteString(" ")
	}

	return strings.TrimSpace(splitText.String()), cleanedSegments, whisperResp.Usage, nil
}
