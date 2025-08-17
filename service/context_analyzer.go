package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ContextAnalysisResult kết quả phân tích ngữ cảnh
type ContextAnalysisResult struct {
	Characters   []string               `json:"characters"`
	Relationship string                 `json:"relationship"`
	PronounRules map[string]PronounRule `json:"pronoun_rules"`
	Glossary     map[string]string      `json:"glossary"`
	AnalysisTime time.Duration          `json:"analysis_time"`
}

// PronounRule quy tắc xưng hô cho một nhân vật
type PronounRule struct {
	Self        string            `json:"self"`         // Cách nhân vật tự xưng
	RefersTo    map[string]string `json:"refers_to"`    // Cách nhân vật gọi người khác
	FormalLevel string            `json:"formal_level"` // Mức độ trang trọng (formal/informal)
}

// ContextAnalyzer phân tích ngữ cảnh SRT
type ContextAnalyzer struct {
	apiKey    string
	modelName string
	timeout   time.Duration
}

// NewContextAnalyzer tạo context analyzer mới
func NewContextAnalyzer(apiKey, modelName string) *ContextAnalyzer {
	return &ContextAnalyzer{
		apiKey:    apiKey,
		modelName: modelName,
		timeout:   30 * time.Second, // 30s timeout cho context analysis
	}
}

// AnalyzeSRTContext phân tích ngữ cảnh của file SRT
func (ca *ContextAnalyzer) AnalyzeSRTContext(srtFilePath, targetLanguage string) (*ContextAnalysisResult, error) {
	startTime := time.Now()
	log.Printf("🔍 [CONTEXT ANALYZER] Bắt đầu phân tích ngữ cảnh cho %s", srtFilePath)

	// Đọc file SRT
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Tạo prompt cho context analysis
	prompt := ca.createContextAnalysisPrompt(string(srtContent), targetLanguage)

	// Gọi API để phân tích
	var result *ContextAnalysisResult
	if strings.Contains(strings.ToLower(ca.modelName), "gpt") {
		result, err = ca.callGPTForContextAnalysis(prompt)
	} else {
		result, err = ca.callGeminiForContextAnalysis(prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("context analysis failed: %v", err)
	}

	result.AnalysisTime = time.Since(startTime)
	log.Printf("✅ [CONTEXT ANALYZER] Phân tích ngữ cảnh hoàn thành trong %v", result.AnalysisTime)
	log.Printf("📊 [CONTEXT ANALYZER] Kết quả: %d nhân vật, %d thuật ngữ", len(result.Characters), len(result.Glossary))

	return result, nil
}

// createContextAnalysisPrompt tạo prompt cho context analysis
func (ca *ContextAnalyzer) createContextAnalysisPrompt(srtContent, targetLanguage string) string {
	languageMap := map[string]string{
		"vi": "Tiếng Việt", "en": "Tiếng Anh", "ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn", "zh": "Tiếng Trung", "fr": "Tiếng Pháp",
		"de": "Tiếng Đức", "es": "Tiếng Tây Ban Nha",
	}

	languageName := languageMap[targetLanguage]
	if languageName == "" {
		languageName = "Tiếng Việt"
	}

	return fmt.Sprintf(`Tôi sẽ cung cấp cho bạn toàn bộ nội dung của một file phụ đề. Nhiệm vụ của bạn KHÔNG PHẢI LÀ DỊCH. Thay vào đó, hãy phân tích và trả về cho tôi một bản tóm tắt ngữ cảnh dưới dạng JSON, bao gồm:

1. Liệt kê tất cả các nhân vật có thể có trong đoạn hội thoại.
2. Dựa vào nội dung, phán đoán mối quan hệ giữa họ (ví dụ: đồng nghiệp, bạn bè, thầy trò, người yêu...).
3. Đề xuất một bộ quy tắc xưng hô nhất quán bằng %s cho các nhân vật đó.
4. Liệt kê 5-10 thuật ngữ hoặc tên riêng quan trọng có vẻ cần được dịch thống nhất.

LƯU Ý QUAN TRỌNG:
- Chỉ trả về JSON, không có text nào khác
- Đảm bảo JSON hợp lệ và có thể parse được
- Quy tắc xưng hô phải nhất quán và phù hợp với văn hóa %s
- Tên nhân vật và địa danh nên giữ nguyên hoặc chuyển sang dạng hán việt nếu phù hợp

Đây là nội dung phụ đề:
%s

Trả về JSON theo format sau:
{
  "characters": ["tên_nhân_vật_1", "tên_nhân_vật_2"],
  "relationship": "mô_tả_mối_quan_hệ",
  "pronoun_rules": {
    "tên_nhân_vật_1": {
      "self": "cách_tự_xưng",
      "refers_to": {
        "tên_nhân_vật_2": "cách_gọi_người_khác"
      },
      "formal_level": "formal/informal"
    }
  },
  "glossary": {
    "thuật_ngữ_gốc": "dịch_thống_nhất"
  }
}`, languageName, languageName, srtContent)
}

// callGPTForContextAnalysis gọi GPT API để phân tích ngữ cảnh
func (ca *ContextAnalyzer) callGPTForContextAnalysis(prompt string) (*ContextAnalysisResult, error) {
	url := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model":       ca.modelName,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.1, // Thấp để đảm bảo kết quả nhất quán
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+ca.apiKey)
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), ca.timeout)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GPT API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GPT API error: %s - %s", resp.Status, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse GPT response: %v", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in GPT response")
	}

	content := response.Choices[0].Message.Content
	return ca.parseContextAnalysisResponse(content)
}

// callGeminiForContextAnalysis gọi Gemini API để phân tích ngữ cảnh
func (ca *ContextAnalyzer) callGeminiForContextAnalysis(prompt string) (*ContextAnalysisResult, error) {
	// Sử dụng service Gemini có sẵn
	content, err := GenerateWithGemini(prompt, ca.apiKey, ca.modelName)
	if err != nil {
		return nil, fmt.Errorf("Gemini API request failed: %v", err)
	}

	return ca.parseContextAnalysisResponse(content)
}

// parseContextAnalysisResponse parse response từ API thành struct
func (ca *ContextAnalyzer) parseContextAnalysisResponse(content string) (*ContextAnalysisResult, error) {
	// Clean up content - tìm JSON trong response
	content = strings.TrimSpace(content)

	// Tìm JSON object trong response
	startIndex := strings.Index(content, "{")
	endIndex := strings.LastIndex(content, "}")

	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return nil, fmt.Errorf("invalid JSON response format")
	}

	jsonContent := content[startIndex : endIndex+1]

	var result ContextAnalysisResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return &result, nil
}

// GenerateContextAwarePrompt tạo prompt với context awareness
func (ca *ContextAnalyzer) GenerateContextAwarePrompt(contextResult *ContextAnalysisResult, targetLanguage string) string {
	languageMap := map[string]string{
		"vi": "Tiếng Việt", "en": "Tiếng Anh", "ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn", "zh": "Tiếng Trung", "fr": "Tiếng Pháp",
		"de": "Tiếng Đức", "es": "Tiếng Tây Ban Nha",
	}

	languageName := languageMap[targetLanguage]
	if languageName == "" {
		languageName = "Tiếng Việt"
	}

	// Tạo context rules string
	var contextRules strings.Builder
	contextRules.WriteString(fmt.Sprintf("NGỮ CẢNH ĐÃ PHÂN TÍCH:\n"))
	contextRules.WriteString(fmt.Sprintf("- Ngôn ngữ đích: %s\n", languageName))
	contextRules.WriteString(fmt.Sprintf("- Mối quan hệ: %s\n", contextResult.Relationship))

	if len(contextResult.Characters) > 0 {
		contextRules.WriteString("- Nhân vật: " + strings.Join(contextResult.Characters, ", ") + "\n")
	}

	contextRules.WriteString("- QUY TẮC XƯNG HÔ:\n")
	for char, rules := range contextResult.PronounRules {
		contextRules.WriteString(fmt.Sprintf("  + %s: tự xưng '%s', formal_level: %s\n", char, rules.Self, rules.FormalLevel))
		for target, pronoun := range rules.RefersTo {
			contextRules.WriteString(fmt.Sprintf("    gọi %s: '%s'\n", target, pronoun))
		}
	}

	if len(contextResult.Glossary) > 0 {
		contextRules.WriteString("- THUẬT NGỮ THỐNG NHẤT:\n")
		for term, translation := range contextResult.Glossary {
			contextRules.WriteString(fmt.Sprintf("  + %s → %s\n", term, translation))
		}
	}

	return fmt.Sprintf(`Hãy dịch phần SRT sau sang %s, tối ưu hóa đặc biệt cho Text-to-Speech (TTS).

%s

TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:
QUY TẮC 1: TIMESTAMP VÀ SỐ THỨ TỰ LÀ BẤT BIẾN
Giữ nguyên 100%% số thứ tự và dòng thời gian (timestamps) từ file gốc.
TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian.

QUY TẮC 2: ƯU TIÊN HÀNG ĐẦU LÀ ĐỘ DÀI CÂU DỊCH
Ngắn gọn là Vua: Câu dịch phải đủ ngắn để đọc xong trong khoảng thời gian của timestamp.
Áp dụng quy tắc Ký tự/Giây (CPS): Cố gắng giữ cho câu dịch không vượt quá 17 ký tự cho mỗi giây thời lượng.

QUY TẮC 3: ÁP DỤNG QUY TẮC XƯNG HÔ ĐÃ PHÂN TÍCH
TUYỆT ĐỐI TUÂN THỦ quy tắc xưng hô đã được phân tích ở trên.
Đảm bảo tính nhất quán trong cách xưng hô giữa các nhân vật.

QUY TẮC 4: SỬ DỤNG THUẬT NGỮ THỐNG NHẤT
Áp dụng chính xác các thuật ngữ đã được định nghĩa trong phần phân tích ngữ cảnh.

QUY TẮC 5: QUẢ TRẢ VỀ LUÔN LÀ ĐỊNH DẠNG SRT
Kết quả trả về chỉ là nội dung file srt, không thêm bất kỳ ghi chú hay giải thích nào khác.
QUY TẮC 6: XỬ LÝ ĐẠI TỪ NHÂN XƯNG CÓ LỰA CHỌN
Khi quy tắc xưng hô cung cấp một lựa chọn (ví dụ: 'thầy/cô', 'tôi/em' ...), bạn BẮT BUỘC PHẢI CHỌN MỘT phương án phù hợp nhất với ngữ cảnh của câu thoại đó. TUYỆT ĐỐI KHÔNG được viết cả hai lựa chọn cách nhau bằng dấu gạch chéo trong câu dịch.

Phần SRT cần dịch:
{{SRT_CONTENT}}`, languageName, contextRules.String())
}
