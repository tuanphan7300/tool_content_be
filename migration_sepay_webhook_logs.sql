-- Migration cho bảng log Sepay webhook
-- Chạy lệnh: mysql -u root -p tool < migration_sepay_webhook_logs.sql

-- Bảng log Sepay webhook
CREATE TABLE IF NOT EXISTS `sepay_webhook_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_code` varchar(50) DEFAULT NULL,
  `amount` decimal(12,0) DEFAULT NULL,
  `status` varchar(50) DEFAULT NULL,
  `transaction_id` varchar(100) DEFAULT NULL,
  `signature` varchar(255) DEFAULT NULL,
  `timestamp` bigint DEFAULT NULL,
  `raw_payload` json DEFAULT NULL,
  `headers` json DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` varchar(500) DEFAULT NULL,
  `processing_status` enum('received','validated','processed','failed','ignored') DEFAULT 'received',
  `error_message` text DEFAULT NULL,
  `processing_time_ms` int DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_code` (`order_code`),
  KEY `idx_transaction_id` (`transaction_id`),
  KEY `idx_processing_status` (`processing_status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng log tất cả webhook từ Sepay';

-- Tạo index cho performance
CREATE INDEX `idx_sepay_logs_order_status` ON `sepay_webhook_logs` (`order_code`, `processing_status`);
CREATE INDEX `idx_sepay_logs_created_status` ON `sepay_webhook_logs` (`created_at`, `processing_status`); 