-- 創建測試數據庫
CREATE DATABASE IF NOT EXISTS g38_lottery_test;
USE g38_lottery_test;

-- 清理現有表 (如果存在)
DROP TABLE IF EXISTS users;

-- 創建用戶表
CREATE TABLE users (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(100) NOT NULL UNIQUE,
  phone VARCHAR(20) NOT NULL UNIQUE,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(20) NOT NULL DEFAULT 'user',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 插入測試用戶數據
-- 密碼為 "password123" 的 bcrypt 雜湊
INSERT INTO users (name, email, phone, password, role) VALUES 
('測試用戶', 'test@example.com', '0987654321', '$2a$10$XhpYF4QzSA3YihlfFM68MeEuNf/ZcD4Dw7aIl7XnzpUYA3xVXAKMS', 'user'),
('管理員', 'admin@example.com', '0912345678', '$2a$10$XhpYF4QzSA3YihlfFM68MeEuNf/ZcD4Dw7aIl7XnzpUYA3xVXAKMS', 'admin');

-- 可以在這裡添加更多測試用的表和數據 