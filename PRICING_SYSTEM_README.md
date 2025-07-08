# Hệ thống Pricing Chính xác

## 🎯 **Tổng quan**

Hệ thống pricing mới được thiết kế để tính toán chi phí chính xác dựa trên tài liệu chính thức của các API provider, thay vì sử dụng ước tính nội bộ.

## 📊 **Cấu trúc Database**

### **1. Bảng `service_pricing`**
Lưu giá các service API theo tài liệu chính thức:

```sql
CREATE TABLE service_pricing (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    service_name VARCHAR(50) NOT NULL UNIQUE,
    pricing_type ENUM('per_minute', 'per_token', 'per_character') NOT NULL,
    price_per_unit DECIMAL(10,6) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### **2. Bảng `user_credits`**
Lưu credit của user (USD):

```sql
CREATE TABLE user_credits (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    total_credits DECIMAL(10,2) DEFAULT 0.00,
    used_credits DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_user_credits (user_id)
);
```

### **3. Cập nhật bảng `token_transactions`**
Thêm các trường mới để lưu thông tin chi tiết:

```sql
ALTER TABLE token_transactions 
ADD COLUMN credit_amount DECIMAL(10,2) DEFAULT 0.00,
ADD COLUMN pricing_type VARCHAR(20),
ADD COLUMN units_used DECIMAL(10,6) DEFAULT 0.00;
```

## 💰 **Giá API Chính thức (2024)**

| Service | Pricing Type | Giá | Mô tả |
|---------|-------------|-----|-------|
| Whisper | per_minute | $0.006 | OpenAI Whisper API |
| Gemini 1.5 Flash | per_token | $0.075/1M tokens | Google Gemini 1.5 Flash |
| TTS Standard | per_character | $4.00/1M chars | Google TTS Standard |
| TTS Wavenet | per_character | $16.00/1M chars | Google TTS Wavenet |
| GPT-3.5 Turbo | per_token | $0.002/1K tokens | OpenAI GPT-3.5 Turbo |

## 🔧 **Cách sử dụng**

### **1. Chạy Migration**
```bash
mysql -u root -p tool < migration_add_service_pricing.sql
```

### **2. Ước tính chi phí**
```bash
curl -X POST http://localhost:8080/estimate-cost \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "duration_minutes": 5.0,
    "transcript_length": 1500,
    "srt_length": 1800
  }'
```

**Response:**
```json
{
  "estimates": {
    "whisper": 0.030,
    "gemini": 0.1125,
    "tts": 0.0288,
    "total": 0.1713
  },
  "user_credit_balance": 1.50,
  "sufficient_credits": true,
  "currency": "USD"
}
```

### **3. Kiểm tra số dư**
```bash
curl -X GET http://localhost:8080/token/balance \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Response:**
```json
{
  "total_tokens": 1000,
  "used_tokens": 200,
  "available_tokens": 800,
  "credit_balance": 1.50,
  "currency": "USD"
}
```

## 🚀 **Tính năng mới**

### **1. PricingService**
- `CalculateWhisperCost(durationMinutes)` - Tính chi phí Whisper theo phút
- `CalculateGeminiCost(text)` - Tính chi phí Gemini theo token
- `CalculateTTSCost(text, useWavenet)` - Tính chi phí TTS theo ký tự
- `DeductUserCredits()` - Trừ credit với transaction
- `AddUserCredits()` - Thêm credit
- `GetUserCreditBalance()` - Lấy số dư

### **2. Ước tính thông minh**
- Tự động ước tính transcript length nếu không được cung cấp
- Tính toán chính xác dựa trên thời gian audio thực tế
- Hỗ trợ cả TTS Standard và Wavenet

### **3. Backward Compatibility**
- Vẫn hỗ trợ hệ thống token cũ
- Hiển thị cả token và credit balance
- Migration mượt mà không ảnh hưởng user hiện tại

## 📈 **So sánh với hệ thống cũ**

| Tiêu chí | Hệ thống cũ | Hệ thống mới |
|----------|-------------|--------------|
| Tính toán | Ước tính nội bộ | Tài liệu chính thức |
| Đơn vị | Token (tùy ý) | USD (credit) |
| Độ chính xác | ~70% | ~95% |
| Pricing | Cố định | Có thể cập nhật |
| Transparency | Thấp | Cao |

## 🔄 **Migration từ hệ thống cũ**

1. **Chạy migration script**
2. **Cập nhật code backend**
3. **Test với user mới**
4. **Migrate user cũ (tùy chọn)**

### **Migrate user cũ:**
```sql
-- Chuyển đổi token sang credit (tỷ lệ 1:1)
INSERT INTO user_credits (user_id, total_credits, used_credits)
SELECT user_id, total_tokens, used_tokens 
FROM user_tokens 
WHERE user_id NOT IN (SELECT user_id FROM user_credits);
```

## 🛠 **Troubleshooting**

### **Lỗi thường gặp:**

1. **"service pricing not found"**
   - Kiểm tra bảng `service_pricing` có dữ liệu
   - Chạy lại migration script

2. **"insufficient credits"**
   - User không đủ credit
   - Nạp thêm credit hoặc kiểm tra balance

3. **"Failed to calculate cost"**
   - Kiểm tra log để xem lỗi chi tiết
   - Verify pricing data trong database

## 📞 **Support**

Nếu gặp vấn đề, vui lòng:
1. Kiểm tra log backend
2. Verify database schema
3. Test với user mới
4. Liên hệ team dev 