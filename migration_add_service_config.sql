-- Migration: Tạo bảng service_config để quản lý dịch vụ sử dụng cho từng nghiệp vụ
CREATE TABLE IF NOT EXISTS service_config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    service_type VARCHAR(64) NOT NULL, -- Loại nghiệp vụ: srt_translation, speech_to_text, text_to_speech, ...
    service_name VARCHAR(64) NOT NULL, -- Trùng với service_name trong service_pricings
    is_active BOOLEAN NOT NULL DEFAULT 1, -- Đánh dấu dịch vụ đang active cho nghiệp vụ này
    config_json TEXT DEFAULT NULL, -- Tuỳ chọn: lưu config đặc biệt dạng JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_service_type (service_type, service_name)
);
-- Index để truy vấn nhanh theo type và trạng thái
CREATE INDEX idx_service_type_active ON service_config(service_type, is_active); 