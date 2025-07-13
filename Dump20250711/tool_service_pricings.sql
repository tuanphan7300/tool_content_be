-- MySQL dump 10.13  Distrib 8.0.38, for macos14 (arm64)
--
-- Host: 127.0.0.1    Database: tool
-- ------------------------------------------------------
-- Server version	8.0.42

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
-- Table structure for table `service_pricings`
--

DROP TABLE IF EXISTS `service_pricings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_pricings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `service_name` varchar(50) NOT NULL,
  `model_api_name` varchar(45) DEFAULT NULL,
  `pricing_type` enum('per_minute','per_token','per_character') NOT NULL,
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
INSERT INTO `service_pricings` VALUES (1,'whisper',NULL,'per_minute',0.006000,'USD','OpenAI Whisper API - $0.006 per minute',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(2,'gemini_1.5_flash','gemini-1.5-flash','per_token',0.000075,'USD','Google Gemini 1.5 Flash - $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-11 10:53:03'),(3,'tts_standard',NULL,'per_character',0.000004,'USD','Google TTS Standard - $4.00 per 1M characters',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(4,'tts_wavenet',NULL,'per_character',0.000016,'USD','Google TTS Wavenet - $16.00 per 1M characters',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(5,'gpt_3.5_turbo',NULL,'per_token',0.000002,'USD','OpenAI GPT-3.5 Turbo - $0.002 per 1K tokens',1,'2025-07-08 03:26:07','2025-07-08 03:26:07'),(6,'gemini_2.5_pro','gemini-2.5-pro','per_token',0.000075,'USD','Google Gemini 2.5 Pro- $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-11 10:48:42'),(7,'gemini_2.0_flash','gemini_2.0_flash','per_token',0.000075,'USD','Google Gemini 2.0 Flash- $0.075 per 1M tokens',1,'2025-07-08 03:26:07','2025-07-11 10:41:25');
/*!40000 ALTER TABLE `service_pricings` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-07-11 17:53:59
