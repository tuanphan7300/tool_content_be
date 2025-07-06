#!/bin/bash

# Script kiểm tra trạng thái database
# Sử dụng: ./check_database.sh

echo "🔍 Kiểm tra trạng thái database..."

# Đọc config từ env hoặc sử dụng default
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"3306"}
DB_USER=${DB_USER:-"root"}
DB_PASSWORD=${DB_PASSWORD:-"root"}
DB_NAME=${DB_NAME:-"tool"}

# Tạo file SQL tạm để kiểm tra
cat > /tmp/check_db.sql << 'EOF'
-- Kiểm tra database
SELECT 
    SCHEMA_NAME as 'Database',
    DEFAULT_CHARACTER_SET_NAME as 'Charset',
    DEFAULT_COLLATION_NAME as 'Collation'
FROM information_schema.SCHEMATA 
WHERE SCHEMA_NAME = 'tool';

-- Kiểm tra các bảng
SELECT 
    TABLE_NAME as 'Table',
    TABLE_ROWS as 'Rows',
    DATA_LENGTH as 'Data Size (bytes)',
    INDEX_LENGTH as 'Index Size (bytes)',
    (DATA_LENGTH + INDEX_LENGTH) as 'Total Size (bytes)'
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA = 'tool' 
ORDER BY TABLE_NAME;

-- Kiểm tra foreign keys
SELECT 
    CONSTRAINT_NAME as 'Constraint',
    TABLE_NAME as 'Table',
    COLUMN_NAME as 'Column',
    REFERENCED_TABLE_NAME as 'References Table',
    REFERENCED_COLUMN_NAME as 'References Column'
FROM information_schema.KEY_COLUMN_USAGE 
WHERE TABLE_SCHEMA = 'tool' 
AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- Kiểm tra dữ liệu test
SELECT 'Users count:' as 'Info', COUNT(*) as 'Count' FROM users
UNION ALL
SELECT 'User tokens count:', COUNT(*) FROM user_tokens
UNION ALL
SELECT 'Token transactions count:', COUNT(*) FROM token_transactions
UNION ALL
SELECT 'Caption histories count:', COUNT(*) FROM caption_histories;

-- Kiểm tra user test
SELECT 
    id, email, name, auth_provider, email_verified, created_at
FROM users 
WHERE email = 'test@example.com';

-- Kiểm tra token của user test
SELECT 
    ut.user_id,
    u.email,
    ut.total_tokens,
    ut.used_tokens,
    (ut.total_tokens - ut.used_tokens) as available_tokens
FROM user_tokens ut
JOIN users u ON ut.user_id = u.id
WHERE u.email = 'test@example.com';
EOF

# Chạy kiểm tra
echo "📊 Thông tin database:"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" < /tmp/check_db.sql

# Xóa file tạm
rm /tmp/check_db.sql

echo ""
echo "✅ Kiểm tra hoàn tất!"
echo ""
echo "💡 Nếu thấy lỗi, hãy chạy migration trước:"
echo "   ./run_migration.sh" 