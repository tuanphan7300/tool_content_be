package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	prompt := fmt.Sprintf(`Hãy dịch file SRT sang %s, tối ưu hóa đặc biệt cho Text-to-Speech (TTS).
Mục tiêu cuối cùng là bản dịch khi được đọc lên phải vừa vặn một cách tự nhiên trong khoảng thời gian cho phép, đồng thời phản ánh đúng sắc thái và mối quan hệ của nhân vật qua cách xưng hô.
TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:
QUY TẮC 1: TIMESTAMP VÀ SỐ THỨ TỰ LÀ BẤT BIẾN
Giữ nguyên 100% số thứ tự và dòng thời gian (timestamps) từ file gốc.
TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Đây là quy tắc quan trọng nhất.
QUY TẮC 2: ƯU TIÊN HÀNG ĐẦU LÀ ĐỘ DÀI CÂU DỊCH
Ngắn gọn là Vua: Câu dịch phải đủ ngắn để đọc xong trong khoảng thời gian của timestamp. Đây là ưu tiên cao hơn việc dịch đầy đủ từng chữ.
Chủ động cô đọng ý: Nắm bắt ý chính và diễn đạt lại một cách súc tích nhất có thể trong văn nói. Mạnh dạn loại bỏ các từ phụ không làm thay đổi ý nghĩa cốt lõi.
Áp dụng quy tắc Ký tự/Giây (CPS): Cố gắng giữ cho câu dịch không vượt quá 17 ký tự cho mỗi giây thời lượng.
Ví dụ: Nếu thời lượng là 2 giây, câu dịch nên dài khoảng 34 ký tự.
QUY TẮC 3: DỊCH TỰ NHIÊN, LINH HOẠT VỀ XƯNG HÔ
Ưu tiên sự tự nhiên và phù hợp với ngữ cảnh giao tiếp.
Về cách xưng hô:
Đối với ngôi thứ nhất, "tôi" là lựa chọn mặc định an toàn và phổ biến nhất trong các tình huống trang trọng hoặc với người lạ.
Tuy nhiên, khi bối cảnh là cuộc trò chuyện thân mật, suồng sã giữa bạn bè, người thân hoặc những người ngang hàng, hãy chủ động sử dụng các đại từ tự nhiên hơn như "tao - mày", "tớ - cậu", v.v., để giữ được sự chân thực của lời thoại.
Hạn chế sử dụng đại từ "ta" trừ khi bối cảnh thật sự đặc trưng (nhân vật là vua chúa, thần linh, hoặc có tính cách rất ngạo mạn).
Mục tiêu là làm cho lời thoại chân thực như người Việt đang nói chuyện, chứ không phải là một bản dịch máy móc.
KIỂM TRA CUỐI CÙNG:
Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn:
Không có dòng thời gian nào bị sai lệch.
Độ dài câu dịch hợp lý với thời gian hiển thị.
Cách xưng hô ("tôi", "tao", "tớ", "mày"...) tự nhiên và phù hợp với ngữ cảnh của đoạn hội thoại.
File SRT gốc:
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
		translatedContent := gptResp.Choices[0].Message.Content

		// Clean up the response - remove any extra text that might be added by GPT
		translatedContent = strings.TrimSpace(translatedContent)

		// Remove markdown code blocks if present
		translatedContent = strings.TrimPrefix(translatedContent, "```")
		translatedContent = strings.TrimSuffix(translatedContent, "```")
		translatedContent = strings.TrimSpace(translatedContent)

		// If GPT added any prefix or explanation, try to extract just the SRT content
		if strings.Contains(translatedContent, "1\n") {
			// Find the start of the SRT content
			startIndex := strings.Index(translatedContent, "1\n")
			if startIndex != -1 {
				translatedContent = translatedContent[startIndex:]
			}
		}

		// Remove any explanatory text that might be added after the SRT content
		endMarkers := []string{
			"\n\n**Giải thích",
			"\n\nGiải thích",
			"\n\n**",
			"\n\nTôi đã",
			"\n\nHy vọng",
			"\n\nBản dịch",
			"\n\n---",
			"\n\nNote:",
			"\n\nLưu ý:",
			"\n```",
			"```",
		}

		for _, marker := range endMarkers {
			if index := strings.Index(translatedContent, marker); index != -1 {
				translatedContent = strings.TrimSpace(translatedContent[:index])
				break
			}
		}

		// Final cleanup - ensure we only have valid SRT content
		lines := strings.Split(translatedContent, "\n")
		var cleanLines []string
		inSRTContent := false

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Start SRT content when we see a number
			if isNumeric(line) {
				inSRTContent = true
			}

			// Stop if we see explanatory text
			if inSRTContent && (strings.Contains(line, "**") ||
				strings.Contains(line, "Giải thích") ||
				strings.Contains(line, "Tôi đã") ||
				strings.Contains(line, "Hy vọng") ||
				strings.Contains(line, "Bản dịch") ||
				strings.Contains(line, "Note:") ||
				strings.Contains(line, "Lưu ý:") ||
				strings.Contains(line, "```")) {
				break
			}

			if inSRTContent {
				cleanLines = append(cleanLines, line)
			}
		}

		if len(cleanLines) > 0 {
			translatedContent = strings.Join(cleanLines, "\n")
		}

		return translatedContent, nil
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
