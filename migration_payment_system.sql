-- Migration cho hệ thống thanh toán QR động
-- Chạy lệnh: mysql -u root -p tool < migration_payment_system.sql

-- Bảng đơn hàng thanh toán
CREATE TABLE IF NOT EXISTS `payment_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `order_code` varchar(50) NOT NULL,
  `amount_vnd` decimal(12,0) NOT NULL,
  `amount_usd` decimal(10,2) NOT NULL,
  `exchange_rate` decimal(10,4) NOT NULL,
  `bank_account` varchar(50) NOT NULL,
  `bank_name` varchar(100) NOT NULL,
  `qr_code_url` varchar(500) DEFAULT NULL,
  `qr_code_data` text DEFAULT NULL,
  `order_status` enum('pending','paid','expired','cancelled') DEFAULT 'pending',
  `payment_method` enum('qr_code','bank_transfer') DEFAULT 'qr_code',
  `expires_at` timestamp NOT NULL,
  `paid_at` timestamp NULL DEFAULT NULL,
  `transaction_id` varchar(100) DEFAULT NULL,
  `email_confirmation` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_code` (`order_code`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_order_status` (`order_status`),
  KEY `idx_expires_at` (`expires_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu thông tin đơn hàng thanh toán';

-- Bảng tài khoản ngân hàng
CREATE TABLE IF NOT EXISTS `bank_accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `bank_name` varchar(100) NOT NULL,
  `account_number` varchar(50) NOT NULL,
  `account_name` varchar(200) NOT NULL,
  `bank_code` varchar(20) NOT NULL,
  `is_active` boolean DEFAULT true,
  `daily_limit` decimal(12,0) DEFAULT 100000000,
  `monthly_limit` decimal(12,0) DEFAULT 1000000000,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account_number` (`account_number`),
  KEY `idx_bank_name` (`bank_name`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu thông tin tài khoản ngân hàng';

-- Bảng template mail xác nhận thanh toán
CREATE TABLE IF NOT EXISTS `email_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `bank_name` varchar(100) NOT NULL,
  `template_name` varchar(100) NOT NULL,
  `subject_pattern` varchar(200) DEFAULT NULL,
  `sender_pattern` varchar(200) DEFAULT NULL,
  `amount_pattern` varchar(100) DEFAULT NULL,
  `account_pattern` varchar(100) DEFAULT NULL,
  `content_pattern` text DEFAULT NULL,
  `is_active` boolean DEFAULT true,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_bank_name` (`bank_name`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu template mail xác nhận thanh toán';

-- Bảng log thanh toán
CREATE TABLE IF NOT EXISTS `payment_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL,
  `log_type` enum('order_created','qr_generated','payment_detected','payment_confirmed','payment_failed') NOT NULL,
  `message` text NOT NULL,
  `metadata` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_log_type` (`log_type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu log thanh toán';

-- Thêm dữ liệu mẫu cho tài khoản ngân hàng
INSERT INTO `bank_accounts` (`bank_name`, `account_number`, `account_name`, `bank_code`, `is_active`, `daily_limit`, `monthly_limit`) VALUES
('Vietcombank', '1234567890', 'NGUYEN VAN A', 'VCB', true, 100000000, 1000000000),
('BIDV', '0987654321', 'NGUYEN VAN A', 'BIDV', true, 100000000, 1000000000),
('Techcombank', '1122334455', 'NGUYEN VAN A', 'TCB', true, 100000000, 1000000000)
ON DUPLICATE KEY UPDATE `is_active` = VALUES(`is_active`);

-- Thêm dữ liệu mẫu cho email templates
INSERT INTO `email_templates` (`bank_name`, `template_name`, `subject_pattern`, `sender_pattern`, `amount_pattern`, `account_pattern`, `content_pattern`, `is_active`) VALUES
('Vietcombank', 'payment_confirmation', 'Thông báo giao dịch thành công', 'noreply@vietcombank.com.vn', 'Số tiền:\\s*([\\d,]+)\\s*VND', 'Tài khoản:\\s*(\\d+)', 'Giao dịch chuyển tiền thành công', true),
('BIDV', 'payment_confirmation', 'Thông báo giao dịch', 'noreply@bidv.com.vn', 'Số tiền:\\s*([\\d,]+)\\s*VNĐ', 'Tài khoản:\\s*(\\d+)', 'Giao dịch thành công', true),
('Techcombank', 'payment_confirmation', 'Xác nhận giao dịch', 'noreply@tcb.com.vn', 'Số tiền:\\s*([\\d,]+)\\s*VND', 'Tài khoản:\\s*(\\d+)', 'Giao dịch hoàn tất', true)
ON DUPLICATE KEY UPDATE `is_active` = VALUES(`is_active`);

-- Tạo index cho performance
CREATE INDEX `idx_payment_orders_user_status` ON `payment_orders` (`user_id`, `order_status`);
CREATE INDEX `idx_payment_orders_expires` ON `payment_orders` (`expires_at`, `order_status`);
CREATE INDEX `idx_payment_logs_order_type` ON `payment_logs` (`order_id`, `log_type`);

'1','Vietcombank','1020760830','PHAN VAN TUAN','VCB','0','100000000','1000000000','2025-07-31 14:45:56','970436'
'2','BIDV','0987654321','NGUYEN VAN A','BIDV','0','100000000','1000000000','2025-07-31 14:45:56',''
'3','Techcombank','19038740478018','PHAN VAN TUAN','TCB','0','100000000','1000000000','2025-07-31 14:45:56','970407'
'4','ACB','21360471','PHAN VAN TUAN','ACB','1','100000000','1000000000','2025-07-31 14:45:56','970416'
