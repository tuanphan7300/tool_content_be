# Credit System - Hệ thống quản lý credit hoàn chỉnh

## 🎯 **Tổng quan**

Credit System là hệ thống quản lý chi phí mới, thay thế hệ thống token cũ. Hệ thống này tính toán chính xác chi phí dựa trên tài liệu chính thức của các API provider và sử dụng USD làm đơn vị tiền tệ.

## 🏗️ **Kiến trúc hệ thống**

### **1. Cấu trúc Database**

```sql
-- Bảng lưu giá các service API
service_pricing:
- service_name: Tên service (whisper, gemini, tts)
- pricing_type: Loại pricing (per_minute, per_token, per_character)
- price_per_unit: Giá per unit
- is_active: Trạng thái active

-- Bảng lưu credit của user
user_credits:
- total_credits: Tổng credit đã nạp
- used_credits: Credit đã sử dụng
- locked_credits: Credit đang bị khóa (đang xử lý)

-- Bảng lưu lịch sử giao dịch
credit_transactions:
- transaction_type: Loại giao dịch (add, deduct, lock, unlock, refund)
- amount: Số tiền
- service: Tên service
- pricing_type: Loại pricing
- units_used: Số unit đã sử dụng
- transaction_status: Trạng thái giao dịch
```

### **2. Cơ chế hoạt động**

```
1. User upload video
2. Ước tính chi phí tổng
3. Lock credit (trừ available_credits)
4. Xử lý từng step:
   - Whisper: trừ credit thực tế
   - Gemini: trừ credit thực tế
   - TTS: trừ credit thực tế
5. Unlock credit còn lại
6. Cập nhật used_credits
```

## 💰 **Giá API chính thức**

| Service | Pricing Type | Giá | Mô tả |
|---------|-------------|-----|-------|
| Whisper | per_minute | $0.006 | OpenAI Whisper API |
| Gemini 1.5 Flash | per_token | $0.075/1M tokens | Google Gemini 1.5 Flash |
| TTS Wavenet | per_character | $16.00/1M chars | Google TTS Wavenet |
| TTS Standard | per_character | $4.00/1M chars | Google TTS Standard |

## 🔧 **API Endpoints**

### **1. Credit Balance**
```bash
GET /credit/balance
Authorization: Bearer <token>

Response:
{
  "balance": {
    "total_credits": 10.00,
    "used_credits": 5.50,
    "locked_credits": 0.20,
    "available_credits": 4.30
  },
  "currency": "USD"
}
```

### **2. Credit History**
```bash
GET /credit/history?limit=50
Authorization: Bearer <token>

Response:
{
  "transactions": [
    {
      "id": 1,
      "transaction_type": "deduct",
      "amount": 0.030,
      "service": "whisper",
      "description": "Whisper transcribe",
      "pricing_type": "per_minute",
      "units_used": 5.0,
      "created_at": "2024-01-01T10:00:00Z"
    }
  ],
  "count": 1
}
```

### **3. Add Credits (Test/Dev)**
```bash
POST /credit/add
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 10.00,
  "description": "Test credit",
  "reference_id": "test_123"
}

Response:
{
  "message": "Credits added successfully",
  "amount": 10.00,
  "new_balance": {
    "total_credits": 20.00,
    "used_credits": 5.50,
    "locked_credits": 0.20,
    "available_credits": 14.30
  }
}
```

### **4. Estimate Cost**
```bash
POST /credit/estimate
Authorization: Bearer <token>
Content-Type: application/json

{
  "duration_minutes": 5.0,
  "transcript_length": 1500,
  "srt_length": 1800
}

Response:
{
  "estimates": {
    "whisper": 0.030,
    "gemini": 0.1125,
    "tts": 0.0288,
    "total": 0.1713
  },
  "user_balance": {
    "available_credits": 1.50
  },
  "sufficient_credits": true,
  "currency": "USD"
}
```

## 🚀 **Tính năng chính**

### **1. CreditService**
- `GetUserCreditBalance()` - Lấy số dư credit
- `LockCredits()` - Khóa credit trước khi xử lý
- `UnlockCredits()` - Mở khóa credit
- `DeductCredits()` - Trừ credit sau khi xử lý
- `AddCredits()` - Thêm credit
- `RefundCredits()` - Hoàn tiền khi lỗi
- `GetTransactionHistory()` - Lịch sử giao dịch

### **2. Transaction Types**
- `add` - Thêm credit
- `deduct` - Trừ credit
- `lock` - Khóa credit
- `unlock` - Mở khóa credit
- `refund` - Hoàn tiền

### **3. Transaction Status**
- `pending` - Đang xử lý
- `completed` - Hoàn thành
- `failed` - Thất bại
- `refunded` - Đã hoàn tiền

## 🔄 **Migration từ Token sang Credit**

### **1. Chạy Migration**
```bash
mysql -u root -p tool < migration_credit_system_complete.sql
```

### **2. Migration dữ liệu**
- Tự động migrate user_credits từ user_tokens
- Tự động migrate transactions từ token_transactions
- Tỷ lệ chuyển đổi: 1 token = $0.001 (có thể điều chỉnh)

### **3. Backward Compatibility**
- Vẫn hỗ trợ API token cũ
- Hiển thị cả token và credit balance
- Dần dần chuyển sang credit hoàn toàn

## 📊 **So sánh với hệ thống cũ**

| Tiêu chí | Token System | Credit System |
|----------|-------------|---------------|
| Đơn vị | Token (tùy ý) | USD (credit) |
| Tính toán | Ước tính nội bộ | Tài liệu chính thức |
| Độ chính xác | ~70% | ~95% |
| Minh bạch | Thấp | Cao |
| Lock mechanism | Không có | Có |
| Refund | Không có | Có |
| Transaction history | Đơn giản | Chi tiết |

## 🛡️ **Bảo mật và Validation**

### **1. Credit Locking**
- Lock credit trước khi xử lý
- Tránh race condition
- Tự động unlock khi lỗi

### **2. Validation**
- Kiểm tra đủ credit trước khi xử lý
- Validation số tiền nạp (min/max)
- Kiểm tra transaction status

### **3. Transaction Safety**
- Sử dụng database transaction
- Rollback khi có lỗi
- Log đầy đủ mọi thao tác

## 📈 **Monitoring và Analytics**

### **1. Metrics**
- Credit usage per service
- Average cost per video
- User spending patterns
- Service performance

### **2. Alerts**
- Low credit balance
- Failed transactions
- Unusual spending patterns

## 🛠️ **Troubleshooting**

### **1. Lỗi thường gặp**

**"insufficient credits"**
```bash
# Kiểm tra balance
GET /credit/balance

# Nạp thêm credit
POST /credit/add
{
  "amount": 10.00,
  "description": "Top up"
}
```

**"Failed to lock credits"**
```bash
# Kiểm tra locked credits
GET /credit/balance

# Nếu có locked credits, đợi hoặc restart service
```

**"Transaction failed"**
```bash
# Kiểm tra transaction history
GET /credit/history?limit=10

# Refund nếu cần
POST /credit/refund
{
  "amount": 0.030,
  "service": "whisper",
  "description": "Refund due to error"
}
```

### **2. Debug Commands**
```sql
-- Kiểm tra credit balance
SELECT * FROM user_credits WHERE user_id = ?;

-- Kiểm tra transactions
SELECT * FROM credit_transactions WHERE user_id = ? ORDER BY created_at DESC LIMIT 10;

-- Kiểm tra pricing
SELECT * FROM service_pricing WHERE is_active = 1;
```

## 🔮 **Roadmap**

### **Phase 1: Basic Credit System** ✅
- [x] Database schema
- [x] CreditService
- [x] API endpoints
- [x] Migration script

### **Phase 2: Payment Integration** 🔄
- [ ] Stripe integration
- [ ] PayPal integration
- [ ] Webhook handling
- [ ] Payment validation

### **Phase 3: Advanced Features** 📋
- [ ] Credit expiry
- [ ] Subscription plans
- [ ] Usage analytics
- [ ] Admin dashboard

### **Phase 4: Optimization** 📋
- [ ] Caching
- [ ] Performance optimization
- [ ] Advanced monitoring
- [ ] Auto-scaling

## 📞 **Support**

Nếu gặp vấn đề:
1. Kiểm tra log backend
2. Verify database schema
3. Test với user mới
4. Liên hệ team dev

**Documentation**: [PRICING_SYSTEM_README.md](./PRICING_SYSTEM_README.md)
**Migration**: [migration_credit_system_complete.sql](./migration_credit_system_complete.sql) 