-- Script kiểm tra và setup TikTok Optimizer Service Config
-- Chạy script này để đảm bảo TikTok Optimizer hoạt động

-- 1. Kiểm tra xem service config đã tồn tại chưa
SELECT 'Checking TikTok Optimizer Service Config...' as message;

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ TikTok Optimizer service config exists'
        ELSE '❌ TikTok Optimizer service config NOT found'
    END as status,
    COUNT(*) as count
FROM service_config 
WHERE service_type = 'tiktok-optimizer' AND is_active = 1;

-- 2. Kiểm tra service pricing
SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ TikTok Optimizer service pricing exists'
        ELSE '❌ TikTok Optimizer service pricing NOT found'
    END as status,
    COUNT(*) as count
FROM service_pricing 
WHERE service_name = 'tiktok-optimizer' AND is_active = 1;

-- 3. Nếu chưa có service config, thêm vào
INSERT IGNORE INTO service_config (service_type, service_name, is_active, config_json, created_at, updated_at) 
SELECT 
    'tiktok-optimizer', 
    'TikTok Optimizer AI-Enhanced', 
    1, 
    '{
      "use_ai": true,
      "supported_languages": ["vi", "en"],
      "ai_cost_multiplier": 1.0,
      "max_tokens_per_call": 2000
    }', 
    NOW(), 
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM service_config 
    WHERE service_type = 'tiktok-optimizer' AND is_active = 1
);

-- 4. Nếu chưa có service pricing, thêm vào
INSERT IGNORE INTO service_pricing (service_name, pricing_type, price_per_unit, currency, description, model_api_name, is_active, created_at, updated_at) 
SELECT 
    'tiktok-optimizer', 
    'per_token', 
    0.0001, 
    'USD', 
    'TikTok Optimizer per token', 
    'gpt-4', 
    1, 
    NOW(), 
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM service_pricing 
    WHERE service_name = 'tiktok-optimizer' AND is_active = 1
);

-- 5. Kiểm tra kết quả cuối cùng
SELECT 'Final verification:' as message;

SELECT 'Service Config' as type, service_type, service_name, is_active, created_at
FROM service_config 
WHERE service_type = 'tiktok-optimizer';

SELECT 'Service Pricing' as type, service_name, pricing_type, price_per_unit, currency, is_active
FROM service_pricing 
WHERE service_name = 'tiktok-optimizer';

SELECT '🎉 TikTok Optimizer Service Config setup completed!' as message;
