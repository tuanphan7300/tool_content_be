-- Script test để kiểm tra migration video_duration
-- Chạy từng lệnh một để kiểm tra

-- 1. Kiểm tra cấu trúc bảng hiện tại
DESCRIBE caption_history;

-- 2. Kiểm tra xem trường video_duration đã tồn tại chưa
SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_COMMENT
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = DATABASE() 
AND TABLE_NAME = 'caption_history' 
AND COLUMN_NAME = 'video_duration';

-- 3. Nếu chưa có, thêm trường
ALTER TABLE caption_history ADD COLUMN video_duration DECIMAL(10,2) COMMENT 'Duration in seconds';

-- 4. Kiểm tra lại cấu trúc bảng
DESCRIBE caption_history;

-- 5. Kiểm tra dữ liệu hiện tại
SELECT id, process_type, video_duration, created_at 
FROM caption_history 
ORDER BY created_at DESC 
LIMIT 10;

-- 6. Cập nhật video_duration = 0 cho các record cũ (nếu cần)
UPDATE caption_history SET video_duration = 0 WHERE video_duration IS NULL;
