package service

import (
	"creator-tool-backend/config"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

// TikTokAnalysisResult represents the comprehensive analysis result
type TikTokAnalysisResult struct {
	HookScore         int      `json:"hook_score"`
	ViralPotential    int      `json:"viral_potential"`
	OptimizationTips  []string `json:"optimization_tips"`
	TrendingHashtags  []string `json:"trending_hashtags"`
	SuggestedCaption  string   `json:"suggested_caption"`
	BestPostingTime   string   `json:"best_posting_time"`
	EngagementPrompts []string `json:"engagement_prompts"`
	CallToAction      string   `json:"call_to_action"`
	ContentCategory   string   `json:"content_category"`
	TargetAudience    string   `json:"target_audience"`
	TrendingTopics    []string `json:"trending_topics"`
	VideoPacing       string   `json:"video_pacing"`
	ThumbnailTips     []string `json:"thumbnail_tips"`
	SoundSuggestions  []string `json:"sound_suggestions"`
	AnalysisMethod    string   `json:"analysis_method"` // "ai", "rule-based", "hybrid"
}

// GenerateAdvancedTikTokOptimization creates a comprehensive TikTok optimization analysis
func GenerateAdvancedTikTokOptimization(transcript, currentCaption, targetAudience, targetLanguage string, duration float64) (*TikTokAnalysisResult, error) {
	// Analyze content category and audience (rule-based)
	contentCategory := analyzeContentCategory(transcript, currentCaption)

	// Generate comprehensive analysis
	result := &TikTokAnalysisResult{
		ContentCategory: contentCategory,
		TargetAudience:  targetAudience,
	}

	// Hook Score Analysis (rule-based)
	result.HookScore = calculateHookScore(transcript, duration, contentCategory)

	// Viral Potential Analysis (rule-based)
	result.ViralPotential = calculateViralPotential(transcript, duration, contentCategory, targetAudience)

	// Video Pacing Analysis (rule-based)
	result.VideoPacing = analyzeVideoPacing(duration, contentCategory)

	// Thumbnail Tips (rule-based)
	result.ThumbnailTips = generateThumbnailTips(contentCategory, targetAudience, targetLanguage)

	// Sound Suggestions (rule-based)
	result.SoundSuggestions = generateSoundSuggestions(contentCategory, targetAudience, targetLanguage)

	// AI-powered analysis for trending and dynamic content
	aiAnalysis, err := generateAIAnalysis(transcript, currentCaption, contentCategory, targetAudience, targetLanguage, duration)
	if err != nil {
		// Fallback to rule-based if AI fails
		result.AnalysisMethod = "rule-based"
		result.OptimizationTips = generateOptimizationTips(transcript, duration, contentCategory, targetAudience, targetLanguage)
		result.TrendingHashtags = generateTrendingHashtags(contentCategory, targetAudience)
		result.SuggestedCaption = generateSuggestedCaption(transcript, currentCaption, contentCategory, targetAudience, targetLanguage)
		result.BestPostingTime = getBestPostingTime(targetAudience, contentCategory)
		result.EngagementPrompts = generateEngagementPrompts(contentCategory, targetAudience, targetLanguage)
		result.CallToAction = generateCallToAction(contentCategory, targetAudience, targetLanguage)
		result.TrendingTopics = getTrendingTopics(contentCategory, targetAudience)
	} else {
		// Use AI results
		result.AnalysisMethod = "hybrid"
		result.OptimizationTips = aiAnalysis.OptimizationTips
		result.TrendingHashtags = aiAnalysis.TrendingHashtags
		result.SuggestedCaption = aiAnalysis.SuggestedCaption
		result.BestPostingTime = aiAnalysis.BestPostingTime
		result.EngagementPrompts = aiAnalysis.EngagementPrompts
		result.CallToAction = aiAnalysis.CallToAction
		result.TrendingTopics = aiAnalysis.TrendingTopics

		// Enhance AI results with rule-based insights
		result.OptimizationTips = enhanceWithRuleBasedTips(result.OptimizationTips, transcript, duration, contentCategory)
		result.TrendingHashtags = enhanceWithRuleBasedHashtags(result.TrendingHashtags, contentCategory)
	}

	return result, nil
}

// generateAIAnalysis uses AI to analyze trending and dynamic content
func generateAIAnalysis(transcript, currentCaption, contentCategory, targetAudience, targetLanguage string, duration float64) (*TikTokAnalysisResult, error) {
	// Get language name for prompt
	languageName := getLanguageName(targetLanguage)

	// Get API key
	configg := config.InfaConfig{}
	configg.LoadConfig()
	apiKey := configg.ApiKey
	// Create comprehensive AI prompt
	prompt := fmt.Sprintf(`Bạn là chuyên gia TikTok với 10+ năm kinh nghiệm. Phân tích video này và đưa ra gợi ý tối ưu để viral trên TikTok.

THÔNG TIN VIDEO:
- Transcript: %s
- Caption hiện tại: %s
- Thời lượng: %.1f giây
- Loại nội dung: %s
- Đối tượng mục tiêu: %s
- Ngôn ngữ: %s

YÊU CẦU PHÂN TÍCH (TRẢ VỀ JSON):

1. OPTIMIZATION_TIPS: 3-5 tips cụ thể để tối ưu video này (bằng %s)
2. TRENDING_HASHTAGS: 8-10 hashtags trending hiện tại phù hợp với nội dung
3. SUGGESTED_CAPTION: Caption tối ưu cho TikTok (bằng %s, tối đa 150 ký tự)
4. BEST_POSTING_TIME: Thời gian đăng tốt nhất cho đối tượng %s
5. ENGAGEMENT_PROMPTS: 3 câu hỏi để tăng tương tác (bằng %s)
6. CALL_TO_ACTION: Gợi ý CTA hiệu quả (bằng %s)
7. TRENDING_TOPICS: 3 chủ đề trending hiện tại phù hợp

LƯU Ý:
- Tập trung vào xu hướng hiện tại (2024-2025)
- Sử dụng hashtags thực sự trending
- Caption phải có hook mạnh và call-to-action
- Tất cả nội dung bằng %s

TRẢ VỀ CHỈ JSON, KHÔNG CÓ TEXT KHÁC:
{
  "optimization_tips": ["Tip 1", "Tip 2", "Tip 3"],
  "trending_hashtags": ["#hashtag1", "#hashtag2"],
  "suggested_caption": "Caption tối ưu...",
  "best_posting_time": "19:00-21:00",
  "engagement_prompts": ["Câu hỏi 1?", "Câu hỏi 2?"],
  "call_to_action": "Follow để xem thêm!",
  "trending_topics": ["Topic 1", "Topic 2", "Topic 3"]
}`, transcript, currentCaption, duration, contentCategory, targetAudience, languageName, languageName, languageName, targetAudience, languageName, languageName, languageName, languageName)

	// Call AI service
	aiResponse, err := GenerateTikTokOptimization(prompt, apiKey)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %v", err)
	}

	// Parse AI response
	var aiResult map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &aiResult); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	// Convert to TikTokAnalysisResult
	result := &TikTokAnalysisResult{}

	// Parse optimization tips
	if tips, ok := aiResult["optimization_tips"].([]interface{}); ok {
		for _, tip := range tips {
			if tipStr, ok := tip.(string); ok {
				result.OptimizationTips = append(result.OptimizationTips, tipStr)
			}
		}
	}

	// Parse trending hashtags
	if hashtags, ok := aiResult["trending_hashtags"].([]interface{}); ok {
		for _, hashtag := range hashtags {
			if hashtagStr, ok := hashtag.(string); ok {
				result.TrendingHashtags = append(result.TrendingHashtags, hashtagStr)
			}
		}
	}

	// Parse other fields
	if caption, ok := aiResult["suggested_caption"].(string); ok {
		result.SuggestedCaption = caption
	}
	if time, ok := aiResult["best_posting_time"].(string); ok {
		result.BestPostingTime = time
	}
	if prompts, ok := aiResult["engagement_prompts"].([]interface{}); ok {
		for _, prompt := range prompts {
			if promptStr, ok := prompt.(string); ok {
				result.EngagementPrompts = append(result.EngagementPrompts, promptStr)
			}
		}
	}
	if cta, ok := aiResult["call_to_action"].(string); ok {
		result.CallToAction = cta
	}
	if topics, ok := aiResult["trending_topics"].([]interface{}); ok {
		for _, topic := range topics {
			if topicStr, ok := topic.(string); ok {
				result.TrendingTopics = append(result.TrendingTopics, topicStr)
			}
		}
	}

	return result, nil
}

// enhanceWithRuleBasedTips combines AI tips with rule-based insights
func enhanceWithRuleBasedTips(aiTips []string, transcript string, duration float64, category string) []string {
	enhancedTips := make([]string, 0)

	// Add AI tips first
	enhancedTips = append(enhancedTips, aiTips...)

	// Add rule-based duration tips if not covered by AI
	hasDurationTip := false
	for _, tip := range aiTips {
		if strings.Contains(tip, "giây") || strings.Contains(tip, "thời lượng") || strings.Contains(tip, "duration") {
			hasDurationTip = true
			break
		}
	}

	if !hasDurationTip {
		if duration < 15 {
			enhancedTips = append(enhancedTips, "🎬 Video quá ngắn! Thêm nội dung để đạt 15-60 giây lý tưởng")
		} else if duration > 60 {
			enhancedTips = append(enhancedTips, "⏱️ Video quá dài! Cắt gọn xuống 60 giây để tăng engagement")
		}
	}

	// Add category-specific rule-based tips
	categoryTips := getCategorySpecificTips(category)
	for _, tip := range categoryTips {
		// Check if AI already covered this tip
		alreadyCovered := false
		for _, aiTip := range aiTips {
			if strings.Contains(strings.ToLower(aiTip), strings.ToLower(tip)) {
				alreadyCovered = true
				break
			}
		}
		if !alreadyCovered {
			enhancedTips = append(enhancedTips, tip)
		}
	}

	// Limit to 5 tips
	if len(enhancedTips) > 5 {
		enhancedTips = enhancedTips[:5]
	}

	return enhancedTips
}

// enhanceWithRuleBasedHashtags combines AI hashtags with rule-based ones
func enhanceWithRuleBasedHashtags(aiHashtags []string, category string) []string {
	enhancedHashtags := make([]string, 0)

	// Add AI hashtags first
	enhancedHashtags = append(enhancedHashtags, aiHashtags...)

	// Add rule-based category hashtags if not already present
	categoryHashtags := getCategoryHashtags(category)
	for _, hashtag := range categoryHashtags {
		alreadyPresent := false
		for _, aiHashtag := range aiHashtags {
			if strings.EqualFold(hashtag, aiHashtag) {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			enhancedHashtags = append(enhancedHashtags, hashtag)
		}
	}

	// Add essential TikTok hashtags
	essentialHashtags := []string{"#fyp", "#foryou", "#viral", "#trending", "#tiktok"}
	for _, hashtag := range essentialHashtags {
		alreadyPresent := false
		for _, existingHashtag := range enhancedHashtags {
			if strings.EqualFold(hashtag, existingHashtag) {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			enhancedHashtags = append(enhancedHashtags, hashtag)
		}
	}

	// Limit to 10 hashtags
	if len(enhancedHashtags) > 10 {
		enhancedHashtags = enhancedHashtags[:10]
	}

	return enhancedHashtags
}

// getLanguageName returns language name for prompts
func getLanguageName(language string) string {
	languageNames := map[string]string{
		"vi": "Tiếng Việt",
		"en": "Tiếng Anh",
		"ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn",
		"zh": "Tiếng Trung",
		"fr": "Tiếng Pháp",
		"de": "Tiếng Đức",
		"es": "Tiếng Tây Ban Nha",
	}

	if name, exists := languageNames[language]; exists {
		return name
	}
	return "Tiếng Việt"
}

// analyzeContentCategory determines the content category based on transcript and caption
func analyzeContentCategory(transcript, caption string) string {
	text := strings.ToLower(transcript + " " + caption)

	// Define category keywords
	categories := map[string][]string{
		"comedy":        {"hài", "vui", "cười", "funny", "joke", "humor", "comedy"},
		"education":     {"học", "dạy", "kiến thức", "education", "learn", "teach", "knowledge"},
		"lifestyle":     {"cuộc sống", "lifestyle", "daily", "routine", "life"},
		"food":          {"ăn", "nấu", "món", "food", "cook", "recipe", "delicious"},
		"fashion":       {"thời trang", "fashion", "style", "outfit", "clothes"},
		"beauty":        {"làm đẹp", "beauty", "makeup", "skincare", "cosmetic"},
		"fitness":       {"tập", "gym", "fitness", "workout", "exercise", "health"},
		"travel":        {"du lịch", "travel", "trip", "vacation", "destination"},
		"technology":    {"công nghệ", "tech", "technology", "gadget", "app"},
		"business":      {"kinh doanh", "business", "money", "investment", "entrepreneur"},
		"entertainment": {"giải trí", "entertainment", "music", "movie", "celebrity"},
		"news":          {"tin tức", "news", "current", "event", "update"},
	}

	// Count matches for each category
	scores := make(map[string]int)
	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				scores[category]++
			}
		}
	}

	// Find category with highest score
	maxScore := 0
	bestCategory := "general"
	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			bestCategory = category
		}
	}

	return bestCategory
}

// calculateHookScore evaluates the hook strength in the first 3 seconds
func calculateHookScore(transcript string, duration float64, category string) int {
	// Base score based on content category
	baseScores := map[string]int{
		"comedy":        85,
		"education":     75,
		"lifestyle":     70,
		"food":          80,
		"fashion":       75,
		"beauty":        80,
		"fitness":       75,
		"travel":        85,
		"technology":    70,
		"business":      65,
		"entertainment": 80,
		"news":          70,
		"general":       70,
	}

	baseScore := baseScores[category]

	// Adjust based on transcript length and content
	words := strings.Fields(transcript)
	wordCount := len(words)

	// Ideal hook length is 10-20 words
	if wordCount >= 10 && wordCount <= 20 {
		baseScore += 10
	} else if wordCount < 5 {
		baseScore -= 15
	} else if wordCount > 30 {
		baseScore -= 10
	}

	// Check for hook indicators
	hookIndicators := []string{"bạn có biết", "điều này sẽ", "shock", "không thể tin", "amazing", "incredible", "must see", "viral"}
	text := strings.ToLower(transcript)
	for _, indicator := range hookIndicators {
		if strings.Contains(text, indicator) {
			baseScore += 5
		}
	}

	// Ensure score is within 0-100 range
	if baseScore > 100 {
		baseScore = 100
	} else if baseScore < 0 {
		baseScore = 0
	}

	return baseScore
}

// calculateViralPotential calculates the viral potential score
func calculateViralPotential(transcript string, duration float64, category, audience string) int {
	baseScore := 70

	// Duration optimization (15-60 seconds is ideal for TikTok)
	if duration >= 15 && duration <= 60 {
		baseScore += 15
	} else if duration < 10 {
		baseScore -= 10
	} else if duration > 120 {
		baseScore -= 20
	}

	// Audience-specific adjustments
	audienceScores := map[string]int{
		"teenagers":     85,
		"young_adults":  80,
		"adults":        70,
		"business":      60,
		"entertainment": 90,
		"education":     75,
		"fitness":       80,
		"beauty":        85,
		"food":          90,
		"general":       70,
	}

	if score, exists := audienceScores[audience]; exists {
		baseScore = (baseScore + score) / 2
	}

	// Content category adjustments
	categoryScores := map[string]int{
		"comedy":        90,
		"education":     75,
		"lifestyle":     80,
		"food":          95,
		"fashion":       85,
		"beauty":        90,
		"fitness":       85,
		"travel":        90,
		"technology":    70,
		"business":      65,
		"entertainment": 95,
		"news":          70,
		"general":       70,
	}

	if score, exists := categoryScores[category]; exists {
		baseScore = (baseScore + score) / 2
	}

	// Check for viral keywords
	viralKeywords := []string{"trending", "viral", "popular", "hot", "trend", "fyp", "foryou"}
	text := strings.ToLower(transcript)
	for _, keyword := range viralKeywords {
		if strings.Contains(text, keyword) {
			baseScore += 3
		}
	}

	if baseScore > 100 {
		baseScore = 100
	}

	return baseScore
}

// generateOptimizationTips creates specific optimization tips (rule-based fallback)
func generateOptimizationTips(transcript string, duration float64, category, audience, language string) []string {
	tips := []string{}

	// Duration tips
	if duration < 15 {
		tips = append(tips, "🎬 Video quá ngắn! Thêm nội dung để đạt 15-60 giây lý tưởng")
	} else if duration > 60 {
		tips = append(tips, "⏱️ Video quá dài! Cắt gọn xuống 60 giây để tăng engagement")
	}

	// Category-specific tips
	categoryTips := getCategorySpecificTips(category)
	tips = append(tips, categoryTips...)

	// General optimization tips
	tips = append(tips, "🎯 Tạo hook mạnh trong 3 giây đầu để giữ chân người xem")
	tips = append(tips, "🏷️ Sử dụng 3-5 hashtags trending để tăng khả năng viral")
	tips = append(tips, "💬 Thêm câu hỏi để tăng tương tác từ người xem")

	// Limit to 5 tips
	if len(tips) > 5 {
		tips = tips[:5]
	}

	return tips
}

// generateTrendingHashtags creates relevant trending hashtags (rule-based fallback)
func generateTrendingHashtags(category, audience string) []string {
	hashtags := []string{}

	// Category-specific hashtags
	categoryHashtags := getCategoryHashtags(category)
	hashtags = append(hashtags, categoryHashtags...)

	// Trending general hashtags
	hashtags = append(hashtags, "#fyp", "#foryou", "#viral", "#trending", "#tiktok")

	// Limit to 10 hashtags
	if len(hashtags) > 10 {
		hashtags = hashtags[:10]
	}

	return hashtags
}

// generateSuggestedCaption creates an optimized caption (rule-based fallback)
func generateSuggestedCaption(transcript, currentCaption, category, audience, language string) string {
	// Use current caption if it's good, otherwise generate new one
	if len(currentCaption) > 20 && len(currentCaption) < 150 {
		return enhanceExistingCaption(currentCaption, category, audience)
	}

	// Generate new caption based on content
	return generateNewCaption(transcript, category, audience)
}

// getBestPostingTime returns optimal posting time
func getBestPostingTime(audience, category string) string {
	// Audience-specific posting times
	audienceTimes := map[string]string{
		"teenagers":     "19:00-21:00",
		"young_adults":  "20:00-22:00",
		"adults":        "18:00-20:00",
		"business":      "12:00-14:00",
		"entertainment": "19:00-21:00",
		"education":     "15:00-17:00",
		"fitness":       "06:00-08:00",
		"beauty":        "19:00-21:00",
		"food":          "11:00-13:00",
		"general":       "19:00-21:00",
	}

	if time, exists := audienceTimes[audience]; exists {
		return time
	}

	return "19:00-21:00"
}

// Helper functions
func getCategorySpecificTips(category string) []string {
	tips := map[string][]string{
		"comedy": {
			"😄 Sử dụng nhạc nền vui nhộn và hiệu ứng âm thanh",
			"🎭 Thêm biểu cảm và cử chỉ để tăng tính hài hước",
		},
		"education": {
			"📚 Sử dụng text overlay để nhấn mạnh điểm quan trọng",
			"🎯 Chia sẻ 1-2 tips thực tế mà người xem có thể áp dụng ngay",
		},
		"food": {
			"🍽️ Quay cận cảnh quá trình nấu và thành phẩm",
			"👅 Thêm biểu cảm thưởng thức để tăng cảm giác ngon",
		},
		"fitness": {
			"💪 Quay toàn thân để người xem thấy được động tác",
			"⏱️ Hiển thị thời gian tập và số lần lặp lại",
		},
	}

	if categoryTips, exists := tips[category]; exists {
		return categoryTips
	}

	return []string{}
}

func getCategoryHashtags(category string) []string {
	hashtags := map[string][]string{
		"comedy":     {"#comedy", "#funny", "#humor", "#viral"},
		"education":  {"#education", "#learn", "#knowledge", "#tips"},
		"food":       {"#food", "#cooking", "#recipe", "#delicious"},
		"fitness":    {"#fitness", "#workout", "#gym", "#health"},
		"beauty":     {"#beauty", "#makeup", "#skincare", "#glow"},
		"fashion":    {"#fashion", "#style", "#outfit", "#trend"},
		"travel":     {"#travel", "#vacation", "#trip", "#adventure"},
		"technology": {"#tech", "#technology", "#gadget", "#innovation"},
		"business":   {"#business", "#entrepreneur", "#success", "#money"},
		"lifestyle":  {"#lifestyle", "#daily", "#routine", "#life"},
	}

	if categoryHashtags, exists := hashtags[category]; exists {
		return categoryHashtags
	}

	return []string{"#viral", "#trending"}
}

func getTrendingTopics(category, audience string) []string {
	topics := map[string][]string{
		"comedy":    {"Hài kịch tình huống", "Meme trending", "Parody"},
		"education": {"Tips cuộc sống", "Kiến thức mới", "Học hỏi"},
		"food":      {"Món ăn trending", "Công thức nấu ăn", "Review đồ ăn"},
		"fitness":   {"Workout tại nhà", "Chế độ ăn", "Motivation"},
		"beauty":    {"Makeup tutorial", "Skincare routine", "Beauty tips"},
		"fashion":   {"Outfit ideas", "Style tips", "Fashion trends"},
	}

	if categoryTopics, exists := topics[category]; exists {
		return categoryTopics
	}

	return []string{"Trending topics", "Viral content", "Popular trends"}
}

func analyzeVideoPacing(duration float64, category string) string {
	if duration < 15 {
		return "Quá nhanh - cần thêm nội dung"
	} else if duration <= 30 {
		return "Tốt - phù hợp với attention span"
	} else if duration <= 60 {
		return "Lý tưởng - thời lượng tối ưu"
	} else {
		return "Chậm - cần cắt gọn để tăng engagement"
	}
}

func generateThumbnailTips(category, audience, language string) []string {
	return []string{
		"🎯 Sử dụng màu sắc tương phản mạnh",
		"📝 Thêm text ngắn gọn, dễ đọc",
		"😊 Hiển thị biểu cảm rõ ràng",
		"💡 Tạo cảm giác tò mò",
	}
}

func generateSoundSuggestions(category, audience, language string) []string {
	sounds := map[string][]string{
		"comedy":    {"Nhạc vui nhộn", "Sound effect hài hước", "Nhạc trending"},
		"education": {"Nhạc nền nhẹ nhàng", "Podcast style", "Background music"},
		"food":      {"Nhạc ẩm thực", "Sound cooking", "Nhạc chill"},
		"fitness":   {"Nhạc workout", "Nhạc motivation", "Nhạc energy"},
	}

	if categorySounds, exists := sounds[category]; exists {
		return categorySounds
	}

	return []string{"Nhạc trending", "Background music", "Sound effects"}
}

func enhanceExistingCaption(caption, category, audience string) string {
	// Add emoji based on category
	emojis := map[string]string{
		"comedy":     "😄",
		"education":  "📚",
		"food":       "🍽️",
		"fitness":    "💪",
		"beauty":     "💄",
		"fashion":    "👗",
		"travel":     "✈️",
		"technology": "💻",
		"business":   "💼",
		"lifestyle":  "🌟",
	}

	emoji := emojis[category]
	if emoji != "" {
		caption = emoji + " " + caption
	}

	// Add engagement prompt if not present
	if !strings.Contains(caption, "?") && !strings.Contains(caption, "Comment") {
		engagementPrompts := []string{
			" Comment nếu bạn thích!",
			" Bạn nghĩ sao?",
			" Follow để xem thêm!",
		}
		caption += engagementPrompts[rand.Intn(len(engagementPrompts))]
	}

	return caption
}

func generateNewCaption(transcript, category, audience string) string {
	// Extract key points from transcript
	words := strings.Fields(transcript)
	if len(words) < 5 {
		return "Nội dung thú vị! Follow để xem thêm! 🎬"
	}

	// Create caption based on category
	captions := map[string]string{
		"comedy":     "😂 " + getFirstSentence(transcript, 50) + " #comedy #funny",
		"education":  "📚 " + getFirstSentence(transcript, 60) + " #education #learn",
		"food":       "🍽️ " + getFirstSentence(transcript, 50) + " #food #cooking",
		"fitness":    "💪 " + getFirstSentence(transcript, 50) + " #fitness #motivation",
		"beauty":     "💄 " + getFirstSentence(transcript, 50) + " #beauty #glow",
		"fashion":    "👗 " + getFirstSentence(transcript, 50) + " #fashion #style",
		"travel":     "✈️ " + getFirstSentence(transcript, 50) + " #travel #adventure",
		"technology": "💻 " + getFirstSentence(transcript, 50) + " #tech #innovation",
		"business":   "💼 " + getFirstSentence(transcript, 50) + " #business #success",
		"lifestyle":  "🌟 " + getFirstSentence(transcript, 50) + " #lifestyle #daily",
	}

	if caption, exists := captions[category]; exists {
		return caption
	}

	return "🎬 " + getFirstSentence(transcript, 50) + " #viral #trending"
}

func getFirstSentence(text string, maxLength int) string {
	sentences := strings.Split(text, ".")
	if len(sentences) > 0 {
		sentence := strings.TrimSpace(sentences[0])
		if len(sentence) > maxLength {
			sentence = sentence[:maxLength] + "..."
		}
		return sentence
	}
	return text[:min(len(text), maxLength)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateEngagementPrompts(category, audience, language string) []string {
	prompts := map[string][]string{
		"comedy": {
			"Bạn có cười không? 😂",
			"Comment nếu bạn thấy vui!",
			"Tag bạn bè cần cười!",
		},
		"education": {
			"Bạn có biết điều này không?",
			"Comment tip hay nhất bạn biết!",
			"Follow để học thêm!",
		},
		"food": {
			"Bạn có thích món này không?",
			"Comment món yêu thích của bạn!",
			"Tag bạn bè thích ăn!",
		},
		"fitness": {
			"Bạn có tập thể dục không?",
			"Comment goal fitness của bạn!",
			"Tag bạn bè cần motivation!",
		},
	}

	if categoryPrompts, exists := prompts[category]; exists {
		return categoryPrompts
	}

	return []string{
		"Bạn nghĩ sao về video này?",
		"Comment nếu bạn thích!",
		"Follow để xem thêm nội dung hay!",
	}
}

func generateCallToAction(category, audience, language string) string {
	ctas := map[string]string{
		"comedy":    "Follow để cười mỗi ngày! 😂",
		"education": "Follow để học thêm kiến thức mới! 📚",
		"food":      "Follow để xem công thức nấu ăn! 🍽️",
		"fitness":   "Follow để có motivation tập luyện! 💪",
		"beauty":    "Follow để học makeup và skincare! 💄",
		"fashion":   "Follow để cập nhật xu hướng thời trang! 👗",
		"travel":    "Follow để khám phá những điểm đến mới! ✈️",
		"business":  "Follow để học kinh doanh và thành công! 💼",
	}

	if cta, exists := ctas[category]; exists {
		return cta
	}

	return "Follow để xem thêm nội dung hay! 🎬"
}
