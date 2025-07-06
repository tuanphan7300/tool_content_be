-- Fix database charset cho UTF-8mb4
-- Chạy script này để đảm bảo database hỗ trợ tiếng Trung, tiếng Việt

-- 1. Tạo lại database với charset đúng
DROP DATABASE IF EXISTS `tool`;
CREATE DATABASE `tool` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `tool`;

-- 2. Tạo lại tất cả bảng với charset đúng
-- Bảng Users
CREATE TABLE `users` (
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
    UNIQUE KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng UserTokens
CREATE TABLE `user_tokens` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `total_tokens` int NOT NULL DEFAULT 0,
    `used_tokens` int NOT NULL DEFAULT 0,
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng TokenTransaction
CREATE TABLE `token_transactions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `type` varchar(20) NOT NULL COMMENT 'add hoặc deduct',
    `amount` int NOT NULL,
    `description` varchar(500) DEFAULT NULL,
    `service` varchar(100) DEFAULT NULL COMMENT 'whisper, gemini, tts, topup',
    `video_id` bigint unsigned DEFAULT NULL,
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Bảng CaptionHistory
CREATE TABLE `caption_histories` (
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
    PRIMARY KEY (`id`),
    KEY `idx_video_filename` (`video_filename`(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Insert dữ liệu mẫu
INSERT INTO `users` (`email`, `name`, `auth_provider`, `email_verified`) 
VALUES ('test@example.com', 'Test User', 'local', 1);

INSERT INTO `user_tokens` (`user_id`, `total_tokens`, `used_tokens`)
SELECT `id`, 1000, 0 FROM `users` WHERE `email` = 'test@example.com';

-- 4. Kiểm tra charset
SELECT 
    TABLE_SCHEMA,
    TABLE_NAME,
    TABLE_COLLATION
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'tool';

-- 5. Kiểm tra charset của các cột text
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    CHARACTER_SET_NAME,
    COLLATION_NAME
FROM information_schema.COLUMNS 
WHERE TABLE_SCHEMA = 'tool' 
AND DATA_TYPE IN ('varchar', 'text', 'longtext')
ORDER BY TABLE_NAME, COLUMN_NAME; 