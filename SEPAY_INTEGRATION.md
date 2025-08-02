# Sepay Integration Guide

## Tổng quan

Hệ thống đã được tích hợp với Sepay để nhận thông báo thanh toán tự động từ ngân hàng. 
User vẫn sử dụng QR code để thanh toán, Sepay chỉ đóng vai trò gateway để nhận callback từ ngân hàng.

## API Endpoints

### Webhook URL
```
POST /api/v1/webhook/sepay
```

### Request Body Format
```json
{
  "id": 92704,                              // ID giao dịch trên SePay
  "gateway": "Vietcombank",                 // Brand name của ngân hàng
  "transactionDate": "2024-07-25 14:02:37", // Thời gian xảy ra giao dịch phía ngân hàng
  "accountNumber": "0123499999",            // Số tài khoản ngân hàng
  "code": null,                             // Mã code thanh toán (sepay tự nhận diện dựa vào cấu hình)
  "content": "chuyen tien mua iphone",      // Nội dung chuyển khoản
  "transferType": "in",                     // Loại giao dịch. in là tiền vào, out là tiền ra
  "transferAmount": 2277000,                // Số tiền giao dịch
  "accumulated": 19077000,                  // Số dư tài khoản (lũy kế)
  "subAccount": null,                       // Tài khoản ngân hàng phụ
  "referenceCode": "MBVCB.3278907687",      // Mã tham chiếu
  "description": ""                         // Toàn bộ nội dung tin notify ngân hàng
}
```

### Response Format
```json
{
  "message": "Payment processed successfully",
  "order_code": "PAY202501011200001234",
  "status": "paid"
}
```

## Cấu hình

### 1. Secret Key
Thêm secret key vào file config:
```go
// config/config.go
type InfaConfig struct {
    // ... existing fields
    SepaySecretKey string `env:"SEPAY_SECRET_KEY"`
}
```

### 2. Environment Variables
```bash
SEPAY_SECRET_KEY=your_secret_key_here
```

## Luồng xử lý

1. **User tạo đơn hàng** → `POST /payment/create-order`
2. **Frontend hiển thị QR code** → User quét QR để thanh toán
3. **User thanh toán** → Qua app ngân hàng
4. **Ngân hàng thông báo** → Sepay (gateway)
5. **Sepay callback** → `POST /api/v1/webhook/sepay`
6. **Backend xử lý**:
   - Parse webhook data từ Sepay
   - Extract order code từ `code` field hoặc `content`
   - Kiểm tra `transferType` phải là "in"
   - Validate order và amount
   - Cập nhật trạng thái thành "paid"
   - Cộng credit cho user

## Security

### Signature Verification
- Sử dụng HMAC-SHA256
- Format: `key1=value1&key2=value2&...`
- Sort keys alphabetically
- Exclude signature field

### Validation
- Order code phải tồn tại (từ `code` field hoặc extract từ `content`)
- Order status phải là "pending"
- Amount phải khớp với đơn hàng (`transferAmount`)
- `transferType` phải là "in" (tiền vào)
- Gateway phải hợp lệ

## Testing

### Test Webhook
```bash
curl -X POST http://localhost:8888/api/v1/webhook/sepay \
  -H "Content-Type: application/json" \
  -d '{
    "id": 92704,
    "gateway": "Vietcombank",
    "transactionDate": "2024-07-25 14:02:37",
    "accountNumber": "0123499999",
    "code": "PAY202501011200001234",
    "content": "PAY202501011200001234 chuyen tien mua credit",
    "transferType": "in",
    "transferAmount": 2500000,
    "accumulated": 19077000,
    "subAccount": null,
    "referenceCode": "MBVCB.3278907687",
    "description": "BankAPINotify MBVCB.3278907687"
  }'
```

## Monitoring

### Logs
- Payment logs được lưu trong bảng `payment_logs`
- Credit transactions được lưu trong bảng `credit_transactions`
- **Sepay webhook logs** được lưu trong bảng `sepay_webhook_logs`

### Sepay Webhook Logs
Bảng `sepay_webhook_logs` lưu tất cả webhook từ Sepay với các thông tin:
- **Raw payload**: Toàn bộ JSON gốc từ Sepay
- **Headers**: Headers của request
- **IP Address**: IP của Sepay
- **Processing Status**: received, validated, processed, failed, ignored
- **Processing Time**: Thời gian xử lý tính bằng milliseconds
- **Error Message**: Thông báo lỗi nếu có

### Admin API
```
GET /admin/sepay/webhook-logs?page=1&limit=50&status=processed&order_code=PAY123
```

### Status Tracking
- Frontend polling mỗi 10 giây
- Backend cron job kiểm tra đơn hàng hết hạn mỗi 1 phút

## Troubleshooting

### Common Issues
1. **Invalid signature**: Kiểm tra secret key và format signature
2. **Order not found**: Kiểm tra order code
3. **Amount mismatch**: Kiểm tra số tiền
4. **Order already processed**: Kiểm tra trạng thái đơn hàng

### Debug Mode
Thêm log để debug:
```go
log.Printf("Webhook received: %+v", webhookData)
log.Printf("Order found: %+v", order)
log.Printf("Signature verification: %v", isValid)
``` 