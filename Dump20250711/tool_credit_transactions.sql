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
  `units_used` decimal(10,6) DEFAULT '0.000000',
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
) ENGINE=InnoDB AUTO_INCREMENT=29 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng lưu lịch sử giao dịch credit';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `credit_transactions`
--

LOCK TABLES `credit_transactions` WRITE;
/*!40000 ALTER TABLE `credit_transactions` DISABLE KEYS */;
INSERT INTO `credit_transactions` VALUES (1,2,'lock',0.06,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-08 07:43:56','2025-07-08 07:43:56'),(2,2,'deduct',0.01,0,'whisper','Whisper transcribe','per_minute',1.843374,NULL,'completed','','2025-07-08 07:43:56','2025-07-08 07:43:56'),(3,2,'deduct',0.03,0,'gemini','Gemini dịch SRT','per_token',407.000000,NULL,'completed','','2025-07-08 07:44:08','2025-07-08 07:44:08'),(4,2,'deduct',0.04,0,'tts','Google TTS','per_character',3301.000000,NULL,'completed','','2025-07-08 07:44:08','2025-07-08 07:44:08'),(5,2,'lock',0.06,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-08 08:00:14','2025-07-08 08:00:14'),(6,2,'deduct',0.01,0,'whisper','Whisper transcribe','per_minute',1.843374,2,'completed','','2025-07-08 08:00:14','2025-07-08 08:00:14'),(7,2,'deduct',0.03,0,'gemini','Gemini dịch SRT','per_token',407.000000,2,'completed','','2025-07-08 08:00:25','2025-07-08 08:00:25'),(8,2,'deduct',0.04,0,'tts','Google TTS','per_character',3325.000000,2,'completed','','2025-07-08 08:00:25','2025-07-08 08:00:25'),(9,2,'lock',0.06,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-08 08:08:53','2025-07-08 08:08:53'),(10,2,'deduct',0.01,0,'whisper','Whisper transcribe','per_minute',1.843374,3,'completed','','2025-07-08 08:08:53','2025-07-08 08:08:53'),(11,2,'deduct',0.03,0,'gemini','Gemini dịch SRT','per_token',407.000000,3,'completed','','2025-07-08 08:09:03','2025-07-08 08:09:03'),(12,2,'deduct',0.04,0,'tts','Google TTS','per_character',3259.000000,3,'completed','','2025-07-08 08:09:03','2025-07-08 08:09:03'),(13,2,'lock',0.06,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-08 08:14:13','2025-07-08 08:14:13'),(14,2,'deduct',0.02,0.011060244899999998,'whisper','Whisper transcribe','per_minute',1.843374,4,'completed','','2025-07-08 08:14:13','2025-07-08 08:14:13'),(15,2,'deduct',0.04,0.030524999999999997,'gemini','Gemini dịch SRT','per_token',407.000000,4,'completed','','2025-07-08 08:14:23','2025-07-08 08:14:23'),(16,2,'deduct',0.06,0.04272,'tts','Google TTS','per_character',3259.000000,4,'completed','','2025-07-08 08:14:23','2025-07-08 08:14:23'),(17,2,'lock',0.08,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 08:59:54','2025-07-11 08:59:54'),(18,2,'deduct',0.03,0.020503510200000002,'whisper','Whisper transcribe','per_minute',3.417252,5,'completed','','2025-07-11 08:59:54','2025-07-11 08:59:54'),(19,2,'lock',0.09,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 10:25:44','2025-07-11 10:25:44'),(20,2,'deduct',0.03,0.0205061224,'whisper','Whisper transcribe','per_minute',3.417687,6,'completed','','2025-07-11 10:25:44','2025-07-11 10:25:44'),(21,2,'lock',0.09,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 10:37:22','2025-07-11 10:37:22'),(22,2,'deduct',0.03,0.0205061224,'whisper','Whisper transcribe','per_minute',3.417687,7,'completed','','2025-07-11 10:37:22','2025-07-11 10:37:22'),(23,2,'lock',0.09,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 10:43:03','2025-07-11 10:43:03'),(24,2,'deduct',0.03,0.0205061224,'whisper','Whisper transcribe','per_minute',3.417687,8,'completed','','2025-07-11 10:43:03','2025-07-11 10:43:03'),(25,2,'lock',0.09,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 10:46:20','2025-07-11 10:46:20'),(26,2,'deduct',0.03,0.0205061224,'whisper','Whisper transcribe','per_minute',3.417687,9,'completed','','2025-07-11 10:46:20','2025-07-11 10:46:20'),(27,2,'lock',0.09,0,'process-video','Lock credit for video processing','',0.000000,NULL,'completed','','2025-07-11 10:49:35','2025-07-11 10:49:35'),(28,2,'deduct',0.03,0.0205061224,'whisper','Whisper transcribe','per_minute',3.417687,10,'completed','','2025-07-11 10:49:35','2025-07-11 10:49:35');
/*!40000 ALTER TABLE `credit_transactions` ENABLE KEYS */;
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
