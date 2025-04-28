package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GeminiRequest định nghĩa cấu trúc của request gửi tới Gemini API
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content định nghĩa nội dung gửi tới Gemini API (text, image, v.v.)
type Content struct {
	Parts []Part `json:"parts"`
}

// Part định nghĩa một phần nội dung (text, image, v.v.)
type Part struct {
	Text string `json:"text"`
}

// GeminiResponse định nghĩa cấu trúc phản hồi từ Gemini API
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate chứa nội dung trả về từ Gemini
type Candidate struct {
	Content Content `json:"content"`
}

// GenerateWithGemini gửi text tới Gemini API và nhận phản hồi (ví dụ: caption, dịch, hoặc phân tích)
func GenerateWithGemini(prompt, apiKey string) (string, error) {
	// Endpoint của Gemini API (Google AI API)
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent"

	// Tạo payload cho request
	requestBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	// Chuyển payload thành JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini request: %v", err)
	}

	// Tạo HTTP request
	req, err := http.NewRequest("POST", url+"?key="+apiKey, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Gửi request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send Gemini request: %v", err)
	}
	defer resp.Body.Close()

	// Đọc phản hồi
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Gemini response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API error: %s", string(body))
	}

	// Parse phản hồi
	var geminiResp GeminiResponse
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %v", err)
	}

	// Kiểm tra nếu không có candidates
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content returned from Gemini API")
	}

	// Lấy text từ phản hồi
	result := geminiResp.Candidates[0].Content.Parts[0].Text
	return result, nil
}

func TranslateSegmentsWithGemini(segmentsJSON string, apiKey string) ([]Segment, error) {
	// Parse JSON segments
	var segments []Segment
	err := json.Unmarshal([]byte(segmentsJSON), &segments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse segments JSON: %v", err)
	}

	// Nếu không có segments, trả về ngay
	if len(segments) == 0 {
		return segments, nil
	}

	// Gộp tất cả text thành một prompt duy nhất
	var promptBuilder strings.Builder
	promptBuilder.WriteString("Dịch các câu sau sang tiếng Việt tự nhiên, phù hợp với ngữ cảnh phụ đề video. Trả về danh sách các câu đã dịch theo định dạng:\n")
	promptBuilder.WriteString("1. Câu dịch 1\n2. Câu dịch 2\n3. Câu dịch 3\n...\n\n")
	promptBuilder.WriteString("Danh sách các câu cần dịch:\n")
	for i, segment := range segments {
		fmt.Fprintf(&promptBuilder, "%d. %s\n", i+1, segment.Text)
	}

	// Gọi Gemini API một lần duy nhất
	translatedText, err := GenerateWithGemini(promptBuilder.String(), apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to translate segments: %v", err)
	}

	// Parse phản hồi từ Gemini
	translatedLines := strings.Split(translatedText, "\n")
	if len(translatedLines) < len(segments) {
		return nil, fmt.Errorf("incomplete translation: expected %d lines, got %d", len(segments), len(translatedLines))
	}

	// Cập nhật text đã dịch vào segments
	for i, line := range translatedLines {
		// Bỏ qua dòng trống hoặc không đúng định dạng
		if i >= len(segments) || strings.TrimSpace(line) == "" {
			continue
		}

		// Loại bỏ số thứ tự (ví dụ: "1. Câu dịch 1" -> "Câu dịch 1")
		parts := strings.SplitN(line, ". ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid translation format at line %d: %s", i+1, line)
		}

		// Cập nhật text đã dịch
		segments[i].Text = strings.TrimSpace(parts[1])
	}

	return segments, nil
}
