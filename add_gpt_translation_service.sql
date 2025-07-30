-- Thêm GPT translation service vào service_config
INSERT INTO service_config (service_type, service_name, is_active, config_json, created_at, updated_at) 
VALUES ('srt_translation', 'gpt_translation', 1, '{"model": "gpt-4o-mini"}', NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    is_active = 1,
    config_json = '{"model": "gpt-4o-mini"}',
    updated_at = NOW();

-- Thêm pricing cho GPT translation
INSERT INTO service_pricings (service_name, price_per_unit, pricing_type, is_active, created_at, updated_at)
VALUES ('gpt_translation', 0.00015, 'per_token', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    price_per_unit = 0.00015,
    is_active = 1,
    updated_at = NOW();

-- Thêm service markup cho GPT translation (nếu cần)
INSERT INTO service_pricings (service_name, price_per_unit, pricing_type, model_api_name, is_active, created_at, updated_at)
VALUES ('gpt_translation', 0.00015, 'per_token', 'gpt-4o-mini', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    price_per_unit = 0.00015,
    model_api_name = 'gpt-4o-mini',
    is_active = 1,
    updated_at = NOW();

-- Hiển thị kết quả
SELECT 'GPT Translation Service added successfully' as message; 