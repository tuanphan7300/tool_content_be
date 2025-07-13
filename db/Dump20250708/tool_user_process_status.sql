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
-- Table structure for table `user_process_status`
--

DROP TABLE IF EXISTS `user_process_status`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_process_status` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `status` enum('processing','completed','failed','cancelled') NOT NULL DEFAULT 'processing',
  `process_type` enum('process','process-video','process-voice','process-background') NOT NULL,
  `started_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` timestamp NULL DEFAULT NULL,
  `video_id` bigint unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Bảng theo dõi trạng thái process của user để tránh spam';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_process_status`
--

LOCK TABLES `user_process_status` WRITE;
/*!40000 ALTER TABLE `user_process_status` DISABLE KEYS */;
INSERT INTO `user_process_status` VALUES (1,2,'completed','process-video','2025-07-08 07:43:45','2025-07-08 07:45:19',1,'2025-07-08 07:43:45','2025-07-08 07:45:19'),(2,2,'completed','process-video','2025-07-08 08:00:03','2025-07-08 08:01:37',2,'2025-07-08 08:00:03','2025-07-08 08:01:37'),(3,2,'completed','process-video','2025-07-08 08:08:12',NULL,NULL,'2025-07-08 08:08:12','2025-07-08 08:08:35'),(4,2,'completed','process-video','2025-07-08 08:08:41','2025-07-08 08:10:14',3,'2025-07-08 08:08:41','2025-07-08 08:10:14'),(5,2,'completed','process-video','2025-07-08 08:13:59','2025-07-08 08:15:33',4,'2025-07-08 08:13:59','2025-07-08 08:15:33');
/*!40000 ALTER TABLE `user_process_status` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-07-08 17:47:54
