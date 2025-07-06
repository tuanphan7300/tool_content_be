#!/bin/bash

# Script kiểm tra cấu hình Google OAuth
# Sử dụng: ./check_oauth.sh

echo "🔍 Kiểm tra cấu hình Google OAuth..."

# Kiểm tra file .env
if [ ! -f ".env" ]; then
    echo "❌ Không tìm thấy file .env"
    echo "   Hãy tạo file .env từ env.example"
    exit 1
fi

echo "✅ File .env tồn tại"

# Đọc các biến OAuth từ .env
source .env

# Kiểm tra các biến bắt buộc
echo ""
echo "📋 Kiểm tra environment variables:"

# Kiểm tra GOOGLE_CLIENT_ID
if [ -z "$GOOGLE_CLIENT_ID" ]; then
    echo "❌ GOOGLE_CLIENT_ID chưa được cấu hình"
else
    echo "✅ GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID:0:20}..."
fi

# Kiểm tra GOOGLE_CLIENT_SECRET
if [ -z "$GOOGLE_CLIENT_SECRET" ]; then
    echo "❌ GOOGLE_CLIENT_SECRET chưa được cấu hình"
else
    echo "✅ GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET:0:20}..."
fi

# Kiểm tra GOOGLE_REDIRECT_URL
if [ -z "$GOOGLE_REDIRECT_URL" ]; then
    echo "❌ GOOGLE_REDIRECT_URL chưa được cấu hình"
else
    echo "✅ GOOGLE_REDIRECT_URL: $GOOGLE_REDIRECT_URL"
fi

# Kiểm tra JWTACCESSKEY
if [ -z "$JWTACCESSKEY" ]; then
    echo "❌ JWTACCESSKEY chưa được cấu hình"
else
    echo "✅ JWTACCESSKEY: ${JWTACCESSKEY:0:10}..."
fi

echo ""
echo "🔗 Test OAuth endpoints:"

# Kiểm tra xem server có đang chạy không
if curl -s http://localhost:8888/ping > /dev/null 2>&1; then
    echo "✅ Backend server đang chạy"
    
    # Test OAuth login endpoint
    echo "📡 Testing OAuth login endpoint..."
    OAUTH_RESPONSE=$(curl -s http://localhost:8888/auth/google/login)
    
    if echo "$OAUTH_RESPONSE" | grep -q "auth_url"; then
        echo "✅ OAuth login endpoint hoạt động"
        echo "   Auth URL: $(echo "$OAUTH_RESPONSE" | grep -o '"auth_url":"[^"]*"' | cut -d'"' -f4)"
    else
        echo "❌ OAuth login endpoint có lỗi"
        echo "   Response: $OAUTH_RESPONSE"
    fi
else
    echo "❌ Backend server không chạy"
    echo "   Hãy chạy: go run main.go"
fi

echo ""
echo "📊 Kiểm tra database connection:"

# Kiểm tra kết nối database
if mysql -h"${DB_HOST:-localhost}" -P"${DB_PORT:-3306}" -u"${DB_USER:-root}" -p"${DB_PASSWORD:-root}" -e "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Database connection OK"
    
    # Kiểm tra bảng users
    USER_COUNT=$(mysql -h"${DB_HOST:-localhost}" -P"${DB_PORT:-3306}" -u"${DB_USER:-root}" -p"${DB_PASSWORD:-root}" -s -N -e "SELECT COUNT(*) FROM tool.users;" 2>/dev/null)
    if [ "$USER_COUNT" -ge 0 ]; then
        echo "✅ Bảng users tồn tại ($USER_COUNT users)"
    else
        echo "❌ Bảng users không tồn tại"
    fi
else
    echo "❌ Không thể kết nối database"
fi

echo ""
echo "🎯 Recommendations:"

# Đưa ra gợi ý dựa trên kết quả kiểm tra
if [ -z "$GOOGLE_CLIENT_ID" ] || [ -z "$GOOGLE_CLIENT_SECRET" ]; then
    echo "📝 Cần thiết lập Google OAuth:"
    echo "   1. Tạo Google Cloud Project"
    echo "   2. Enable Google+ API"
    echo "   3. Tạo OAuth 2.0 Client ID"
    echo "   4. Cập nhật .env file"
    echo "   Xem chi tiết: cat GOOGLE_OAUTH_SETUP.md"
fi

if [ -z "$JWTACCESSKEY" ]; then
    echo "🔐 Cần thiết lập JWT:"
    echo "   Thêm JWTACCESSKEY vào .env file"
fi

echo ""
echo "📚 Tài liệu tham khảo:"
echo "   - GOOGLE_OAUTH_SETUP.md: Hướng dẫn chi tiết"
echo "   - MIGRATION_README.md: Hướng dẫn database"
echo ""
echo "✅ Kiểm tra hoàn tất!" 