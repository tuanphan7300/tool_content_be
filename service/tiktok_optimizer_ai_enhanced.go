package service

import (
	"encoding/json"
	"fmt"
)

// AITikTokOptimizer sử dụng AI để tạo nội dung đa ngôn ngữ và trending
type AITikTokOptimizer struct {
	apiKey string
}

// NewAITikTokOptimizer tạo instance mới
func NewAITikTokOptimizer(apiKey string) *AITikTokOptimizer {
	return &AITikTokOptimizer{
		apiKey: apiKey,
	}
}

// GenerateLocalizedContent tạo nội dung đa ngôn ngữ bằng AI
func (a *AITikTokOptimizer) GenerateLocalizedContent(transcript, category, targetLanguage string, duration float64) (*LocalizedTikTokContent, error) {
	// Tạo prompt cho AI để generate nội dung đa ngôn ngữ
	prompt := a.createLocalizationPrompt(transcript, category, targetLanguage, duration)

	// Gọi AI để tạo nội dung
	aiResponse, err := GenerateTikTokOptimization(prompt, a.apiKey)
	if err != nil {
		return nil, fmt.Errorf("AI localization failed: %v", err)
	}

	// Parse AI response
	var aiResult map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &aiResult); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	// Convert to LocalizedTikTokContent
	content := &LocalizedTikTokContent{
		Language: targetLanguage,
	}

	// Parse optimization tips
	if tips, ok := aiResult["optimization_tips"].([]interface{}); ok {
		for _, tip := range tips {
			if tipStr, ok := tip.(string); ok {
				content.OptimizationTips = append(content.OptimizationTips, tipStr)
			}
		}
	}

	// Parse engagement prompts
	if prompts, ok := aiResult["engagement_prompts"].([]interface{}); ok {
		for _, prompt := range prompts {
			if promptStr, ok := prompt.(string); ok {
				content.EngagementPrompts = append(content.EngagementPrompts, promptStr)
			}
		}
	}

	// Parse call to action
	if cta, ok := aiResult["call_to_action"].(string); ok {
		content.CallToAction = cta
	}

	// Parse suggested caption
	if caption, ok := aiResult["suggested_caption"].(string); ok {
		content.SuggestedCaption = caption
	}

	// Parse trending hashtags
	if hashtags, ok := aiResult["trending_hashtags"].([]interface{}); ok {
		for _, hashtag := range hashtags {
			if hashtagStr, ok := hashtag.(string); ok {
				content.TrendingHashtags = append(content.TrendingHashtags, hashtagStr)
			}
		}
	}

	// Parse trending topics
	if topics, ok := aiResult["trending_topics"].([]interface{}); ok {
		for _, topic := range topics {
			if topicStr, ok := topic.(string); ok {
				content.TrendingTopics = append(content.TrendingTopics, topicStr)
			}
		}
	}

	return content, nil
}

// createLocalizationPrompt tạo prompt cho AI để generate nội dung đa ngôn ngữ
func (a *AITikTokOptimizer) createLocalizationPrompt(transcript, category, targetLanguage string, duration float64) string {
	languageName := getLanguageNameForAI(targetLanguage)

	prompt := fmt.Sprintf(`Bạn là chuyên gia TikTok với 10+ năm kinh nghiệm. Tạo nội dung tối ưu cho TikTok bằng %s.

THÔNG TIN VIDEO:
- Transcript: %s
- Loại nội dung: %s
- Thời lượng: %.1f giây
- Ngôn ngữ mục tiêu: %s

YÊU CẦU TẠO NỘI DUNG (TRẢ VỀ JSON):

1. OPTIMIZATION_TIPS: 3-5 tips cụ thể để tối ưu video này (bằng %s, tập trung vào xu hướng 2024-2025)
2. ENGAGEMENT_PROMPTS: 3 câu hỏi để tăng tương tác (bằng %s, phù hợp với văn hóa %s)
3. CALL_TO_ACTION: Gợi ý CTA hiệu quả (bằng %s, phù hợp với %s)
4. SUGGESTED_CAPTION: Caption tối ưu cho TikTok (bằng %s, tối đa 150 ký tự, có hook mạnh)
5. TRENDING_HASHTAGS: 8-10 hashtags trending hiện tại phù hợp với nội dung (có thể mix ngôn ngữ)
6. TRENDING_TOPICS: 3 chủ đề trending hiện tại phù hợp với %s

LƯU Ý QUAN TRỌNG:
- Tất cả nội dung phải bằng %s (trừ hashtags có thể mix)
- Tập trung vào xu hướng hiện tại (2024-2025)
- Sử dụng hashtags thực sự trending
- Caption phải có hook mạnh và call-to-action
- Phù hợp với văn hóa và phong cách của %s
- Engagement prompts phải tự nhiên và dễ trả lời

TRẢ VỀ CHỈ JSON, KHÔNG CÓ TEXT KHÁC:
{
  "optimization_tips": ["Tip 1", "Tip 2", "Tip 3"],
  "engagement_prompts": ["Câu hỏi 1?", "Câu hỏi 2?", "Câu hỏi 3?"],
  "call_to_action": "CTA hiệu quả...",
  "suggested_caption": "Caption tối ưu...",
  "trending_hashtags": ["#hashtag1", "#hashtag2"],
  "trending_topics": ["Topic 1", "Topic 2", "Topic 3"]
}`, languageName, transcript, category, duration, languageName, languageName, languageName, languageName, languageName, languageName, languageName, languageName, languageName, languageName)

	return prompt
}

// getLanguageNameForAI trả về tên ngôn ngữ cho AI prompt
func getLanguageNameForAI(language string) string {
	languageNames := map[string]string{
		"vi":  "Tiếng Việt",
		"en":  "Tiếng Anh",
		"ja":  "Tiếng Nhật",
		"ko":  "Tiếng Hàn",
		"zh":  "Tiếng Trung",
		"fr":  "Tiếng Pháp",
		"de":  "Tiếng Đức",
		"es":  "Tiếng Tây Ban Nha",
		"it":  "Tiếng Ý",
		"pt":  "Tiếng Bồ Đào Nha",
		"ru":  "Tiếng Nga",
		"ar":  "Tiếng Ả Rập",
		"hi":  "Tiếng Hindi",
		"th":  "Tiếng Thái",
		"id":  "Tiếng Indonesia",
		"ms":  "Tiếng Malaysia",
		"tr":  "Tiếng Thổ Nhĩ Kỳ",
		"pl":  "Tiếng Ba Lan",
		"nl":  "Tiếng Hà Lan",
		"sv":  "Tiếng Thụy Điển",
		"da":  "Tiếng Đan Mạch",
		"no":  "Tiếng Na Uy",
		"fi":  "Tiếng Phần Lan",
		"cs":  "Tiếng Séc",
		"hu":  "Tiếng Hungary",
		"ro":  "Tiếng Romania",
		"bg":  "Tiếng Bulgaria",
		"hr":  "Tiếng Croatia",
		"sk":  "Tiếng Slovakia",
		"sl":  "Tiếng Slovenia",
		"et":  "Tiếng Estonia",
		"lv":  "Tiếng Latvia",
		"lt":  "Tiếng Lithuania",
		"mt":  "Tiếng Malta",
		"el":  "Tiếng Hy Lạp",
		"he":  "Tiếng Hebrew",
		"fa":  "Tiếng Ba Tư",
		"ur":  "Tiếng Urdu",
		"bn":  "Tiếng Bengali",
		"ta":  "Tiếng Tamil",
		"te":  "Tiếng Telugu",
		"ml":  "Tiếng Malayalam",
		"kn":  "Tiếng Kannada",
		"gu":  "Tiếng Gujarati",
		"pa":  "Tiếng Punjabi",
		"or":  "Tiếng Odia",
		"as":  "Tiếng Assamese",
		"ne":  "Tiếng Nepal",
		"si":  "Tiếng Sinhala",
		"my":  "Tiếng Myanmar",
		"km":  "Tiếng Khmer",
		"lo":  "Tiếng Lào",
		"mn":  "Tiếng Mông Cổ",
		"ka":  "Tiếng Georgia",
		"am":  "Tiếng Amharic",
		"sw":  "Tiếng Swahili",
		"yo":  "Tiếng Yoruba",
		"ig":  "Tiếng Igbo",
		"ha":  "Tiếng Hausa",
		"zu":  "Tiếng Zulu",
		"xh":  "Tiếng Xhosa",
		"af":  "Tiếng Afrikaans",
		"is":  "Tiếng Iceland",
		"ga":  "Tiếng Ireland",
		"cy":  "Tiếng Wales",
		"eu":  "Tiếng Basque",
		"ca":  "Tiếng Catalan",
		"gl":  "Tiếng Galician",
		"sq":  "Tiếng Albania",
		"mk":  "Tiếng Macedonia",
		"sr":  "Tiếng Serbia",
		"bs":  "Tiếng Bosnia",
		"me":  "Tiếng Montenegro",
		"uk":  "Tiếng Ukraine",
		"be":  "Tiếng Belarus",
		"kk":  "Tiếng Kazakhstan",
		"ky":  "Tiếng Kyrgyz",
		"uz":  "Tiếng Uzbekistan",
		"tk":  "Tiếng Turkmen",
		"tg":  "Tiếng Tajik",
		"az":  "Tiếng Azerbaijan",
		"hy":  "Tiếng Armenia",
		"ab":  "Tiếng Abkhaz",
		"os":  "Tiếng Ossetian",
		"ce":  "Tiếng Chechen",
		"cv":  "Tiếng Chuvash",
		"tt":  "Tiếng Tatar",
		"ba":  "Tiếng Bashkir",
		"udm": "Tiếng Udmurt",
		"mhr": "Tiếng Mari",
		"mrj": "Tiếng Hill Mari",
		"myv": "Tiếng Erzya",
		"mdf": "Tiếng Moksha",
		"koi": "Tiếng Komi-Permyak",
		"kpv": "Tiếng Komi-Zyrian",
	}

	if name, exists := languageNames[language]; exists {
		return name
	}
	return "Tiếng Việt" // Default fallback
}

// LocalizedTikTokContent chứa nội dung đa ngôn ngữ được tạo bởi AI
type LocalizedTikTokContent struct {
	Language          string   `json:"language"`
	OptimizationTips  []string `json:"optimization_tips"`
	EngagementPrompts []string `json:"engagement_prompts"`
	CallToAction      string   `json:"call_to_action"`
	SuggestedCaption  string   `json:"suggested_caption"`
	TrendingHashtags  []string `json:"trending_hashtags"`
	TrendingTopics    []string `json:"trending_topics"`
}

// HybridTikTokOptimizer kết hợp AI và rule-based để tối ưu hiệu suất
type HybridTikTokOptimizer struct {
	aiOptimizer        *AITikTokOptimizer
	ruleOptimizer      *RuleBasedOptimizer
	useAI              bool
	supportedLanguages []string
	ruleBasedLanguages []string
	fallbackToRule     bool
}

// NewHybridTikTokOptimizer tạo instance mới
func NewHybridTikTokOptimizer(apiKey string, useAI bool) *HybridTikTokOptimizer {
	return &HybridTikTokOptimizer{
		aiOptimizer:        NewAITikTokOptimizer(apiKey),
		ruleOptimizer:      NewRuleBasedOptimizer(),
		useAI:              useAI,
		supportedLanguages: []string{"vi", "en", "ja", "ko", "zh", "fr", "de", "es", "it", "pt", "ru", "ar", "hi", "th", "id", "ms", "tr", "pl", "nl", "sv", "da", "no", "fi", "cs", "hu", "ro", "bg", "hr", "sk", "sl", "et", "lv", "lt", "mt", "el", "he", "fa", "ur", "bn", "ta", "te", "ml", "kn", "gu", "pa", "or", "as", "ne", "si", "my", "km", "lo", "mn", "ka", "am", "sw", "yo", "ig", "ha", "zu", "xh", "af", "is", "ga", "cy", "eu", "ca", "gl", "sq", "mk", "sr", "bs", "me", "uk", "be", "kk", "ky", "uz", "tk", "tg", "az", "hy", "ab", "os", "ce", "cv", "tt", "ba", "udm", "mhr", "mrj", "myv", "mdf", "koi", "kpv"},
		ruleBasedLanguages: []string{"vi", "en", "ja"},
		fallbackToRule:     true,
	}
}

// GenerateOptimizedContent tạo nội dung tối ưu với hybrid approach
func (h *HybridTikTokOptimizer) GenerateOptimizedContent(transcript, category, targetLanguage string, duration float64) (*LocalizedTikTokContent, error) {
	// Kiểm tra xem ngôn ngữ có được hỗ trợ không
	if !h.isLanguageSupported(targetLanguage) {
		// Fallback to Vietnamese
		targetLanguage = "vi"
	}

	// Nếu sử dụng AI hoặc ngôn ngữ không có trong rule-based
	if h.useAI || !h.isRuleBasedLanguage(targetLanguage) {
		return h.aiOptimizer.GenerateLocalizedContent(transcript, category, targetLanguage, duration)
	}

	// Sử dụng rule-based cho các ngôn ngữ được hỗ trợ
	return h.ruleOptimizer.GenerateLocalizedContent(transcript, category, targetLanguage, duration)
}

// isLanguageSupported kiểm tra xem ngôn ngữ có được hỗ trợ không
func (h *HybridTikTokOptimizer) isLanguageSupported(language string) bool {
	for _, lang := range h.supportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// isRuleBasedLanguage kiểm tra xem ngôn ngữ có trong rule-based không
func (h *HybridTikTokOptimizer) isRuleBasedLanguage(language string) bool {
	for _, lang := range h.ruleBasedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// RuleBasedOptimizer sử dụng rule-based cho các ngôn ngữ được hỗ trợ
type RuleBasedOptimizer struct{}

// NewRuleBasedOptimizer tạo instance mới
func NewRuleBasedOptimizer() *RuleBasedOptimizer {
	return &RuleBasedOptimizer{}
}

// GenerateLocalizedContent tạo nội dung rule-based
func (r *RuleBasedOptimizer) GenerateLocalizedContent(transcript, category, targetLanguage string, duration float64) (*LocalizedTikTokContent, error) {
	// Sử dụng hệ thống đa ngôn ngữ hiện tại
	content := &LocalizedTikTokContent{
		Language: targetLanguage,
	}

	// Lấy nội dung từ hệ thống đa ngôn ngữ
	content.OptimizationTips = getLocalizedTips(targetLanguage, category)
	content.EngagementPrompts = getLocalizedPrompts(targetLanguage, category)
	content.CallToAction = getLocalizedCTA(targetLanguage, category)
	content.SuggestedCaption = getLocalizedCaption(targetLanguage, category, transcript)
	content.TrendingHashtags = GetCategoryHashtags(category) // Sử dụng hashtags cố định
	content.TrendingTopics = GetTrendingTopics(category, "general")

	return content, nil
}
