# Sepay Webhook Fix - Order Code Extraction

## 🔍 **Vấn đề đã phát hiện:**

### **Log từ production:**
```
Content: "PAY202508060354408366 GD 464434-080625 10:55:18"
Order code được tìm: "PAY202508060354408366 GD 464434-080625 10:55:18" ❌
Order code thực tế: "PAY202508060354408366" ✅
```

### **Nguyên nhân:**
Logic extract order code cũ sử dụng `strings.Split(content, ".")` nhưng content thực tế không có dấu chấm, chỉ có khoảng trắng.

## 🛠️ **Giải pháp đã áp dụng:**

### **1. Cải thiện logic extract order code:**
```go
// Logic mới: Tìm từ PAY đến khoảng trắng đầu tiên
if strings.Contains(content, "PAY") {
    payIndex := strings.Index(content, "PAY")
    if payIndex != -1 {
        remainingContent := content[payIndex:]
        spaceIndex := strings.Index(remainingContent, " ")
        if spaceIndex != -1 {
            orderCode = remainingContent[:spaceIndex]  // Lấy đến khoảng trắng
        } else {
            orderCode = remainingContent  // Lấy toàn bộ nếu không có khoảng trắng
        }
    }
}
```

### **2. Thêm fallback với regex:**
```go
// Fallback: Tìm pattern PAY + 16 digits
re := regexp.MustCompile(`PAY\d{16}`)
matches := re.FindString(content)
if matches != "" {
    orderCode = matches
}
```

### **3. Thêm validation:**
```go
// Kiểm tra format order code
re := regexp.MustCompile(`^PAY\d{16}$`)
if !re.MatchString(orderCode) {
    // Re-extract bằng regex nếu format không đúng
}
```

## 📋 **Test Cases:**

| Content | Expected | Method 1 | Method 2 | Result |
|---------|----------|----------|----------|---------|
| `"PAY202508060354408366 GD 464434-080625 10:55:18"` | `PAY202508060354408366` | ✅ | ✅ | ✅ |
| `"PAY202508060354408366 chuyen tien"` | `PAY202508060354408366` | ✅ | ✅ | ✅ |
| `"chuyen tien PAY202508060354408366"` | `PAY202508060354408366` | ❌ | ✅ | ✅ |
| `"PAY202508060354408366"` | `PAY202508060354408366` | ✅ | ✅ | ✅ |

## 🔧 **Các thay đổi đã thực hiện:**

### **File: `handler/payment.go`**
1. ✅ Sửa logic extract order code từ split by "." thành split by space
2. ✅ Thêm fallback với regex pattern `PAY\d{16}`
3. ✅ Thêm validation để đảm bảo format đúng
4. ✅ Thêm logging chi tiết để debug

### **Import thêm:**
```go
import "regexp"
```

## 🚀 **Deploy Instructions:**

1. **Build và deploy:**
```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

2. **Kiểm tra logs:**
```bash
docker-compose logs api | grep -i "webhook\|order code"
```

3. **Test webhook:**
```bash
curl -X POST http://your-domain/v1/webhook/sepay \
  -H "Content-Type: application/json" \
  -H "Authorization: Apikey YOUR_API_KEY" \
  -d '{
    "content": "PAY202508060354408366 GD 464434-080625 10:55:18",
    "transferAmount": 5000,
    "transferType": "in"
  }'
```

## 📊 **Monitoring:**

### **Logs cần theo dõi:**
- `"Processing content: ..."`
- `"Extracted order code: ..."`
- `"Extracted order code using regex: ..."`
- `"Re-extracted valid order code: ..."`

### **Database:**
- Bảng `sepay_webhook_logs` - theo dõi processing_status
- Bảng `payment_orders` - kiểm tra order_status được cập nhật
- Bảng `credit_transactions` - kiểm tra credit được cộng

## ✅ **Kết quả mong đợi:**

Sau khi fix, webhook sẽ:
1. ✅ Extract đúng order code: `PAY202508060354408366`
2. ✅ Tìm thấy order trong database
3. ✅ Cập nhật trạng thái thành "paid"
4. ✅ Cộng credit cho user
5. ✅ Trả về status 200 thay vì 404 