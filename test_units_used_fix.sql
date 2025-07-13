-- Test script để kiểm tra fix units_used
USE tool;

-- Test insert với số lớn
INSERT INTO credit_transactions (
    user_id, 
    transaction_type, 
    amount, 
    base_amount, 
    service, 
    description, 
    pricing_type, 
    units_used, 
    video_id, 
    transaction_status, 
    reference_id, 
    created_at, 
    updated_at
) VALUES (
    2,
    'deduct',
    0.21161279999999996,
    0.15115199999999998,
    'tts',
    'Google TTS Test',
    'per_character',
    11803,
    5,
    'completed',
    '',
    NOW(),
    NOW()
);

-- Kiểm tra kết quả
SELECT * FROM credit_transactions WHERE description = 'Google TTS Test';

-- Xóa test data
DELETE FROM credit_transactions WHERE description = 'Google TTS Test'; 