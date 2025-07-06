-- Script fix lỗi database cho Tool Content Backend
-- Chạy script này để fix lỗi "BLOB/TEXT column used in key specification without a key length"

USE `tool`;

-- Xóa các index có vấn đề
DROP INDEX IF EXISTS `idx_video_filename` ON `caption_histories`;

-- Tạo lại index với độ dài key được chỉ định
CREATE INDEX `idx_video_filename` ON `caption_histories` (`video_filename`(255));

-- Kiểm tra và tạo lại các index khác nếu cần
CREATE INDEX IF NOT EXISTS `idx_user_id` ON `caption_histories` (`user_id`);
CREATE INDEX IF NOT EXISTS `idx_created_at` ON `caption_histories` (`created_at`);

-- Kiểm tra foreign keys
SELECT 
    CONSTRAINT_NAME,
    TABLE_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE 
WHERE TABLE_SCHEMA = 'tool' 
AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- Hiển thị thông tin về các bảng
SELECT 
    TABLE_NAME,
    TABLE_ROWS,
    DATA_LENGTH,
    INDEX_LENGTH,
    (DATA_LENGTH + INDEX_LENGTH) as TOTAL_SIZE
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'tool' 
ORDER BY TABLE_NAME;

-- Hiển thị thông tin về indexes của caption_histories
SHOW INDEX FROM `caption_histories`; 