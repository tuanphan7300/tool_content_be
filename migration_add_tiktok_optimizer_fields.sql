-- Migration: Thêm các trường TikTok Optimizer vào tool_caption_histories
ALTER TABLE tool_caption_histories
ADD COLUMN hook_score INT DEFAULT 0,
ADD COLUMN viral_potential INT DEFAULT 0,
ADD COLUMN trending_hashtags JSON,
ADD COLUMN suggested_caption TEXT,
ADD COLUMN best_posting_time VARCHAR(64),
ADD COLUMN optimization_tips JSON,
ADD COLUMN engagement_prompts JSON,
ADD COLUMN call_to_action TEXT; 