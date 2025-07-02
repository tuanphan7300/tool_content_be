# Google OAuth Setup Guide

## 1. Tạo Google OAuth Credentials

### Bước 1: Truy cập Google Cloud Console
1. Đi đến [Google Cloud Console](https://console.cloud.google.com/)
2. Tạo project mới hoặc chọn project hiện có

### Bước 2: Enable Google+ API
1. Vào "APIs & Services" > "Library"
2. Tìm và enable "Google+ API" hoặc "Google Identity API"

### Bước 3: Tạo OAuth 2.0 Credentials
1. Vào "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth 2.0 Client IDs"
3. Chọn "Web application"
4. Điền thông tin:
   - **Name**: Tên ứng dụng của bạn
   - **Authorized JavaScript origins**: 
     - `http://localhost:3000` (cho development)
     - `https://yourdomain.com` (cho production)
   - **Authorized redirect URIs**:
     - `http://localhost:8888/auth/google/callback` (cho development)
     - `https://yourdomain.com/auth/google/callback` (cho production)

### Bước 4: Lưu thông tin
- **Client ID**: Sẽ được sử dụng trong biến môi trường `GOOGLE_CLIENT_ID`
- **Client Secret**: Sẽ được sử dụng trong biến môi trường `GOOGLE_CLIENT_SECRET`

## 2. Cấu hình Environment Variables

Thêm các biến sau vào file `.env`:

```env
# Google OAuth2 Configuration
GOOGLE_CLIENT_ID=your_google_client_id_here
GOOGLE_CLIENT_SECRET=your_google_client_secret_here
GOOGLE_REDIRECT_URL=http://localhost:8888/auth/google/callback
```

## 3. Chạy Database Migration

Chạy script migration để cập nhật database schema:

```bash
# Kết nối vào MySQL và chạy migration
mysql -u your_username -p your_database < migration_add_google_oauth.sql
```

Hoặc chạy từng lệnh SQL trong file migration.

## 4. Test Google OAuth

1. Khởi động backend server
2. Truy cập frontend và click "Sign in with Google"
3. Popup sẽ mở và yêu cầu đăng nhập Google
4. Sau khi đăng nhập thành công, popup sẽ đóng và user sẽ được đăng nhập

## 5. Troubleshooting

### Lỗi "redirect_uri_mismatch"
- Kiểm tra lại Authorized redirect URIs trong Google Cloud Console
- Đảm bảo URL trong `GOOGLE_REDIRECT_URL` khớp với cấu hình

### Lỗi "popup blocked"
- Đảm bảo browser cho phép popup cho domain của bạn
- Kiểm tra cài đặt popup blocker

### Lỗi "invalid_client"
- Kiểm tra lại Client ID và Client Secret
- Đảm bảo đã enable đúng API

## 6. Production Deployment

Khi deploy lên production:

1. Cập nhật Authorized JavaScript origins và redirect URIs trong Google Cloud Console
2. Cập nhật `GOOGLE_REDIRECT_URL` trong environment variables
3. Đảm bảo HTTPS được sử dụng (Google OAuth yêu cầu HTTPS cho production)
4. Cập nhật CORS settings nếu cần

## 7. Security Considerations

1. **Client Secret**: Không bao giờ commit client secret vào source code
2. **HTTPS**: Luôn sử dụng HTTPS trong production
3. **State Parameter**: Implement proper state parameter validation
4. **Token Storage**: Lưu trữ token an toàn (không trong localStorage cho production)
5. **User Verification**: Luôn verify email từ Google trước khi tạo account 