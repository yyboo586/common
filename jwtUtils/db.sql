CREATE DATABASE IF NOT EXISTS `test_db`;

CREATE TABLE `test_db`.`t_jwt_keys` (
  `id` varchar(36) NOT NULL,
  `data` text NOT NULL,
  `sid` varchar(36) NOT NULL COMMENT 'set id', 
  `created_at` timestamp DEFAULT current_timestamp,
  PRIMARY KEY (`id`),
  KEY `idx_sid` (`sid`),
  KEY `idx_created_at` (`created_at`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;