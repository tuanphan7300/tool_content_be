# Tính năng Chống Spam Process

## Mô tả
Tính năng này ngăn chặn user spam các endpoint xử lý video/audio bằng cách mở nhiều tab hoặc gửi nhiều request đồng thời.

## Cách hoạt động

### 1. Kiểm tra trạng thái process
- Khi user gửi request đến endpoint `/process-video`, hệ thống sẽ kiểm tra xem user có process nào đang chạy không
- Nếu có process đang chạy và chưa quá 10 phút, request sẽ bị từ chối
- Nếu process cũ bị treo quá 10 phút, user được phép bắt đầu process mới

### 2. Cập nhật trạng thái
- Khi process bắt đầu: trạng thái = "processing"
- Khi process hoàn thành: trạng thái = "completed"
- Khi process lỗi: trạng thái = "failed"

### 3. Thông báo cho user
- Frontend hiển thị toast notification màu vàng khi bị chặn
- Modal chi tiết hiển thị thông tin process đang chạy
- Thời gian đã chạy và thời gian còn lại

## Cài đặt

### 1. Chạy migration database
```bash
# Cấp quyền thực thi cho script
chmod +x run_process_status_migration.sh

# Chạy migration
./run_process_status_migration.sh
```

### 2. Build lại backend
```bash
go build -o main main.go
```

### 3. Restart service
```bash
# Nếu dùng Docker
docker-compose restart

# Nếu chạy trực tiếp
./main
```

## Cấu trúc Database

### Bảng `user_process_status`
```sql
CREATE TABLE user_process_status (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    status ENUM('processing', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'processing',
    process_type ENUM('process', 'process-video', 'process-voice', 'process-background') NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    video_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

## API Endpoints

### Endpoint được bảo vệ
- `POST /process-video` - Xử lý video (đã áp dụng middleware)

### Response khi bị chặn (HTTP 409)
```json
{
  "error": "Bạn đang có một quá trình xử lý đang chạy. Vui lòng đợi quá trình hiện tại hoàn thành.",
  "details": {
    "process_id": 123,
    "process_type": "process-video",
    "started_at": "2024-01-01T10:00:00Z",
    "time_elapsed": "2m30s",
    "remaining_time": "7m30s"
  }
}
```

## Frontend Features

### 1. Toast Notification
- Hiển thị thông báo màu vàng khi bị chặn
- Thời gian hiển thị: 5 giây

### 2. Process Status Modal
- Hiển thị chi tiết process đang chạy
- Thông tin: loại xử lý, thời gian bắt đầu, thời gian đã chạy, thời gian còn lại
- Nút "Làm mới trang" để kiểm tra trạng thái mới

## Mở rộng

### Áp dụng cho endpoint khác
Để áp dụng tính năng này cho endpoint khác, thêm middleware vào router:

```go
// Trong router.go
protected.POST("/process", middleware.ProcessStatusMiddleware("process"), handler.ProcessHandler)
protected.POST("/process-voice", middleware.ProcessStatusMiddleware("process-voice"), handler.ProcessVoiceHandler)
```

### Tùy chỉnh thời gian timeout
Thay đổi thời gian timeout trong `service/process_status_service.go`:

```go
// Thay đổi từ 10 phút thành thời gian khác
if timeSinceStart > 15*time.Minute { // 15 phút
    // ...
}
```

## Troubleshooting

### 1. Process bị treo
- Hệ thống tự động đánh dấu process là "failed" sau 10 phút
- User có thể thử lại sau khi process bị treo

### 2. Database connection error
- Kiểm tra thông tin kết nối database trong config
- Đảm bảo bảng `user_process_status` đã được tạo

### 3. Frontend không hiển thị modal
- Kiểm tra console browser để xem lỗi JavaScript
- Đảm bảo component `ProcessStatusModal.vue` đã được import

## Monitoring

### Logs
- Backend log sẽ ghi lại các thao tác với process status
- Tìm kiếm log với keyword: "process status", "user process"

### Database queries
```sql
-- Xem tất cả process đang chạy
SELECT * FROM user_process_status WHERE status = 'processing';

-- Xem process của user cụ thể
SELECT * FROM user_process_status WHERE user_id = 123;

-- Xem process bị treo
SELECT * FROM user_process_status 
WHERE status = 'processing' 
AND started_at < DATE_SUB(NOW(), INTERVAL 10 MINUTE);
``` 