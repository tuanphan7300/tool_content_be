package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func GenerateSuggestion(transcript, apiKey string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	prompt := fmt.Sprintf(`Bạn là một chuyên gia content Tiktok. Dựa vào nội dung sau:
"%s"

Hãy viết (TẤT CẢ BẰNG TIẾNG VIỆT):
- 3 caption hấp dẫn bằng tiếng Việt, mỗi caption không quá 100 ký tự.
- 5-10 hashtag liên quan, viết theo định dạng #abc #xyz (hashtag có thể có tiếng Anh)

LƯU Ý: Caption phải bằng tiếng Việt, chỉ hashtag có thể có tiếng Anh.`, transcript)

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
