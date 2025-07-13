-- Migration để sửa cấu trúc cột units_used trong bảng credit_transactions
-- Thay đổi từ decimal(10,6) thành decimal(15,6) để hỗ trợ số lớn hơn

USE tool;

-- Thay đổi cấu trúc cột units_used
ALTER TABLE credit_transactions 
MODIFY COLUMN units_used DECIMAL(15,6) DEFAULT 0.000000;

-- Kiểm tra cấu trúc sau khi thay đổi
DESCRIBE credit_transactions; 