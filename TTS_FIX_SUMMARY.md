# TTS Filter Complex Error Fix Summary

## Vấn đề gốc

Lỗi xảy ra khi xử lý TTS với 15 segments:
```
[AVFilterGraph @ 0x600000448070] More input link labels specified for filter 'adelay' than it has inputs: 2 > 1
[AVFilterGraph @ 0x600000448070] Error linking filters
```

## Nguyên nhân

1. **Filter Complex quá phức tạp**: Code cũ cố gắng tạo một filter_complex phức tạp với nhiều adelay filters
2. **Cú pháp FFmpeg sai**: Filter complex được tạo ra có cú pháp không đúng cho multiple inputs
3. **Giới hạn FFmpeg**: FFmpeg có giới hạn về độ phức tạp của filter_complex

## Giải pháp đã áp dụng

### 1. Loại bỏ Filter Complex phức tạp
- Xóa bỏ approach tạo filter_complex phức tạp cho segments ≤20
- Thay thế bằng approach tạo delayed files riêng biệt

### 2. Sử dụng Individual Delayed Files
```go
// Tạo delayed file cho từng segment
for i, segmentFile := range segmentFiles {
    delayedFile := filepath.Join(tempDir, fmt.Sprintf("delayed_%d.wav", i))
    
    cmd := exec.Command("ffmpeg",
        "-i", segmentFile,
        "-af", fmt.Sprintf("adelay=%d|%d", int(entries[i].Start*1000), int(entries[i].Start*1000)),
        "-ar", "44100",
        "-ac", "2",
        "-acodec", "pcm_s16le",
        "-y",
        delayedFile)
}
```

### 3. Mix tất cả delayed files
```go
// Mix tất cả delayed files với base silence
args := []string{"-i", baseAudioFile}
for _, delayedFile := range delayedFiles {
    args = append(args, "-i", delayedFile)
}
args = append(args,
    "-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest[out]", len(delayedFiles)+1),
    "-map", "[out]",
    "-ar", "44100",
    "-ac", "2",
    "-acodec", "libmp3lame",
    "-b:a", "192k",
    "-y",
    outputPath)
```

## Lợi ích của fix

1. **Tránh giới hạn FFmpeg**: Không còn bị giới hạn bởi độ phức tạp của filter_complex
2. **Xử lý lỗi tốt hơn**: Mỗi segment được xử lý riêng biệt, lỗi không ảnh hưởng toàn bộ
3. **Code sạch hơn**: Logic đơn giản và dễ hiểu hơn
4. **Tương thích tốt**: Hoạt động với mọi số lượng segments

## Test Cases

### Test với 15 segments (như lỗi gốc)
- ✅ Không còn filter complex error
- ✅ Tạo được TTS audio thành công
- ✅ Timing chính xác

### Test với các số lượng segments khác
- ✅ ≤20 segments: Individual delayed files
- ✅ >20 segments: Concat approach  
- ✅ >50 segments: Progressive mixing
- ✅ >200 segments: Batch processing

## Files đã thay đổi

- `service/text_to_speech.go`: Sửa logic xử lý segments
- `test_tts_fix.sh`: Script test mới
- `TTS_FIX_SUMMARY.md`: Tài liệu này

## Cách test

1. Chạy script test:
```bash
./test_tts_fix.sh
```

2. Upload video với SRT có 15+ segments
3. Kiểm tra logs không còn filter complex error
4. Verify TTS audio được tạo thành công

## Kết luận

Fix này giải quyết hoàn toàn vấn đề filter complex error và cải thiện độ ổn định của hệ thống TTS. Code mới đơn giản hơn, dễ maintain hơn và hoạt động ổn định với mọi số lượng segments. 