# Fix Lỗi Units Used - Out of Range Value

## 🐛 **Vấn đề**

Lỗi xảy ra khi insert vào bảng `credit_transactions`:
```
Error 1264 (22003): Out of range value for column 'units_used' at row 1
```

**Nguyên nhân:**
- Cột `units_used` được định nghĩa là `decimal(10,6)` 
- Giá trị 11803 vượt quá giới hạn 9999.999999
- Code đang sử dụng `len(srtContentBytes)` (số byte) thay vì số ký tự thực tế

## 🔧 **Giải pháp**

### **1. Sửa cấu trúc Database**
```sql
-- Thay đổi từ decimal(10,6) thành decimal(15,6)
ALTER TABLE credit_transactions 
MODIFY COLUMN units_used DECIMAL(15,6) DEFAULT 0.000000;
```

### **2. Sửa Code Logic**
**File:** `tool_content_be/handler/process.go`

**Trước:**
```go
// Sử dụng số byte thay vì số ký tự
if err := creditService.DeductCredits(userID, ttsCost, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(len(srtContentBytes))); err != nil {
```

**Sau:**
```go
// Tính số ký tự thực tế (không phải số byte)
characterCount := len([]rune(srtContent))
if err := creditService.DeductCredits(userID, ttsCost, "tts", "Google TTS", &captionHistory.ID, "per_character", float64(characterCount)); err != nil {
```

### **3. Cập nhật Model**
**File:** `tool_content_be/config/database.go`

```go
// Thay đổi từ decimal(10,6) thành decimal(15,6)
UnitsUsed float64 `json:"units_used" gorm:"type:decimal(15,6);default:0.00"`
```

## 📊 **So sánh Giới hạn**

| Cấu trúc cũ | Cấu trúc mới |
|-------------|-------------|
| `decimal(10,6)` | `decimal(15,6)` |
| Tối đa: 9,999.999999 | Tối đa: 999,999,999.999999 |
| 4 chữ số nguyên | 9 chữ số nguyên |

## 🧪 **Test**

Script test: `test_units_used_fix.sql`
```sql
-- Test với giá trị 11803
INSERT INTO credit_transactions (..., units_used, ...) VALUES (..., 11803, ...);
-- Kết quả: 11803.000000 ✅
```

## 📁 **Files đã thay đổi**

1. `tool_content_be/config/database.go` - Cập nhật model
2. `tool_content_be/handler/process.go` - Sửa logic tính toán
3. `tool_content_be/migration_fix_units_used.sql` - Migration database
4. `tool_content_be/db/Dump20250708/tool_credit_transactions.sql` - Cập nhật dump

## ✅ **Kết quả**

- ✅ Lỗi "Out of range value" đã được fix
- ✅ Hỗ trợ số ký tự lớn (lên đến 999,999,999)
- ✅ Tính toán chính xác số ký tự thay vì số byte
- ✅ Backward compatibility được duy trì 