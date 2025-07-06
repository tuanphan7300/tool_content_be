# Google OAuth Setup Guide

Hướng dẫn thiết lập Google OAuth cho Tool Content Backend.

## 🚨 Lỗi hiện tại
```
Lỗi 401: invalid_client
The OAuth client was not found.
```

## 📋 Bước 1: Tạo Google Cloud Project

### 1.1 Truy cập Google Cloud Console
- Vào [Google Cloud Console](https://console.cloud.google.com/)
- Đăng nhập bằng tài khoản Google của bạn

### 1.2 Tạo Project mới
- Click "Select a project" → "New Project"
- Đặt tên project: `tool-content-backend` (hoặc tên khác)
- Click "Create"

### 1.3 Kích hoạt Google+ API
- Vào "APIs & Services" → "Library"
- Tìm "Google+ API" hoặc "Google Identity"
- Click "Enable"

## 🔑 Bước 2: Tạo OAuth 2.0 Credentials

### 2.1 Vào OAuth consent screen
- Vào "APIs & Services" → "OAuth consent screen"
- Chọn "External" → "Create"

### 2.2 Cấu hình OAuth consent screen
```
App name: Tool Content Backend
User support email: [email của bạn]
Developer contact information: [email của bạn]
```

### 2.3 Thêm scopes
- Click "Add or remove scopes"
- Chọn:
  - `.../auth/userinfo.email`
  - `.../auth/userinfo.profile`
- Click "Update"

### 2.4 Thêm test users (nếu cần)
- Trong "Test users"
- Click "Add Users"
- Thêm email của bạn: `tuanphan7396@gmail.com`

## 🔐 Bước 3: Tạo OAuth 2.0 Client ID

### 3.1 Vào Credentials
- Vào "APIs & Services" → "Credentials"
- Click "Create Credentials" → "OAuth client ID"

### 3.2 Chọn Application type
- Chọn "Web application"
- Đặt tên: `Tool Content Backend Web Client`

### 3.3 Cấu hình Authorized redirect URIs
Thêm các URI sau:

**Cho Development:**
```
http://localhost:8888/auth/google/callback
http://localhost:8080/auth/google/callback
```

**Cho Production (nếu có):**
```
https://yourdomain.com/auth/google/callback
```

### 3.4 Lưu thông tin
Sau khi tạo, bạn sẽ nhận được:
- **Client ID**: `123456789-abcdefghijklmnop.apps.googleusercontent.com`
- **Client Secret**: `GOCSPX-abcdefghijklmnopqrstuvwxyz`

## ⚙️ Bước 4: Cập nhật Environment Variables

### 4.1 Tạo file .env
```bash
cd tool_content_be
cp env.example .env
```

### 4.2 Cập nhật .env
```env
# Google OAuth Configuration
GOOGLE_CLIENT_ID=123456789-abcdefghijklmnop.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-abcdefghijklmnopqrstuvwxyz
GOOGLE_REDIRECT_URL=http://localhost:8888/auth/google/callback

# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root

# JWT Configuration
JWTACCESSKEY=your-secret-jwt-key-here

# API Keys
API_KEY=your-openai-api-key
GEMINI_KEY=your-gemini-api-key
```

## 🔄 Bước 5: Restart Application

```bash
# Dừng ứng dụng hiện tại (Ctrl+C)
# Chạy lại
go run main.go
```

## 🧪 Bước 6: Test OAuth

### 6.1 Test từ Frontend
- Mở browser: `http://localhost:8080`
- Click "Sign in with Google"
- Đăng nhập với `tuanphan7396@gmail.com`

### 6.2 Test trực tiếp API
```bash
# Test OAuth URL
curl http://localhost:8888/auth/google/login
```

## 🛠️ Troubleshooting

### Lỗi "OAuth client was not found"
- ✅ Kiểm tra Client ID và Client Secret đúng
- ✅ Đảm bảo redirect URI khớp chính xác
- ✅ Kiểm tra project đã được chọn đúng

### Lỗi "redirect_uri_mismatch"
- ✅ Kiểm tra GOOGLE_REDIRECT_URL trong .env
- ✅ Đảm bảo URI đã được thêm vào Google Console
- ✅ Không có khoảng trắng thừa

### Lỗi "access_denied"
- ✅ Thêm email vào test users
- ✅ Đảm bảo OAuth consent screen đã publish
- ✅ Kiểm tra scopes đã được thêm

### Lỗi "invalid_grant"
- ✅ Kiểm tra thời gian server
- ✅ Clear browser cache
- ✅ Thử lại sau vài phút

## 📱 Cấu hình cho Production

### 1. Cập nhật OAuth consent screen
- Vào "OAuth consent screen"
- Click "Publish App"
- Chọn "In production"

### 2. Cập nhật redirect URIs
- Thêm domain production
- Xóa localhost URIs nếu không cần

### 3. Cập nhật environment variables
```env
GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/google/callback
```

## 🔍 Debug OAuth Flow

### 1. Kiểm tra logs
```bash
# Xem logs của ứng dụng
tail -f logs/app.log
```

### 2. Test từng bước
```bash
# 1. Test OAuth URL
curl http://localhost:8888/auth/google/login

# 2. Test callback (thay thế CODE bằng code thực)
curl "http://localhost:8888/auth/google/callback?code=CODE&state=state_123"
```

### 3. Kiểm tra database
```sql
-- Kiểm tra user được tạo
SELECT * FROM users WHERE email = 'tuanphan7396@gmail.com';
```

## ✅ Checklist hoàn thành

- [ ] Google Cloud Project được tạo
- [ ] Google+ API được enable
- [ ] OAuth consent screen được cấu hình
- [ ] OAuth 2.0 Client ID được tạo
- [ ] Redirect URIs được thêm
- [ ] Environment variables được cập nhật
- [ ] Application được restart
- [ ] OAuth flow hoạt động

## 🆘 Nếu vẫn gặp lỗi

1. **Kiểm tra lại từng bước** trong checklist
2. **Xem logs** của ứng dụng để debug
3. **Test với Postman** để isolate vấn đề
4. **Tạo project mới** nếu cần thiết

## 📞 Support

Nếu vẫn gặp vấn đề, hãy cung cấp:
- Screenshot lỗi
- Logs từ console
- Thông tin Google Cloud Project (không có secret)
- Environment variables (che sensitive data) 