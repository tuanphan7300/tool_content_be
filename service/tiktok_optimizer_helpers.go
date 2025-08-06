package service

import (
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

// AnalyzeContentCategory phân tích loại nội dung từ transcript và caption
func AnalyzeContentCategory(transcript, caption string) string {
	text := strings.ToLower(transcript + " " + caption)

	// Comedy/Entertainment
	if strings.Contains(text, "funny") || strings.Contains(text, "hài") || strings.Contains(text, "cười") ||
		strings.Contains(text, "joke") || strings.Contains(text, "comedy") || strings.Contains(text, "vui") {
		return "comedy"
	}

	// Education
	if strings.Contains(text, "learn") || strings.Contains(text, "học") || strings.Contains(text, "education") ||
		strings.Contains(text, "knowledge") || strings.Contains(text, "kiến thức") || strings.Contains(text, "tutorial") {
		return "education"
	}

	// Lifestyle
	if strings.Contains(text, "life") || strings.Contains(text, "cuộc sống") || strings.Contains(text, "daily") ||
		strings.Contains(text, "routine") || strings.Contains(text, "thói quen") || strings.Contains(text, "lifestyle") {
		return "lifestyle"
	}

	// Food/Cooking
	if strings.Contains(text, "food") || strings.Contains(text, "ăn") || strings.Contains(text, "cook") ||
		strings.Contains(text, "nấu") || strings.Contains(text, "recipe") || strings.Contains(text, "món") {
		return "food"
	}

	// Travel
	if strings.Contains(text, "travel") || strings.Contains(text, "du lịch") || strings.Contains(text, "trip") ||
		strings.Contains(text, "visit") || strings.Contains(text, "địa điểm") || strings.Contains(text, "destination") {
		return "travel"
	}

	// Fitness/Health
	if strings.Contains(text, "fitness") || strings.Contains(text, "gym") || strings.Contains(text, "workout") ||
		strings.Contains(text, "exercise") || strings.Contains(text, "tập") || strings.Contains(text, "sức khỏe") {
		return "fitness"
	}

	// Technology
	if strings.Contains(text, "tech") || strings.Contains(text, "technology") || strings.Contains(text, "app") ||
		strings.Contains(text, "software") || strings.Contains(text, "công nghệ") || strings.Contains(text, "máy tính") {
		return "technology"
	}

	// Fashion/Beauty
	if strings.Contains(text, "fashion") || strings.Contains(text, "style") || strings.Contains(text, "beauty") ||
		strings.Contains(text, "makeup") || strings.Contains(text, "thời trang") || strings.Contains(text, "làm đẹp") {
		return "fashion"
	}

	// Music
	if strings.Contains(text, "music") || strings.Contains(text, "song") || strings.Contains(text, "nhạc") ||
		strings.Contains(text, "ca hát") || strings.Contains(text, "dance") || strings.Contains(text, "nhảy") {
		return "music"
	}

	// Gaming
	if strings.Contains(text, "game") || strings.Contains(text, "gaming") || strings.Contains(text, "play") ||
		strings.Contains(text, "chơi") || strings.Contains(text, "esports") || strings.Contains(text, "stream") {
		return "gaming"
	}

	// Business/Finance
	if strings.Contains(text, "business") || strings.Contains(text, "money") || strings.Contains(text, "finance") ||
		strings.Contains(text, "kinh doanh") || strings.Contains(text, "tiền") || strings.Contains(text, "đầu tư") {
		return "business"
	}

	// Default
	return "general"
}

// GetBestPostingTime trả về thời gian đăng tốt nhất
func GetBestPostingTime(audience, category string) string {

	// Dựa trên audience
	switch audience {
	case "teenagers":
		return "19:00-21:00"
	case "young_adults":
		return "20:00-22:00"
	case "adults":
		return "19:00-21:00"
	case "business":
		return "12:00-13:00"
	default:
		return "19:00-21:00"
	}
}

// AnalyzeVideoPacing phân tích pacing của video
func AnalyzeVideoPacing(duration float64, category string) string {
	if duration < 15 {
		return "fast"
	} else if duration < 30 {
		return "medium"
	} else if duration < 60 {
		return "moderate"
	} else {
		return "slow"
	}
}

// GenerateThumbnailTips tạo tips cho thumbnail
func GenerateThumbnailTips(category, audience, language string) []string {
	tips := []string{}

	// Dựa trên category
	switch category {
	case "comedy":
		tips = append(tips, "Sử dụng biểu cảm hài hước")
		tips = append(tips, "Thêm text gây cười")
	case "education":
		tips = append(tips, "Hiển thị key points rõ ràng")
		tips = append(tips, "Sử dụng màu sắc tương phản")
	case "food":
		tips = append(tips, "Chụp món ăn từ góc đẹp")
		tips = append(tips, "Sử dụng ánh sáng tự nhiên")
	default:
		tips = append(tips, "Tạo thumbnail bắt mắt")
		tips = append(tips, "Sử dụng màu sắc nổi bật")
	}

	return tips
}

// GenerateSoundSuggestions tạo gợi ý âm thanh
func GenerateSoundSuggestions(category, audience, language string) []string {
	suggestions := []string{}

	// Dựa trên category
	switch category {
	case "comedy":
		suggestions = append(suggestions, "Âm thanh hài hước")
		suggestions = append(suggestions, "Sound effects vui nhộn")
	case "education":
		suggestions = append(suggestions, "Nhạc nền nhẹ nhàng")
		suggestions = append(suggestions, "Không có nhạc nền")
	case "music":
		suggestions = append(suggestions, "Sử dụng nhạc trending")
		suggestions = append(suggestions, "Original sound")
	case "fitness":
		suggestions = append(suggestions, "Nhạc động lực")
		suggestions = append(suggestions, "Beat mạnh mẽ")
	default:
		suggestions = append(suggestions, "Nhạc trending phù hợp")
		suggestions = append(suggestions, "Original sound")
	}

	return suggestions
}

// CalculateHookScore tính điểm hook
func CalculateHookScore(transcript string, duration float64, category string) int {
	score := 50 // Base score

	// Dựa trên độ dài
	if duration < 15 {
		score += 20 // Ngắn = hook tốt
	} else if duration > 60 {
		score -= 10 // Dài = hook kém
	}

	// Dựa trên category
	switch category {
	case "comedy":
		score += 15
	case "education":
		score += 10
	case "music":
		score += 20
	default:
		score += 5
	}

	// Dựa trên transcript
	if strings.Contains(strings.ToLower(transcript), "shocking") || strings.Contains(strings.ToLower(transcript), "sốc") {
		score += 25
	}
	if strings.Contains(strings.ToLower(transcript), "secret") || strings.Contains(strings.ToLower(transcript), "bí mật") {
		score += 20
	}

	// Giới hạn score
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	return score
}

// CalculateViralPotential tính tiềm năng viral
func CalculateViralPotential(transcript string, duration float64, category, audience string) int {
	score := 40 // Base score

	// Dựa trên độ dài
	if duration >= 15 && duration <= 60 {
		score += 20 // Độ dài lý tưởng
	} else if duration < 15 {
		score += 10
	} else {
		score -= 10
	}

	// Dựa trên category
	switch category {
	case "comedy":
		score += 25
	case "music":
		score += 20
	case "dance":
		score += 20
	case "challenge":
		score += 30
	default:
		score += 10
	}

	// Dựa trên audience
	switch audience {
	case "teenagers":
		score += 15
	case "young_adults":
		score += 10
	default:
		score += 5
	}

	// Giới hạn score
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	return score
}

// GetCategoryHashtags trả về hashtags cho category cụ thể
func GetCategoryHashtags(category string) []string {
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
		"music":      {"#music", "#song", "#dance", "#viral"},
		"gaming":     {"#gaming", "#game", "#esports", "#stream"},
	}

	if categoryHashtags, exists := hashtags[category]; exists {
		return categoryHashtags
	}

	return []string{"#viral", "#trending"}
}

// GetTrendingTopics trả về trending topics cho category cụ thể
func GetTrendingTopics(category, audience string) []string {
	topics := map[string][]string{
		"comedy":    {"Hài kịch tình huống", "Meme trending", "Parody"},
		"education": {"Tips cuộc sống", "Kiến thức mới", "Học hỏi"},
		"food":      {"Món ăn trending", "Công thức nấu ăn", "Review đồ ăn"},
		"fitness":   {"Workout tại nhà", "Chế độ ăn", "Motivation"},
		"beauty":    {"Makeup tutorial", "Skincare routine", "Beauty tips"},
		"fashion":   {"Outfit ideas", "Style tips", "Fashion trends"},
		"music":     {"Nhạc trending", "Dance challenge", "Cover songs"},
		"gaming":    {"Game reviews", "Gaming tips", "Esports highlights"},
		"travel":    {"Địa điểm mới", "Travel tips", "Adventure vlogs"},
		"technology": {"Tech reviews", "App recommendations", "Gadget unboxing"},
	}

	if categoryTopics, exists := topics[category]; exists {
		return categoryTopics
	}

	return []string{"Trending topics", "Viral content", "Popular trends"}
}
