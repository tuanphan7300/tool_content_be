-- MySQL dump 10.13  Distrib 8.0.34, for macos13 (arm64)
--
-- Host: 127.0.0.1    Database: tool
-- ------------------------------------------------------
-- Server version	9.3.0

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `caption_histories`
--

DROP TABLE IF EXISTS `caption_histories`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `caption_histories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `video_filename` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `video_filename_origin` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `transcript` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `suggestion` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `segments` json DEFAULT NULL,
  `segments_vi` json DEFAULT NULL,
  `timestamps` json DEFAULT NULL,
  `background_music` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `srt_file` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `original_srt_file` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tts_file` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `merged_video_file` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `process_type` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `hook_score` int DEFAULT '0',
  `viral_potential` int DEFAULT '0',
  `trending_hashtags` json DEFAULT NULL,
  `suggested_caption` text COLLATE utf8mb4_unicode_ci,
  `best_posting_time` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `optimization_tips` json DEFAULT NULL,
  `engagement_prompts` json DEFAULT NULL,
  `call_to_action` text COLLATE utf8mb4_unicode_ci,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `caption_histories`
--

LOCK TABLES `caption_histories` WRITE;
/*!40000 ALTER TABLE `caption_histories` DISABLE KEYS */;
/*!40000 ALTER TABLE `caption_histories` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `credit_transactions`
--

DROP TABLE IF EXISTS `credit_transactions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `credit_transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `transaction_type` enum('add','deduct','lock','unlock','refund') NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `base_amount` double DEFAULT '0',
  `service` varchar(50) DEFAULT NULL,
  `description` text,
  `pricing_type` varchar(20) DEFAULT NULL,
  `units_used` decimal(13,6) DEFAULT NULL,
  `video_id` bigint unsigned DEFAULT NULL,
  `transaction_status` enum('pending','completed','failed','refunded') DEFAULT 'completed',
  `reference_id` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_transaction_type` (`transaction_type`),
  KEY `idx_service` (`service`),
  KEY `idx_video_id` (`video_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu lịch sử giao dịch credit';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `credit_transactions`
--

LOCK TABLES `credit_transactions` WRITE;
/*!40000 ALTER TABLE `credit_transactions` DISABLE KEYS */;
/*!40000 ALTER TABLE `credit_transactions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pricing_tiers`
--

DROP TABLE IF EXISTS `pricing_tiers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pricing_tiers` (
  `id` int NOT NULL,
  `name` varchar(50) NOT NULL,
  `base_markup` decimal(5,2) NOT NULL,
  `monthly_limit` int DEFAULT NULL,
  `subscription_price` decimal(10,2) DEFAULT '0.00',
  `is_active` tinyint(1) DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `description` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  KEY `idx_name` (`name`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu thông tin các tier pricing';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pricing_tiers`
--

LOCK TABLES `pricing_tiers` WRITE;
/*!40000 ALTER TABLE `pricing_tiers` DISABLE KEYS */;
INSERT INTO `pricing_tiers` VALUES (1,'free',20.00,NULL,0.00,1,'2025-07-08 06:09:34','2025-07-08 06:09:34','Free tier with 20% markup');
/*!40000 ALTER TABLE `pricing_tiers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_config`
--

DROP TABLE IF EXISTS `service_config`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_config` (
  `id` int NOT NULL AUTO_INCREMENT,
  `service_type` varchar(64) NOT NULL,
  `service_name` varchar(64) NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `config_json` text,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_service_type` (`service_type`,`service_name`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_config`
--

LOCK TABLES `service_config` WRITE;
/*!40000 ALTER TABLE `service_config` DISABLE KEYS */;
INSERT INTO `service_config` VALUES (1,'srt_translation','gemini-1.5-flash',0,NULL,'2025-07-12 10:00:00','2025-07-24 23:58:02'),(2,'srt_translation','gemini-2.0-flash',1,NULL,'2025-07-12 10:00:00','2025-07-24 23:58:02'),(3,'speech_to_text	','whisper',1,NULL,'2025-07-12 10:00:00','2025-07-12 10:00:00'),(4,'text_to_speech','tts_wavenet',1,NULL,'2025-07-12 10:00:00','2025-07-12 10:00:00'),(5,'srt_translation','gemini_2.0_flash',0,NULL,'2025-07-12 10:00:00','2025-07-11 17:45:23');
/*!40000 ALTER TABLE `service_config` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_markups`
--

DROP TABLE IF EXISTS `service_markups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_markups` (
  `service_name` varchar(50) NOT NULL,
  `base_markup` decimal(5,2) NOT NULL,
  `premium_markup` decimal(5,2) DEFAULT '0.00',
  `description` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`service_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu markup cho từng service';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_markups`
--

LOCK TABLES `service_markups` WRITE;
/*!40000 ALTER TABLE `service_markups` DISABLE KEYS */;
INSERT INTO `service_markups` VALUES ('background',20.00,0.00,'Background music markup','2025-07-08 06:04:20','2025-07-08 06:06:21'),('gemini',20.00,0.00,'Gemini translation markup','2025-07-08 06:04:20','2025-07-08 06:06:21'),('process-video',20.00,0.00,'Process video markup','2025-07-08 06:04:20','2025-07-08 06:06:21'),('tts',20.00,0.00,'TTS voice markup','2025-07-08 06:04:20','2025-07-08 06:06:21'),('whisper',20.00,0.00,'Whisper transcription markup','2025-07-08 06:04:20','2025-07-08 06:06:21');
/*!40000 ALTER TABLE `service_markups` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `service_pricings`
--

DROP TABLE IF EXISTS `service_pricings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_pricings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `service_name` varchar(50) NOT NULL,
  `model_api_name` varchar(45) DEFAULT NULL,
  `pricing_type` enum('per_minute','per_token','per_character','per_job') NOT NULL,
  `price_per_unit` decimal(10,6) NOT NULL,
  `currency` varchar(3) DEFAULT 'USD',
  `description` text,
  `is_active` tinyint(1) DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `service_name` (`service_name`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu giá các service API theo tài liệu chính thức';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `service_pricings`
--

LOCK TABLES `service_pricings` WRITE;
/*!40000 ALTER TABLE `service_pricings` DISABLE KEYS */;
INSERT INTO `service_pricings` VALUES (1,'whisper',NULL,'per_minute',0.006000,'USD','OpenAI Whisper API - $0.006 per minute',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(2,'gemini-1.5-flash','gemini-1.5-flash','per_token',0.000075,'USD','Google Gemini 1.5 Flash - $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-12 16:23:03'),(3,'tts_standard',NULL,'per_character',0.000004,'USD','Google TTS Standard - $4.00 per 1M characters',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(4,'tts_wavenet',NULL,'per_character',0.000016,'USD','Google TTS Wavenet - $16.00 per 1M characters',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(5,'gpt_3.5_turbo',NULL,'per_token',0.000002,'USD','OpenAI GPT-3.5 Turbo - $0.002 per 1K tokens',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(6,'gemini_2.5_pro','gemini-2.5-pro','per_token',0.000075,'USD','Google Gemini 2.5 Pro- $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-11 10:48:42'),(7,'gemini-2.0-flash','gemini-2.0-flash','per_token',0.000075,'USD','Google Gemini 2.0 Flash- $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-17 16:22:54'),(8,'burn-sub','burn-sub','per_job',0.100000,'USD',NULL,1,'2025-07-08 03:26:07','2025-07-27 10:46:09');
/*!40000 ALTER TABLE `service_pricings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_credits`
--

DROP TABLE IF EXISTS `user_credits`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_credits` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `total_credits` decimal(10,2) DEFAULT '0.00',
  `used_credits` decimal(10,2) DEFAULT '0.00',
  `locked_credits` decimal(10,2) DEFAULT '0.00',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `tier_id` int DEFAULT '1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu credit của user (USD)';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_credits`
--

LOCK TABLES `user_credits` WRITE;
/*!40000 ALTER TABLE `user_credits` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_credits` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_process_status`
--

DROP TABLE IF EXISTS `user_process_status`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_process_status` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `status` enum('processing','completed','failed','cancelled') NOT NULL DEFAULT 'processing',
  `process_type` enum('process','process-video','process-voice','process-background','tiktok-optimize','create-subtitle') NOT NULL,
  `started_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` timestamp NULL DEFAULT NULL,
  `video_id` bigint unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng theo dõi trạng thái process của user để tránh spam';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_process_status`
--

LOCK TABLES `user_process_status` WRITE;
/*!40000 ALTER TABLE `user_process_status` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_process_status` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email` varchar(255) NOT NULL,
  `password_hash` text NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `google_id` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `picture` text,
  `email_verified` tinyint(1) DEFAULT '0',
  `auth_provider` varchar(50) DEFAULT 'local',
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`),
  KEY `idx_users_google_id` (`google_id`),
  KEY `idx_users_auth_provider` (`auth_provider`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-07-28 21:55:22
