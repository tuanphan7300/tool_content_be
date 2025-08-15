-- Migration script để thêm trường video_duration vào bảng caption_history
-- Thực hiện: ALTER TABLE caption_history ADD COLUMN video_duration DECIMAL(10,2) COMMENT 'Duration in seconds';

-- Kiểm tra xem trường đã tồn tại chưa
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = DATABASE() 
     AND TABLE_NAME = 'caption_history' 
     AND COLUMN_NAME = 'video_duration') > 0,
    'SELECT "Column video_duration already exists" as message',
    'ALTER TABLE caption_history ADD COLUMN video_duration DECIMAL(10,2) COMMENT "Duration in seconds"'
));

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Kiểm tra kết quả
SELECT "Migration completed successfully" as message;
