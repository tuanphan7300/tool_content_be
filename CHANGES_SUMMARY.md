# Tóm tắt thay đổi: Thêm trường video_file_name_origin

## Vấn đề
API `/process-video` đang lưu tên file với timestamp (ví dụ: `1234567890_video.mp4`) vào database, nhưng người dùng muốn thấy tên file gốc (ví dụ: `video.mp4`) trong lịch sử.

## Giải pháp
Thêm trường `video_file_name_origin` vào database để lưu tên file gốc, trong khi `video_filename` vẫn lưu tên file đã được xử lý.

## Files đã thay đổi

### 1. Backend (Go)

#### `config/database.go`
- Thêm trường `VideoFilenameOrigin string` vào struct `CaptionHistory`

#### `handler/process.go`
- **ProcessVideoHandler**: Lưu `file.Filename` vào `VideoFilenameOrigin`
- **ProcessHandler**: Lưu `file.Filename` vào `VideoFilenameOrigin`

#### `handler/history.go`
- **SaveHistory**: Lưu `request.VideoFilename` vào `VideoFilenameOrigin`
- **GetHistoryHandler**: Trả về `video_file_name_origin` trong response

#### `handler/text_to_speech.go`
- Lưu `"text_to_speech.mp3"` vào `VideoFilenameOrigin`

### 2. Frontend (Vue.js)

#### `components/HistoryList.vue`
- Hiển thị `history.video_file_name_origin` thay vì `history.video_filename`
- Fallback về `history.video_filename` nếu `video_file_name_origin` không có

### 3. Database Migration

#### `migration_add_video_filename_origin.sql`
- Thêm cột `video_file_name_origin VARCHAR(255)` vào bảng `caption_histories`
- Cập nhật dữ liệu cũ để đảm bảo tương thích ngược

#### `run_migration.sh`
- Script tự động để chạy migration

#### `test_migration.sql`
- Script test để kiểm tra migration đã thành công

## Cách sử dụng

### 1. Chạy migration
```bash
cd tool_content_be
./run_migration.sh
```

### 2. Kiểm tra migration
```bash
mysql -h localhost -P 3306 -u root -proot tool < test_migration.sql
```

### 3. Test API
- Upload video qua API `/process-video`
- Kiểm tra API `/history` trả về `video_file_name_origin`
- Kiểm tra frontend hiển thị tên file gốc

## Kết quả mong đợi

### Trước khi thay đổi
```json
{
  "video_filename": "1234567890_my_video.mp4"
}
```

### Sau khi thay đổi
```json
{
  "video_filename": "1234567890_my_video.mp4",
  "video_file_name_origin": "my_video.mp4"
}
```

## Lưu ý
- Migration tương thích ngược với dữ liệu cũ
- Dữ liệu cũ sẽ có `video_file_name_origin` = `video_filename`
- Dữ liệu mới sẽ có `video_file_name_origin` = tên file gốc
- Frontend sẽ hiển thị tên file gốc cho người dùng 