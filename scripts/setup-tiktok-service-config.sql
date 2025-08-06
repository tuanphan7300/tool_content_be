-- Setup TikTok Optimizer Service Config
-- Chạy script này để thêm service config cho TikTok Optimizer

-- Thêm service config cho TikTok Optimizer
INSERT INTO service_config (service_type, service_name, is_active, config_json, created_at, updated_at) VALUES 
('tiktok-optimizer', 'TikTok Optimizer AI-Enhanced', 1, '{
  "use_ai": true,
  "supported_languages": ["vi", "en"],
  "ai_cost_multiplier": 1.0,
  "max_tokens_per_call": 2000
}', NOW(), NOW());

-- Thêm pricing cho TikTok Optimizer
INSERT INTO service_pricing (service_name, pricing_type, price_per_unit, currency, description, model_api_name, is_active, created_at, updated_at) VALUES 
('tiktok-optimizer', 'per_token', 0.0001, 'USD', 'TikTok Optimizer per token', 'gpt-4', 1, NOW(), NOW());

-- Kiểm tra xem đã insert thành công chưa
SELECT 'Service Config' as type, service_type, service_name, is_active FROM service_config WHERE service_type = 'tiktok-optimizer';
SELECT 'Service Pricing' as type, service_name, pricing_type, price_per_unit, currency FROM service_pricing WHERE service_name = 'tiktok-optimizer'; 