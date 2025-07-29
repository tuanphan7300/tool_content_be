-- Migration: Add admin functionality
-- Date: 2025-07-29

-- Add is_admin column to users table
ALTER TABLE `users` ADD COLUMN `is_admin` TINYINT(1) DEFAULT 0 AFTER `auth_provider`;

-- Create admin_users table for admin accounts
CREATE TABLE `admin_users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(255) NOT NULL,
  `password_hash` text NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `role` enum('super_admin','admin','moderator') DEFAULT 'admin',
  `is_active` tinyint(1) DEFAULT 1,
  `last_login` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng quản lý tài khoản admin';

-- Insert default admin account (password: admin123)
INSERT INTO `admin_users` (`username`, `password_hash`, `email`, `name`, `role`) 
VALUES ('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@inis-hvnh.site', 'System Administrator', 'super_admin');

-- Create admin_audit_logs table for tracking admin actions
CREATE TABLE `admin_audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` int NOT NULL,
  `action` varchar(100) NOT NULL,
  `table_name` varchar(100) DEFAULT NULL,
  `record_id` bigint unsigned DEFAULT NULL,
  `old_values` json DEFAULT NULL,
  `new_values` json DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_admin_id` (`admin_id`),
  KEY `idx_action` (`action`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng log các hành động của admin'; 