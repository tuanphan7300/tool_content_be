#!/bin/bash

# Script fix lỗi database cho Tool Content Backend
# Sử dụng: ./fix_database.sh

echo "🔧 Fix lỗi database..."

# Đọc config từ env hoặc sử dụng default
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"3306"}
DB_USER=${DB_USER:-"root"}
DB_PASSWORD=${DB_PASSWORD:-"root"}

echo "📊 Thông tin kết nối database:"
echo "   Host: $DB_HOST"
echo "   Port: $DB_PORT"
echo "   User: $DB_USER"

# Kiểm tra kết nối database
echo "🔍 Kiểm tra kết nối database..."
if ! mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" >/dev/null 2>&1; then
    echo "❌ Không thể kết nối đến database"
    echo "   Hãy kiểm tra thông tin kết nối và đảm bảo MySQL đang chạy"
    exit 1
fi

echo "✅ Kết nối database thành công"

# Chạy fix script
echo "🔧 Đang fix lỗi database..."
if mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" < fix_database.sql; then
    echo "✅ Fix database thành công!"
    echo ""
    echo "📋 Các thay đổi đã thực hiện:"
    echo "   - Xóa index có vấn đề trên video_filename"
    echo "   - Tạo lại index với độ dài key được chỉ định"
    echo "   - Kiểm tra và tạo lại các index khác"
    echo ""
    echo "🚀 Bây giờ bạn có thể chạy ứng dụng:"
    echo "   go run main.go"
else
    echo "❌ Fix database thất bại!"
    echo "   Hãy kiểm tra log lỗi ở trên"
    exit 1
fi 