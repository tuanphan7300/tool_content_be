# Database Migration Guide

Hướng dẫn thiết lập database cho Tool Content Backend.

## 📋 Yêu cầu

- MySQL 5.7+ hoặc MariaDB 10.2+
- MySQL client (mysql command line tool)
- Quyền tạo database và bảng

## 🚀 Cách chạy migration

### Phương pháp 1: Sử dụng script tự động (Khuyến nghị)

```bash
# Cấp quyền thực thi cho script
chmod +x run_migration.sh

# Chạy migration
./run_migration.sh
```

### Phương pháp 2: Chạy thủ công

```bash
# Kết nối trực tiếp vào MySQL
mysql -h localhost -P 3306 -u root -p < migration_init_database.sql
```

### Phương pháp 3: Sử dụng Docker

```bash
# Nếu sử dụng Docker MySQL
docker exec -i mysql_container mysql -u root -p < migration_init_database.sql
```

## ⚙️ Cấu hình database

### Environment Variables

Tạo file `.env` với các thông tin sau:

```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
```

### Docker Compose (Nếu sử dụng)

```yaml
version: '3.8'
services:
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: tool
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:
```

## 📊 Cấu trúc Database

### Bảng `users`
- Lưu thông tin người dùng
- Hỗ trợ cả local auth và Google OAuth
- Fields: id, email, password_hash, google_id, name, picture, email_verified, auth_provider

### Bảng `user_tokens`
- Quản lý số dư token của user
- Fields: id, user_id, total_tokens, used_tokens

### Bảng `token_transactions`
- Lịch sử giao dịch token
- Fields: id, user_id, type, amount, description, service, video_id

### Bảng `caption_histories`
- Lịch sử xử lý video
- Fields: id, user_id, video_filename, transcript, segments, srt_file, tts_file, merged_video_file

## 🔍 Kiểm tra migration

Sau khi chạy migration, bạn có thể kiểm tra:

```sql
-- Xem danh sách bảng
SHOW TABLES;

-- Xem cấu trúc bảng users
DESCRIBE users;

-- Kiểm tra dữ liệu test
SELECT * FROM users WHERE email = 'test@example.com';
SELECT * FROM user_tokens WHERE user_id = 1;
```

## 🛠️ Troubleshooting

### Lỗi kết nối database
```bash
# Kiểm tra MySQL có đang chạy không
sudo systemctl status mysql

# Khởi động MySQL nếu cần
sudo systemctl start mysql
```

### Lỗi quyền truy cập
```sql
-- Tạo user mới với quyền đầy đủ
CREATE USER 'tool_user'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON tool.* TO 'tool_user'@'localhost';
FLUSH PRIVILEGES;
```

### Lỗi charset
```sql
-- Kiểm tra charset hiện tại
SHOW VARIABLES LIKE 'character_set%';

-- Set charset nếu cần
SET NAMES utf8mb4;
```

## 📝 Rollback (Nếu cần)

```sql
-- Xóa toàn bộ database (CẨN THẬN!)
DROP DATABASE IF EXISTS tool;

-- Hoặc xóa từng bảng
DROP TABLE IF EXISTS token_transactions;
DROP TABLE IF EXISTS user_tokens;
DROP TABLE IF EXISTS caption_histories;
DROP TABLE IF EXISTS users;
```

## ✅ Verification

Sau khi migration thành công, bạn sẽ thấy:

1. ✅ Database `tool` được tạo
2. ✅ 4 bảng chính được tạo với đầy đủ indexes
3. ✅ Foreign keys được thiết lập đúng
4. ✅ User test được tạo với 1000 token
5. ✅ Ứng dụng có thể kết nối và hoạt động bình thường

## 🎯 Next Steps

Sau khi migration thành công:

1. Cập nhật file `.env` với thông tin database
2. Chạy ứng dụng: `go run main.go`
3. Test đăng nhập với user: `test@example.com`
4. Upload video để test toàn bộ workflow

# Database Migration: Thêm trường video_file_name_origin

## Mô tả
Migration này thêm trường `video_file_name_origin` vào bảng `caption_histories` để lưu tên file gốc của video, thay vì tên file đã được xử lý (có timestamp).

## Lý do thực hiện
- API `/process-video` hiện tại đang lưu tên file với timestamp (ví dụ: `1234567890_video.mp4`) vào trường `video_filename`
- Người dùng muốn thấy tên file gốc (ví dụ: `video.mp4`) trong lịch sử
- Trường `video_file_name_origin` sẽ lưu tên file gốc, `video_filename` vẫn lưu tên file đã xử lý

## Cách thực hiện migration

### Cách 1: Sử dụng script tự động
```bash
# Chạy script migration
./run_migration.sh
```

### Cách 2: Chạy SQL thủ công
```sql
-- Thêm cột mới
ALTER TABLE caption_histories 
ADD COLUMN video_file_name_origin VARCHAR(255) DEFAULT NULL 
AFTER video_filename;

-- Cập nhật dữ liệu cũ
UPDATE caption_histories 
SET video_file_name_origin = video_filename 
WHERE video_file_name_origin IS NULL;
```

## Thay đổi trong code

### Backend (Go)
1. **config/database.go**: Thêm trường `VideoFilenameOrigin` vào struct `CaptionHistory`
2. **handler/process.go**: Cập nhật `ProcessVideoHandler` và `ProcessHandler` để lưu tên file gốc
3. **handler/history.go**: Cập nhật API `/history` để trả về tên file gốc
4. **handler/text_to_speech.go**: Cập nhật để lưu tên file gốc

### Frontend (Vue.js)
1. **components/HistoryList.vue**: Hiển thị `video_file_name_origin` thay vì `video_filename`

## Kiểm tra sau migration
1. Chạy API `/process-video` với một file mới
2. Kiểm tra API `/history` có trả về `video_file_name_origin` không
3. Kiểm tra frontend hiển thị tên file gốc đúng không

## Lưu ý
- Migration này tương thích ngược với dữ liệu cũ
- Dữ liệu cũ sẽ có `video_file_name_origin` = `video_filename`
- Dữ liệu mới sẽ có `video_file_name_origin` = tên file gốc, `video_filename` = tên file đã xử lý 