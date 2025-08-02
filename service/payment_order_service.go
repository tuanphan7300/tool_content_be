package service

import (
	"creator-tool-backend/config"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentOrderService quản lý đơn hàng thanh toán
type PaymentOrderService struct {
	qrService *QRService
}

// NewPaymentOrderService tạo instance mới
func NewPaymentOrderService() *PaymentOrderService {
	return &PaymentOrderService{
		qrService: NewQRService(),
	}
}

// CreateOrder tạo đơn hàng mới
func (s *PaymentOrderService) CreateOrder(userID uint, amountUSD float64) (*config.PaymentOrder, error) {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Tạo order code duy nhất
	orderCode := s.generateOrderCode()

	// Chuyển đổi USD sang VND (tỷ giá cố định 25,000 VND/USD)
	exchangeRate := decimal.NewFromFloat(25000)
	amountUSDDecimal := decimal.NewFromFloat(amountUSD)
	amountVND := amountUSDDecimal.Mul(exchangeRate)

	// Lấy tài khoản ngân hàng khả dụng
	bankAccount, err := s.getAvailableBankAccount(amountVND)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get bank account: %v", err)
	}

	// Tạo đơn hàng
	order := &config.PaymentOrder{
		UserID:        userID,
		OrderCode:     orderCode,
		AmountVND:     amountVND,
		AmountUSD:     amountUSDDecimal,
		ExchangeRate:  exchangeRate,
		BankAccount:   bankAccount.AccountNumber,
		BankName:      bankAccount.BankName,
		OrderStatus:   "pending",
		PaymentMethod: "qr_code",                        // Giữ nguyên qr_code để hiển thị QR
		ExpiresAt:     time.Now().Add(30 * time.Minute), // Hết hạn sau 30 phút
		CreatedAt:     time.Now(),
	}

	err = tx.Create(order).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Tạo QR code trong cùng transaction
	err = s.generateQRCodeInTransaction(tx, order)
	if err != nil {
		log.Printf("Failed to generate QR code for order %s: %v", orderCode, err)
		// Không rollback vì đơn hàng đã tạo thành công
	}

	// Log tạo đơn hàng
	s.createPaymentLog(order.ID, "order_created", "Đơn hàng được tạo thành công", nil)

	err = tx.Commit().Error
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return order, nil
}

// generateOrderCode tạo mã đơn hàng duy nhất
func (s *PaymentOrderService) generateOrderCode() string {
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(9999) + 1000
	return fmt.Sprintf("PAY%s%04d", time.Now().Format("20060102150405"), randomNum)
}

// getAvailableBankAccount lấy tài khoản ngân hàng khả dụng
func (s *PaymentOrderService) getAvailableBankAccount(amount decimal.Decimal) (*config.BankAccount, error) {
	var bankAccount config.BankAccount
	err := config.Db.Where("is_active = ? AND daily_limit >= ?", true, amount).First(&bankAccount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Tạo tài khoản mặc định nếu chưa có
			bankAccount = config.BankAccount{
				BankName:      "Vietcombank",
				AccountNumber: "1234567890",
				AccountName:   "NGUYEN VAN A",
				BankCode:      "VCB",
				IsActive:      true,
				DailyLimit:    decimal.NewFromInt(100000000),
				MonthlyLimit:  decimal.NewFromInt(1000000000),
				CreatedAt:     time.Now(),
			}
			err = config.Db.Create(&bankAccount).Error
			if err != nil {
				return nil, fmt.Errorf("failed to create default bank account: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get bank account: %v", err)
		}
	}

	return &bankAccount, nil
}

// generateQRCode tạo QR code cho đơn hàng (với retry logic)
func (s *PaymentOrderService) generateQRCode(order *config.PaymentOrder) error {
	// Lấy thông tin tài khoản ngân hàng
	var bankAccount config.BankAccount
	err := config.Db.Where("account_number = ?", order.BankAccount).First(&bankAccount).Error
	if err != nil {
		return fmt.Errorf("failed to get bank account info: %v", err)
	}

	// Tạo QR code theo chuẩn VietQR NAPAS247
	qrDataURL, err := s.qrService.GenerateVietQRCode(
		bankAccount.CardBin,
		order.BankAccount,
		order.AmountVND.String(),
		order.OrderCode,
	)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Lưu QR data (URL VietQR cũ để backup)
	qrString := fmt.Sprintf("https://api.vietqr.io/image/%s/%s/%s/%s",
		bankAccount.BankCode,
		order.BankAccount,
		order.AmountVND.String(),
		order.OrderCode,
	)

	// Lưu QR data
	order.QRCodeData = &qrString
	order.QRCodeURL = &qrDataURL // Sử dụng QR code tự generate

	// Retry logic cho lock timeout
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := config.Db.Model(order).Updates(map[string]interface{}{
			"qr_code_url":  order.QRCodeURL,
			"qr_code_data": order.QRCodeData,
		}).Error

		if err == nil {
			// Log tạo QR code
			s.createPaymentLog(order.ID, "qr_generated", "QR code được tạo thành công", nil)
			return nil
		}

		// Kiểm tra nếu là lock timeout error
		if i < maxRetries-1 {
			log.Printf("Lock timeout when updating QR code for order %s, retrying... (attempt %d/%d)",
				order.OrderCode, i+1, maxRetries)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		}
	}

	return fmt.Errorf("failed to update QR code after %d retries", maxRetries)
}

// generateQRCodeInTransaction tạo QR code trong cùng transaction
func (s *PaymentOrderService) generateQRCodeInTransaction(tx *gorm.DB, order *config.PaymentOrder) error {
	// Lấy thông tin tài khoản ngân hàng
	var bankAccount config.BankAccount
	err := config.Db.Where("account_number = ?", order.BankAccount).First(&bankAccount).Error
	if err != nil {
		return fmt.Errorf("failed to get bank account info: %v", err)
	}

	// Tạo QR code theo chuẩn VietQR NAPAS247
	qrDataURL, err := s.qrService.GenerateVietQRCode(
		bankAccount.CardBin,
		order.BankAccount,
		order.AmountVND.String(),
		order.OrderCode,
	)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Lưu QR data (URL VietQR cũ để backup)
	qrString := fmt.Sprintf("https://api.vietqr.io/image/%s/%s/%s/%s",
		bankAccount.BankCode,
		order.BankAccount,
		order.AmountVND.String(),
		order.OrderCode,
	)

	// Lưu QR data
	order.QRCodeData = &qrString
	order.QRCodeURL = &qrDataURL // Sử dụng QR code tự generate

	err = tx.Model(order).Updates(map[string]interface{}{
		"qr_code_url":  order.QRCodeURL,
		"qr_code_data": order.QRCodeData,
	}).Error
	if err != nil {
		return fmt.Errorf("failed to update QR code in transaction: %v", err)
	}

	// Log tạo QR code
	s.createPaymentLogInTransaction(tx, order.ID, "qr_generated", "QR code được tạo thành công", nil)

	return nil
}

// UpdateOrderStatus cập nhật trạng thái đơn hàng
func (s *PaymentOrderService) UpdateOrderStatus(orderID uint, status string, transactionID *string) error {
	// Retry logic cho lock timeout
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := s.updateOrderStatusWithRetry(orderID, status, transactionID)
		if err == nil {
			return nil
		}

		// Kiểm tra nếu là lock timeout error và còn retry
		if i < maxRetries-1 {
			log.Printf("Lock timeout when updating order status for order %d, retrying... (attempt %d/%d)",
				orderID, i+1, maxRetries)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
		} else {
			return fmt.Errorf("failed to update order status after %d retries: %v", maxRetries, err)
		}
	}

	return fmt.Errorf("failed to update order status after %d retries", maxRetries)
}

// updateOrderStatusWithRetry thực hiện cập nhật trạng thái với retry
func (s *PaymentOrderService) updateOrderStatusWithRetry(orderID uint, status string, transactionID *string) error {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var order config.PaymentOrder
	err := tx.Where("id = ?", orderID).First(&order).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get order: %v", err)
	}

	updates := map[string]interface{}{
		"order_status": status,
	}

	if status == "paid" {
		now := time.Now()
		updates["paid_at"] = &now
		if transactionID != nil {
			updates["transaction_id"] = transactionID
		}
	}

	err = tx.Model(&order).Updates(updates).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %v", err)
	}

	// Log cập nhật trạng thái
	logMessage := fmt.Sprintf("Trạng thái đơn hàng được cập nhật thành: %s", status)
	s.createPaymentLogInTransaction(tx, orderID, "payment_confirmed", logMessage, transactionID)

	return tx.Commit().Error
}

// GetOrderByCode lấy đơn hàng theo mã
func (s *PaymentOrderService) GetOrderByCode(orderCode string) (*config.PaymentOrder, error) {
	var order config.PaymentOrder
	err := config.Db.Where("order_code = ?", orderCode).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetUserOrders lấy danh sách đơn hàng của user
func (s *PaymentOrderService) GetUserOrders(userID uint, limit int) ([]config.PaymentOrder, error) {
	var orders []config.PaymentOrder
	err := config.Db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&orders).Error
	return orders, err
}

// CheckExpiredOrders kiểm tra và cập nhật đơn hàng hết hạn
func (s *PaymentOrderService) CheckExpiredOrders() error {
	var orders []config.PaymentOrder
	err := config.Db.Where("order_status = ? AND expires_at < ?", "pending", time.Now()).Find(&orders).Error
	if err != nil {
		return fmt.Errorf("failed to get expired orders: %v", err)
	}

	for _, order := range orders {
		err = s.UpdateOrderStatus(order.ID, "expired", nil)
		if err != nil {
			log.Printf("Failed to expire order %s: %v", order.OrderCode, err)
		}
	}

	return nil
}

// createPaymentLog tạo log thanh toán
func (s *PaymentOrderService) createPaymentLog(orderID uint, logType, message string, metadata *string) {
	paymentLog := config.PaymentLog{
		OrderID:   orderID,
		LogType:   logType,
		Message:   message,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	err := config.Db.Create(&paymentLog).Error
	if err != nil {
		log.Printf("Failed to create payment log: %v", err)
	}
}

// createPaymentLogInTransaction tạo log thanh toán trong transaction
func (s *PaymentOrderService) createPaymentLogInTransaction(tx *gorm.DB, orderID uint, logType, message string, metadata *string) error {
	paymentLog := config.PaymentLog{
		OrderID:   orderID,
		LogType:   logType,
		Message:   message,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	return tx.Create(&paymentLog).Error
}
