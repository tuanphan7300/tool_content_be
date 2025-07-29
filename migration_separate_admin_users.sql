-- Migration: Separate admin users completely
-- Date: 2025-07-29

-- Remove is_admin column from users table (clean separation)
ALTER TABLE `users` DROP COLUMN `is_admin`;

-- Update admin_users table with better structure
ALTER TABLE `admin_users` 
ADD COLUMN `permissions` JSON DEFAULT NULL AFTER `role`,
ADD COLUMN `login_attempts` INT DEFAULT 0 AFTER `is_active`,
ADD COLUMN `locked_until` TIMESTAMP NULL DEFAULT NULL AFTER `login_attempts`;

-- Insert default admin accounts with better security
-- Password: admin123 (bcrypt hash)
INSERT INTO `admin_users` (`username`, `password_hash`, `email`, `name`, `role`, `permissions`) VALUES
('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@inis-hvnh.site', 'System Administrator', 'super_admin', '["*"]'),
('moderator', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'moderator@inis-hvnh.site', 'Content Moderator', 'moderator', '["view_users", "view_processes", "view_uploads"]');

-- Create admin_sessions table for better session management
CREATE TABLE `admin_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` int NOT NULL,
  `token` varchar(255) NOT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` text,
  `expires_at` timestamp NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `token` (`token`),
  KEY `idx_admin_id` (`admin_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng quản lý session của admin';

-- Add indexes for better performance
ALTER TABLE `admin_users` ADD INDEX `idx_username_active` (`username`, `is_active`);
ALTER TABLE `admin_users` ADD INDEX `idx_role` (`role`);
ALTER TABLE `admin_audit_logs` ADD INDEX `idx_admin_action` (`admin_id`, `action`);


use tool;
-- CREATE TABLE `admin_users` (
--   `id` int NOT NULL AUTO_INCREMENT,
--   `username` varchar(255) NOT NULL,
--   `password_hash` text NOT NULL,
--   `email` varchar(255) DEFAULT NULL,
--   `name` varchar(255) DEFAULT NULL,
--   `role` enum('super_admin','admin','moderator') DEFAULT 'admin',
--   `is_active` tinyint(1) DEFAULT 1,
--   `last_login` timestamp NULL DEFAULT NULL,
--   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
--   `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
--   PRIMARY KEY (`id`),
--   UNIQUE KEY `username` (`username`),
--   UNIQUE KEY `email` (`email`)
-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng quản lý tài khoản admin';

-- INSERT INTO `admin_users` (`username`, `password_hash`, `email`, `name`, `role`)
-- VALUES ('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@inis-hvnh.site', 'System Administrator', 'super_admin');
-- -- Create admin_audit_logs table for tracking admin actions
-- CREATE TABLE `admin_audit_logs` (
--   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
--   `admin_id` int NOT NULL,
--   `action` varchar(100) NOT NULL,
--   `table_name` varchar(100) DEFAULT NULL,
--   `record_id` bigint unsigned DEFAULT NULL,
--   `old_values` json DEFAULT NULL,
--   `new_values` json DEFAULT NULL,
--   `ip_address` varchar(45) DEFAULT NULL,
--   `user_agent` text,
--   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
--   PRIMARY KEY (`id`),
--   KEY `idx_admin_id` (`admin_id`),
--   KEY `idx_action` (`action`),
--   KEY `idx_created_at` (`created_at`)
-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng log các hành động của admin';
-- ALTER TABLE `admin_users`
-- ADD COLUMN `permissions` JSON DEFAULT NULL AFTER `role`,
-- ADD COLUMN `login_attempts` INT DEFAULT 0 AFTER `is_active`,
-- ADD COLUMN `locked_until` TIMESTAMP NULL DEFAULT NULL AFTER `login_attempts`;
-- INSERT INTO `admin_users` (`username`, `password_hash`, `email`, `name`, `role`, `permissions`) VALUES
-- ('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@inis-hvnh.site', 'System Administrator', 'super_admin', '["*"]'),
-- ('moderator', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'moderator@inis-hvnh.site', 'Content Moderator', 'moderator', '["view_users", "view_processes", "view_uploads"]');

-- CREATE TABLE `admin_sessions` (
--   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
--   `admin_id` int NOT NULL,
--   `token` varchar(255) NOT NULL,
--   `ip_address` varchar(45) DEFAULT NULL,
--   `user_agent` text,
--   `expires_at` timestamp NOT NULL,
--   `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
--   PRIMARY KEY (`id`),
--   UNIQUE KEY `token` (`token`),
--   KEY `idx_admin_id` (`admin_id`),
--   KEY `idx_expires_at` (`expires_at`)
-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng quản lý session của admin';

ALTER TABLE `admin_users` ADD INDEX `idx_username_active` (`username`, `is_active`);
ALTER TABLE `admin_users` ADD INDEX `idx_role` (`role`);
ALTER TABLE `admin_audit_logs` ADD INDEX `idx_admin_action` (`admin_id`, `action`);