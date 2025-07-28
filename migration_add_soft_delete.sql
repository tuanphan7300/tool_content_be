-- Migration: Thêm cột deleted_at cho soft delete
-- Thêm cột deleted_at vào bảng caption_histories
ALTER TABLE caption_histories 
ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- Tạo index cho deleted_at để tối ưu query
CREATE INDEX idx_caption_histories_deleted_at ON caption_histories(deleted_at);

-- Cập nhật các bảng khác nếu cần
-- ALTER TABLE tool_caption_histories 
-- ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
-- CREATE INDEX idx_tool_caption_histories_deleted_at ON tool_caption_histories(deleted_at); 