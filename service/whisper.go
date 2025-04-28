package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type Segment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}
type WhisperResponse struct {
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
}

func TranscribeWhisperOpenAI(filePath, apiKey string) (string, []Segment, error) {

	url := "https://api.openai.com/v1/audio/transcriptions"

	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Thêm file
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return "", nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", nil, err
	}

	// Thêm model + format
	writer.WriteField("model", "whisper-1")
	writer.WriteField("response_format", "verbose_json") // 🔥 Đổi thành verbose_json
	writer.Close()

	// Tạo request
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Gửi request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	// Đọc response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("OpenAI error: %s", string(body))
	}

	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse whisper response: %v", err)
	}

	return whisperResp.Text, whisperResp.Segments, nil
}
