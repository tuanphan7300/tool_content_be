package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type GPTRequest struct {
	Model    string       `json:"model"`
	Messages []GPTMessage `json:"messages"`
}

type GPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GPTResponse struct {
	Choices []struct {
		Message GPTMessage `json:"message"`
	} `json:"choices"`
}

func GenerateSuggestion(transcript, apiKey, targetLanguage string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	// Map language codes to language names
	languageMap := map[string]string{
		"vi": "Tiếng Việt",
		"en": "Tiếng Anh",
		"ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn",
		"zh": "Tiếng Trung",
		"fr": "Tiếng Pháp",
		"de": "Tiếng Đức",
		"es": "Tiếng Tây Ban Nha",
	}

	languageName := languageMap[targetLanguage]
	if languageName == "" {
		languageName = "Tiếng Việt" // Default fallback
	}

	prompt := fmt.Sprintf(`Bạn là một chuyên gia content Tiktok. Dựa vào nội dung sau:
"%s"

Hãy viết (TẤT CẢ BẰNG %s):
- 3 caption hấp dẫn bằng %s, mỗi caption không quá 100 ký tự.
- 5-10 hashtag liên quan, viết theo định dạng #abc #xyz (hashtag có thể có tiếng Anh)

LƯU Ý: Caption phải bằng %s, chỉ hashtag có thể có tiếng Anh.`, transcript, languageName, languageName, languageName)

	reqBody := GPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []GPTMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API Error: %s", string(respBody))
	}

	var gptResp GPTResponse
	json.Unmarshal(respBody, &gptResp)

	if len(gptResp.Choices) > 0 {
		return gptResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("No response from GPT")
}

func TranslateTranscript(transcript, apiKey string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	prompt := fmt.Sprintf(`Dịch nội dung sau từ tiếng Trung sang tiếng Việt một cách tự nhiên, dễ hiểu:

"%s"`, transcript)

	reqBody := GPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []GPTMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API Error: %s", string(respBody))
	}

	var gptResp GPTResponse
	json.Unmarshal(respBody, &gptResp)

	if len(gptResp.Choices) > 0 {
		return gptResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("No response from GPT")
}

func GenerateTikTokOptimization(prompt, apiKey string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	reqBody := GPTRequest{
		Model: "gpt-3.5-turbo",
		Messages: []GPTMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API Error: %s", string(respBody))
	}

	var gptResp GPTResponse
	json.Unmarshal(respBody, &gptResp)

	if len(gptResp.Choices) > 0 {
		return gptResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("No response from GPT")
}

// GenerateCaptionWithService wrapper function that uses service_config to determine which service to use
func GenerateCaptionWithService(transcript, apiKey, serviceName, modelAPIName, targetLanguage string) (string, error) {
	// Currently only GPT-3.5-turbo is supported for caption_generation
	if serviceName == "gpt_3.5_turbo" {
		return GenerateSuggestion(transcript, apiKey, targetLanguage)
	}

	// For future services, we can add more conditions here
	// Example: if serviceName == "claude" { return GenerateCaptionWithClaude(transcript, apiKey, modelAPIName) }

	return "", fmt.Errorf("unsupported caption generation service: %s", serviceName)
}

func TranslateSRTWithGPT(srtFilePath, apiKey, modelName, targetLanguage string) (string, error) {
	// Đọc file SRT
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Map language codes to language names
	languageMap := map[string]string{
		"vi": "Tiếng Việt",
		"en": "Tiếng Anh",
		"ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn",
		"zh": "Tiếng Trung",
		"fr": "Tiếng Pháp",
		"de": "Tiếng Đức",
		"es": "Tiếng Tây Ban Nha",
	}

	languageName := languageMap[targetLanguage]
	if languageName == "" {
		languageName = "Tiếng Việt" // Default fallback
	}

	url := "https://api.openai.com/v1/chat/completions"

	prompt := fmt.Sprintf(`Hãy dịch file SRT sang %s.

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:
- Giữ nguyên 100%% số thứ tự và dòng thời gian (timestamps)
- Dịch tự nhiên, truyền cảm, phù hợp với văn nói
- Rút gọn các câu quá dài để khớp với thời gian hiển thị
- Đối với nội dung hoạt hình/truyện: dịch phù hợp với ngữ cảnh, giữ nguyên tên nhân vật nếu cần
- Chỉ trả về nội dung SRT đã dịch, không thêm giải thích
- Tên nhân vật hãy ưu tiên để dạng hán-việt. ví dụ: Nhị Cẩu, Tiểu Đản, Hồ Nam, Bắc Kinh

Nội dung SRT cần dịch:
%s`, languageName, string(srtContent))

	reqBody := GPTRequest{
		Model: modelName,
		Messages: []GPTMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API Error: %s", string(respBody))
	}

	var gptResp GPTResponse
	json.Unmarshal(respBody, &gptResp)

	if len(gptResp.Choices) > 0 {
		return gptResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("No response from GPT")
}

func EstimateGPTTokens(srtFilePath, modelName string) (int, error) {
	// Đọc file SRT
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Ước tính tokens dựa trên số ký tự
	// GPT thường sử dụng ~4 ký tự = 1 token
	contentLength := len(string(srtContent))
	estimatedTokens := contentLength / 4

	// Thêm tokens cho prompt
	promptTokens := 200 // Ước tính tokens cho prompt

	return estimatedTokens + promptTokens, nil
}
