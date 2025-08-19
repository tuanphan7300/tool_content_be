hân tích nguyên nhân thời gian xử lý tăng hơn tuyến tính: TTS fallback tuần tự và amix nhiều input; burn subtitle re-encode chậm. Redis OK nhưng cần bật optimized TTS, batching mix, preset nhanh.
Đã triển khai:
Optimized TTS: giảm mặc định concurrency xuống 6; thêm batch mixing khi số segment lớn; giữ amix đơn khi ít segment.
Burn subtitle: scale 720p, libx264 preset veryfast, CRF 23, yuv420p, +faststart.
Thêm chọn background_quality trên FE (tab 2) và truyền qua API.
Backend nhận background_quality:
quality (mặc định): giữ Demucs như cũ.
ducking: bỏ tách nền, dùng audio gốc và trộn bằng sidechaincompress để TTS nổi, log đầy đủ.
Đã thêm log xác nhận mode ducking và đường dẫn file sau merge/burn. Hiện bạn thấy video chỉ có sub đã dịch, không nghe TTS; khuyến nghị kiểm tra log “Optimized TTS completed…” và thử tăng tts_volume, giảm background_volume, hoặc thử lại với quality để khoanh vùng ducking filter.