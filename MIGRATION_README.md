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