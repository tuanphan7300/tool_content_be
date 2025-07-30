# GPT Translation Service

## 📋 Tổng quan

Hệ thống đã được nâng cấp để hỗ trợ **GPT translation** thay thế cho **Gemini** trong việc dịch phụ đề SRT.

## 🚀 Tính năng mới

### ✅ Đã triển khai:
- **GPT Translation Service**: Sử dụng GPT-4o-mini để dịch SRT
- **Dynamic Service Selection**: Tự động chọn Gemini hoặc GPT dựa trên cấu hình
- **Cost Calculation**: Tính toán chi phí chính xác cho từng service
- **Credit Management**: Quản lý credit cho cả Gemini và GPT

### 🎯 Lợi ích:
- **Chất lượng dịch tốt hơn**: GPT-4o-mini có khả năng hiểu ngữ cảnh tốt hơn
- **Phù hợp với hoạt hình/truyện**: Dịch tự nhiên, giữ nguyên tên nhân vật
- **Chi phí hợp lý**: $0.00015/token (rẻ hơn GPT-4o gấp 33 lần)

## 📊 So sánh chi phí

| Service | Model | Giá | Ước tính cho video 10 phút |
|---------|-------|-----|---------------------------|
| **Gemini** | 1.5 Flash | $0.075/1M tokens | ~$0.0075 |
| **GPT** | 4o-mini | $0.15/1M tokens | ~$0.015 |
| **GPT** | 4o | $5/1M tokens | ~$0.5 |

## 🔧 Cài đặt

### 1. Chạy script setup:
```bash
cd tool_content_be
./scripts/setup-gpt-translation.sh
```

### 2. Kiểm tra cài đặt:
```sql
SELECT * FROM service_config WHERE service_type = 'srt_translation';
SELECT * FROM service_pricings WHERE service_name = 'gpt_translation';
```

## ⚙️ Cấu hình

### Chuyển đổi giữa Gemini và GPT:

#### Sử dụng GPT:
```sql
UPDATE service_config SET is_active = 0 WHERE service_name = 'gemini_translation';
UPDATE service_config SET is_active = 1 WHERE service_name = 'gpt_translation';
```

#### Sử dụng Gemini (mặc định):
```sql
UPDATE service_config SET is_active = 0 WHERE service_name = 'gpt_translation';
UPDATE service_config SET is_active = 1 WHERE service_name = 'gemini_translation';
```

### Cấu hình model:
```sql
-- Thay đổi model GPT
UPDATE service_config 
SET config_json = '{"model": "gpt-4o"}' 
WHERE service_name = 'gpt_translation';
```

## 🔄 API Endpoints

### Process Video (Tab 2, 4):
- **Endpoint**: `POST /process-video`
- **Logic**: Tự động chọn service dựa trên `service_config`
- **Response**: SRT đã dịch bằng GPT hoặc Gemini

### Create Subtitle (Tab 4):
- **Endpoint**: `POST /create-subtitle`
- **Logic**: Tương tự process-video
- **Response**: SRT gốc + SRT đã dịch (nếu song ngữ)

## 📝 Code Changes

### Files đã sửa:
1. **`service/gpt.go`**: Thêm `TranslateSRTWithGPT()`
2. **`service/srt_translator.go`**: Thêm wrapper function
3. **`handler/process.go`**: Cập nhật logic chọn service
4. **`service/pricing_service.go`**: Đã có sẵn `CalculateGPTCost()`

### Logic chính:
```go
// Chọn service dựa trên cấu hình
if strings.Contains(serviceName, "gpt") {
    translatedContent, err = service.TranslateSRTFileWithGPT(srtPath, apiKey, modelName)
} else {
    translatedContent, err = service.TranslateSRTFileWithModelAndLanguage(srtPath, geminiKey, modelName, targetLanguage)
}
```

## 🧪 Testing

### Test GPT Translation:
1. Chuyển sang GPT: `UPDATE service_config SET is_active = 1 WHERE service_name = 'gpt_translation';`
2. Upload video và chọn "Dịch & Lồng tiếng tự động"
3. Kiểm tra log: `service.TranslateSRTFileWithGPT` được gọi
4. Kiểm tra credit: `GPT dịch SRT` được trừ

### Test Gemini Translation:
1. Chuyển về Gemini: `UPDATE service_config SET is_active = 1 WHERE service_name = 'gemini_translation';`
2. Upload video và chọn "Dịch & Lồng tiếng tự động"
3. Kiểm tra log: `service.TranslateSRTFileWithModelAndLanguage` được gọi
4. Kiểm tra credit: `Gemini dịch SRT` được trừ

## 🚨 Lưu ý

1. **API Key**: Đảm bảo `OPENAI_API_KEY` đã được cấu hình trong `.env`
2. **Credit**: GPT đắt hơn Gemini ~2x, cần đủ credit
3. **Rate Limit**: GPT có rate limit thấp hơn Gemini
4. **Fallback**: Nếu GPT lỗi, có thể chuyển về Gemini

## 📈 Monitoring

### Logs cần theo dõi:
- `service.TranslateSRTFileWithGPT` - GPT translation calls
- `service.TranslateSRTFileWithModelAndLanguage` - Gemini translation calls
- Credit deduction: `GPT dịch SRT` vs `Gemini dịch SRT`

### Metrics:
- Translation success rate
- Average cost per video
- User preference (Gemini vs GPT) 