-- Migration script để khởi tạo database cho Tool Content Backend
-- Chạy script này để tạo tất cả các bảng cần thiết

-- Tạo database nếu chưa có
CREATE DATABASE IF NOT EXISTS `tool` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `tool`;

-- Bảng Users - lưu thông tin người dùng
CREATE TABLE IF NOT EXISTS `users` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `email` varchar(255) NOT NULL,
    `password_hash` varchar(255) DEFAULT NULL,
    `google_id` varchar(255) DEFAULT NULL,
    `name` varchar(255) DEFAULT NULL,
    `picture` text DEFAULT NULL,
    `email_verified` tinyint(1) DEFAULT 0,
    `auth_provider` varchar(50) DEFAULT 'local',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_email` (`email`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng UserTokens - lưu số dư token của user
CREATE TABLE IF NOT EXISTS `user_tokens` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `total_tokens` int NOT NULL DEFAULT 0,
    `used_tokens` int NOT NULL DEFAULT 0,
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng TokenTransaction - lưu lịch sử giao dịch token
CREATE TABLE IF NOT EXISTS `token_transactions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `type` varchar(20) NOT NULL COMMENT 'add hoặc deduct',
    `amount` int NOT NULL,
    `description` varchar(500) DEFAULT NULL,
    `service` varchar(100) DEFAULT NULL COMMENT 'whisper, gemini, tts, topup',
    `video_id` bigint unsigned DEFAULT NULL,
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng CaptionHistory - lưu lịch sử xử lý video
CREATE TABLE IF NOT EXISTS `caption_histories` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `video_filename` varchar(500) NOT NULL,
    `video_filename_origin` varchar(500) DEFAULT NULL,
    `transcript` text DEFAULT NULL,
    `suggestion` text DEFAULT NULL,
    `segments` json DEFAULT NULL,
    `segments_vi` json DEFAULT NULL,
    `timestamps` json DEFAULT NULL,
    `background_music` varchar(500) DEFAULT NULL,
    `srt_file` varchar(500) DEFAULT NULL,
    `original_srt_file` varchar(500) DEFAULT NULL,
    `tts_file` varchar(500) DEFAULT NULL,
    `merged_video_file` varchar(500) DEFAULT NULL,
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert dữ liệu mẫu cho testing (optional)
-- Tạo user test
INSERT INTO `users` (`email`, `name`, `auth_provider`, `email_verified`) 
VALUES ('test@example.com', 'Test User', 'local', 1)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- Tạo token cho user test
INSERT INTO `user_tokens` (`user_id`, `total_tokens`, `used_tokens`)
SELECT `id`, 1000, 0 FROM `users` WHERE `email` = 'test@example.com'
ON DUPLICATE KEY UPDATE `total_tokens` = 1000, `used_tokens` = 0;

-- Hiển thị thông tin về các bảng đã tạo
SELECT 
    TABLE_NAME,
    TABLE_ROWS,
    DATA_LENGTH,
    INDEX_LENGTH,
    (DATA_LENGTH + INDEX_LENGTH) as TOTAL_SIZE
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'tool' 
ORDER BY TABLE_NAME;

-- Hiển thị thông tin về foreign keys
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