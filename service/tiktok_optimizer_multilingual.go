package service

import (
	"strings"
)

// MultilingualTikTokContent chứa nội dung đa ngôn ngữ cho TikTok Optimizer
type MultilingualTikTokContent struct {
	Language string
	Content  map[string]interface{}
}

// getMultilingualContent trả về nội dung theo ngôn ngữ
func getMultilingualContent(language string) map[string]interface{} {
	content := map[string]map[string]interface{}{
		"vi": {
			"duration_short":  "🎬 Video quá ngắn! Thêm nội dung để đạt 15-60 giây lý tưởng",
			"duration_long":   "⏱️ Video quá dài! Cắt gọn xuống 60 giây để tăng engagement",
			"hook_tip":        "🎯 Tạo hook mạnh trong 3 giây đầu để giữ chân người xem",
			"hashtag_tip":     "🏷️ Sử dụng 3-5 hashtags trending để tăng khả năng viral",
			"engagement_tip":  "💬 Thêm câu hỏi để tăng tương tác từ người xem",
			"default_caption": "Nội dung thú vị! Follow để xem thêm! 🎬",
			"comedy_tips": []string{
				"😄 Sử dụng nhạc nền vui nhộn và hiệu ứng âm thanh",
				"🎭 Thêm biểu cảm và cử chỉ để tăng tính hài hước",
			},
			"education_tips": []string{
				"📚 Sử dụng text overlay để nhấn mạnh điểm quan trọng",
				"🎯 Chia sẻ 1-2 tips thực tế mà người xem có thể áp dụng ngay",
			},
			"food_tips": []string{
				"🍽️ Quay cận cảnh quá trình nấu và thành phẩm",
				"👅 Thêm biểu cảm thưởng thức để tăng cảm giác ngon",
			},
			"fitness_tips": []string{
				"💪 Quay toàn thân để người xem thấy được động tác",
				"⏱️ Hiển thị thời gian tập và số lần lặp lại",
			},
			"comedy_prompts": []string{
				"Bạn có cười không? 😂",
				"Comment nếu bạn thấy vui!",
				"Tag bạn bè cần cười!",
			},
			"education_prompts": []string{
				"Bạn có biết điều này không?",
				"Comment tip hay nhất bạn biết!",
				"Follow để học thêm!",
			},
			"food_prompts": []string{
				"Bạn có thích món này không?",
				"Comment món yêu thích của bạn!",
				"Tag bạn bè thích ăn!",
			},
			"fitness_prompts": []string{
				"Bạn có tập thể dục không?",
				"Comment goal fitness của bạn!",
				"Tag bạn bè cần motivation!",
			},
			"general_prompts": []string{
				"Bạn nghĩ sao về video này?",
				"Comment nếu bạn thích!",
				"Follow để xem thêm nội dung hay!",
			},
			"comedy_cta":    "Follow để cười mỗi ngày! 😂",
			"education_cta": "Follow để học thêm kiến thức mới! 📚",
			"food_cta":      "Follow để xem công thức nấu ăn! 🍽️",
			"fitness_cta":   "Follow để có motivation tập luyện! 💪",
			"beauty_cta":    "Follow để học makeup và skincare! 💄",
			"fashion_cta":   "Follow để cập nhật xu hướng thời trang! 👗",
			"travel_cta":    "Follow để khám phá những điểm đến mới! ✈️",
			"business_cta":  "Follow để học kinh doanh và thành công! 💼",
			"default_cta":   "Follow để xem thêm nội dung hay! 🎬",
		},
		"en": {
			"duration_short":  "🎬 Video too short! Add content to reach ideal 15-60 seconds",
			"duration_long":   "⏱️ Video too long! Cut down to 60 seconds to increase engagement",
			"hook_tip":        "🎯 Create a strong hook in the first 3 seconds to keep viewers",
			"hashtag_tip":     "🏷️ Use 3-5 trending hashtags to increase viral potential",
			"engagement_tip":  "💬 Add questions to increase viewer interaction",
			"default_caption": "Interesting content! Follow for more! 🎬",
			"comedy_tips": []string{
				"😄 Use upbeat background music and sound effects",
				"🎭 Add expressions and gestures to enhance humor",
			},
			"education_tips": []string{
				"📚 Use text overlays to emphasize key points",
				"🎯 Share 1-2 practical tips viewers can apply immediately",
			},
			"food_tips": []string{
				"🍽️ Film close-ups of cooking process and final dish",
				"👅 Add tasting expressions to enhance delicious feeling",
			},
			"fitness_tips": []string{
				"💪 Film full body to show viewers the movements",
				"⏱️ Display workout time and repetition count",
			},
			"comedy_prompts": []string{
				"Did this make you laugh? 😂",
				"Comment if you found it funny!",
				"Tag friends who need a laugh!",
			},
			"education_prompts": []string{
				"Did you know this?",
				"Comment your best tip!",
				"Follow to learn more!",
			},
			"food_prompts": []string{
				"Do you like this dish?",
				"Comment your favorite food!",
				"Tag food-loving friends!",
			},
			"fitness_prompts": []string{
				"Do you exercise?",
				"Comment your fitness goal!",
				"Tag friends who need motivation!",
			},
			"general_prompts": []string{
				"What do you think about this video?",
				"Comment if you like it!",
				"Follow for more great content!",
			},
			"comedy_cta":    "Follow for daily laughs! 😂",
			"education_cta": "Follow to learn new knowledge! 📚",
			"food_cta":      "Follow for cooking recipes! 🍽️",
			"fitness_cta":   "Follow for workout motivation! 💪",
			"beauty_cta":    "Follow to learn makeup and skincare! 💄",
			"fashion_cta":   "Follow for fashion trends! 👗",
			"travel_cta":    "Follow to explore new destinations! ✈️",
			"business_cta":  "Follow to learn business and success! 💼",
			"default_cta":   "Follow for more great content! 🎬",
		},
		"ja": {
			"duration_short":  "🎬 動画が短すぎます！理想的な15-60秒に達するようコンテンツを追加してください",
			"duration_long":   "⏱️ 動画が長すぎます！エンゲージメント向上のため60秒に短縮してください",
			"hook_tip":        "🎯 最初の3秒で強力なフックを作成して視聴者を引き留めましょう",
			"hashtag_tip":     "🏷️ バイラル効果を高めるため3-5個のトレンドハッシュタグを使用してください",
			"engagement_tip":  "💬 視聴者のインタラクションを増やすため質問を追加してください",
			"default_caption": "面白いコンテンツ！もっと見るためにフォローしてください！🎬",
			"comedy_tips": []string{
				"😄 明るいBGMとサウンドエフェクトを使用",
				"🎭 ユーモアを高めるため表情とジェスチャーを追加",
			},
			"education_tips": []string{
				"📚 重要なポイントを強調するためテキストオーバーレイを使用",
				"🎯 視聴者がすぐに実践できる1-2の実用的なヒントを共有",
			},
			"food_tips": []string{
				"🍽️ 調理過程と完成品のクローズアップを撮影",
				"👅 美味しさを高めるため味見の表情を追加",
			},
			"fitness_tips": []string{
				"💪 視聴者が動きを見られるよう全身を撮影",
				"⏱️ ワークアウト時間と繰り返し回数を表示",
			},
			"comedy_prompts": []string{
				"笑いましたか？😂",
				"面白かったらコメントしてください！",
				"笑いが必要な友達をタグしてください！",
			},
			"education_prompts": []string{
				"これを知っていましたか？",
				"あなたの最高のヒントをコメントしてください！",
				"もっと学ぶためにフォローしてください！",
			},
			"food_prompts": []string{
				"この料理は好きですか？",
				"あなたの好きな食べ物をコメントしてください！",
				"食べ物好きの友達をタグしてください！",
			},
			"fitness_prompts": []string{
				"運動していますか？",
				"あなたのフィットネス目標をコメントしてください！",
				"モチベーションが必要な友達をタグしてください！",
			},
			"general_prompts": []string{
				"この動画についてどう思いますか？",
				"気に入ったらコメントしてください！",
				"もっと素晴らしいコンテンツを見るためにフォローしてください！",
			},
			"comedy_cta":    "毎日笑うためにフォローしてください！😂",
			"education_cta": "新しい知識を学ぶためにフォローしてください！📚",
			"food_cta":      "料理レシピを見るためにフォローしてください！🍽️",
			"fitness_cta":   "ワークアウトのモチベーションを得るためにフォローしてください！💪",
			"beauty_cta":    "メイクとスキンケアを学ぶためにフォローしてください！💄",
			"fashion_cta":   "ファッショントレンドを更新するためにフォローしてください！👗",
			"travel_cta":    "新しい目的地を探検するためにフォローしてください！✈️",
			"business_cta":  "ビジネスと成功を学ぶためにフォローしてください！💼",
			"default_cta":   "もっと素晴らしいコンテンツを見るためにフォローしてください！🎬",
		},
	}

	// Default to Vietnamese if language not found
	if langContent, exists := content[language]; exists {
		return langContent
	}
	return content["vi"]
}

// getLocalizedText trả về text theo ngôn ngữ
func getLocalizedText(language, key string) string {
	content := getMultilingualContent(language)
	if text, exists := content[key]; exists {
		if str, ok := text.(string); ok {
			return str
		}
	}
	return ""
}

// getLocalizedTips trả về tips theo ngôn ngữ
func getLocalizedTips(language, category string) []string {
	content := getMultilingualContent(language)
	key := category + "_tips"
	if tips, exists := content[key]; exists {
		if tipSlice, ok := tips.([]string); ok {
			return tipSlice
		}
	}
	return []string{}
}

// getLocalizedPrompts trả về prompts theo ngôn ngữ
func getLocalizedPrompts(language, category string) []string {
	content := getMultilingualContent(language)
	key := category + "_prompts"
	if prompts, exists := content[key]; exists {
		if promptSlice, ok := prompts.([]string); ok {
			return promptSlice
		}
	}
	return getLocalizedPrompts(language, "general")
}

// getLocalizedCTA trả về call-to-action theo ngôn ngữ
func getLocalizedCTA(language, category string) string {
	content := getMultilingualContent(language)
	key := category + "_cta"
	if cta, exists := content[key]; exists {
		if str, ok := cta.(string); ok {
			return str
		}
	}
	return getLocalizedText(language, "default_cta")
}

// getLocalizedCaption trả về caption theo ngôn ngữ
func getLocalizedCaption(language, category string, transcript string) string {
	// Category-specific captions
	captions := map[string]string{
		"comedy":     "😂 " + getFirstSentenceMultilingual(transcript, 50) + " #comedy #funny",
		"education":  "📚 " + getFirstSentenceMultilingual(transcript, 60) + " #education #learn",
		"food":       "🍽️ " + getFirstSentenceMultilingual(transcript, 50) + " #food #cooking",
		"fitness":    "💪 " + getFirstSentenceMultilingual(transcript, 50) + " #fitness #motivation",
		"beauty":     "💄 " + getFirstSentenceMultilingual(transcript, 50) + " #beauty #glow",
		"fashion":    "👗 " + getFirstSentenceMultilingual(transcript, 50) + " #fashion #style",
		"travel":     "✈️ " + getFirstSentenceMultilingual(transcript, 50) + " #travel #adventure",
		"technology": "💻 " + getFirstSentenceMultilingual(transcript, 50) + " #tech #innovation",
		"business":   "💼 " + getFirstSentenceMultilingual(transcript, 50) + " #business #success",
		"lifestyle":  "🌟 " + getFirstSentenceMultilingual(transcript, 50) + " #lifestyle #daily",
	}

	if caption, exists := captions[category]; exists {
		return caption
	}

	return getLocalizedText(language, "default_caption")
}

// getFirstSentenceMultilingual helper function
func getFirstSentenceMultilingual(text string, maxLength int) string {
	sentences := strings.Split(text, ".")
	if len(sentences) > 0 {
		sentence := strings.TrimSpace(sentences[0])
		if len(sentence) > maxLength {
			sentence = sentence[:maxLength] + "..."
		}
		return sentence
	}
	return text[:minMultilingual(len(text), maxLength)]
}

func minMultilingual(a, b int) int {
	if a < b {
		return a
	}
	return b
}
