-- MySQL/MariaDB database backup
-- Database: hdns
-- Generated: 2026-01-19T10:14:57+01:00

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;


-- Table structure for gorp_migrations
DROP TABLE IF EXISTS `gorp_migrations`;
CREATE TABLE `gorp_migrations` (
  `id` varchar(255) NOT NULL,
  `applied_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_uca1400_ai_ci;

-- Data for table gorp_migrations
SET FOREIGN_KEY_CHECKS = 1;
