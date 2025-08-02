package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"creator-tool-backend/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// CreatePaymentOrder tạo đơn hàng thanh toán mới
func CreatePaymentOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		AmountUSD     float64 `json:"amount_usd"`
		AmountVND     float64 `json:"amount_vnd"`
		PaymentMethod string  `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Xử lý amount - hỗ trợ cả USD và VND
	var amountUSD float64
	if req.AmountVND > 0 {
		// Nếu có amount_vnd, chuyển đổi sang USD
		amountUSD = req.AmountVND / 25000
	} else if req.AmountUSD > 0 {
		// Nếu có amount_usd, sử dụng trực tiếp
		amountUSD = req.AmountUSD
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	// Validation
	if amountUSD <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	if amountUSD > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount cannot exceed $1000"})
		return
	}

	// Set default payment method
	if req.PaymentMethod == "" {
		req.PaymentMethod = "qr_code"
	}

	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.CreateOrder(userID, amountUSD)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment order", "warning": "Không thể tạo đơn hàng thanh toán. Vui lòng thử lại hoặc liên hệ hỗ trợ!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": gin.H{
			"id":           order.ID,
			"order_code":   order.OrderCode,
			"amount_vnd":   order.AmountVND.String(),
			"amount_usd":   order.AmountUSD.String(),
			"qr_code_url":  order.QRCodeURL,
			"expires_at":   order.ExpiresAt,
			"bank_account": order.BankAccount,
			"bank_name":    order.BankName,
			"order_status": order.OrderStatus,
			"created_at":   order.CreatedAt,
		},
		"message": "Đơn hàng thanh toán được tạo thành công",
	})
}

// GetPaymentOrder lấy thông tin đơn hàng theo mã
func GetPaymentOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderCode := c.Param("order_code")
	if orderCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order code is required"})
		return
	}

	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(orderCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Kiểm tra quyền truy cập
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": gin.H{
			"id":           order.ID,
			"order_code":   order.OrderCode,
			"amount_vnd":   order.AmountVND.String(),
			"amount_usd":   order.AmountUSD.String(),
			"qr_code_url":  order.QRCodeURL,
			"expires_at":   order.ExpiresAt,
			"bank_account": order.BankAccount,
			"bank_name":    order.BankName,
			"order_status": order.OrderStatus,
			"paid_at":      order.PaidAt,
			"created_at":   order.CreatedAt,
		},
	})
}

// GetUserPaymentOrders lấy danh sách đơn hàng của user
func GetUserPaymentOrders(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Lấy limit từ query parameter
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// Lấy status filter
	status := c.Query("status")

	paymentService := service.NewPaymentOrderService()
	orders, err := paymentService.GetUserOrders(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment orders", "warning": "Không thể lấy danh sách đơn hàng. Vui lòng thử lại hoặc liên hệ hỗ trợ!"})
		return
	}

	// Filter theo status nếu có
	var filteredOrders []gin.H
	for _, order := range orders {
		if status == "" || order.OrderStatus == status {
			filteredOrders = append(filteredOrders, gin.H{
				"id":           order.ID,
				"order_code":   order.OrderCode,
				"amount_usd":   order.AmountUSD.String(),
				"order_status": order.OrderStatus,
				"paid_at":      order.PaidAt,
				"created_at":   order.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": filteredOrders,
		"count":  len(filteredOrders),
	})
}

// CancelPaymentOrder hủy đơn hàng thanh toán
func CancelPaymentOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderCode := c.Param("order_code")
	if orderCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order code is required"})
		return
	}

	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(orderCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Kiểm tra quyền truy cập
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Kiểm tra trạng thái đơn hàng
	if order.OrderStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled"})
		return
	}

	// Cập nhật trạng thái thành cancelled
	err = paymentService.UpdateOrderStatus(order.ID, "cancelled", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order", "warning": "Không thể hủy đơn hàng. Vui lòng thử lại hoặc liên hệ hỗ trợ!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Order cancelled successfully",
		"order_status": "cancelled",
	})
}

// GetPaymentOrderStatus lấy trạng thái đơn hàng (dùng cho polling)
func GetPaymentOrderStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderCode := c.Param("order_code")
	if orderCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order code is required"})
		return
	}

	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(orderCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Kiểm tra quyền truy cập
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_code":   order.OrderCode,
		"order_status": order.OrderStatus,
		"paid_at":      order.PaidAt,
		"expires_at":   order.ExpiresAt,
	})
}

// SepayWebhookHandler xử lý webhook từ Sepay
func SepayWebhookHandler(c *gin.Context) {
	startTime := time.Now()

	// Lưu thông tin request
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Đọc raw body để log
	rawBody, err := c.GetRawData()
	if err != nil {
		log.Printf("Failed to read raw body: %v", err)
		rawBody = []byte("{}")
	}

	// Parse headers
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Convert headers to JSON
	headersJSON, _ := json.Marshal(headers)

	// Tạo log entry ban đầu
	logEntry := config.SepayWebhookLog{
		RawPayload:       datatypes.JSON(rawBody),
		Headers:          datatypes.JSON(headersJSON),
		IPAddress:        &ipAddress,
		UserAgent:        &userAgent,
		ProcessingStatus: "received",
		CreatedAt:        time.Now(),
	}
	log.Println("logEntry sepay webhook", logEntry)
	// Lưu log entry ban đầu
	if err := config.Db.Create(&logEntry).Error; err != nil {
		log.Printf("Failed to create initial log entry: %v", err)
	}

	var webhookData struct {
		OrderCode     string  `json:"order_code"`
		Amount        float64 `json:"amount"`
		Status        string  `json:"status"`
		TransactionID string  `json:"transaction_id"`
		Signature     string  `json:"signature"`
		Timestamp     int64   `json:"timestamp"`
	}

	if err := c.ShouldBindJSON(&webhookData); err != nil {
		// Cập nhật log với lỗi parse JSON
		errorMsg := "Invalid webhook data: " + err.Error()
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	// Cập nhật log với thông tin parsed
	config.Db.Model(&logEntry).Updates(map[string]interface{}{
		"order_code":        webhookData.OrderCode,
		"amount":            webhookData.Amount,
		"status":            webhookData.Status,
		"transaction_id":    webhookData.TransactionID,
		"signature":         webhookData.Signature,
		"timestamp":         webhookData.Timestamp,
		"processing_status": "validated",
	})

	// Verify signature từ Sepay (cần thêm secret key vào config)
	// TODO: Lấy secret key từ config
	secretKey := "your_sepay_secret_key" // Thay thế bằng secret key thực từ config

	// Chuyển đổi webhookData thành map để verify
	dataMap := map[string]interface{}{
		"order_code":     webhookData.OrderCode,
		"amount":         webhookData.Amount,
		"status":         webhookData.Status,
		"transaction_id": webhookData.TransactionID,
		"timestamp":      webhookData.Timestamp,
	}

	if !util.VerifySepaySignature(dataMap, webhookData.Signature, secretKey) {
		// Cập nhật log với lỗi signature
		errorMsg := "Invalid signature"
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Kiểm tra trạng thái thanh toán
	if webhookData.Status != "success" {
		// Cập nhật log với trạng thái ignored
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "ignored",
			"error_message":      "Payment status is not success: " + webhookData.Status,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusOK, gin.H{"message": "Payment not successful"})
		return
	}

	// Tìm đơn hàng theo order code
	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(webhookData.OrderCode)
	if err != nil {
		// Cập nhật log với lỗi order not found
		errorMsg := "Order not found: " + webhookData.OrderCode
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Kiểm tra trạng thái đơn hàng
	if order.OrderStatus != "pending" {
		// Cập nhật log với trạng thái ignored
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "ignored",
			"error_message":      "Order already processed: " + order.OrderStatus,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusOK, gin.H{"message": "Order already processed"})
		return
	}

	// Kiểm tra số tiền
	expectedAmount, _ := order.AmountVND.Float64()
	if webhookData.Amount != expectedAmount {
		// Cập nhật log với lỗi amount mismatch
		errorMsg := fmt.Sprintf("Amount mismatch: expected %.0f, got %.0f", expectedAmount, webhookData.Amount)
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount mismatch"})
		return
	}

	// Cập nhật trạng thái đơn hàng thành paid
	err = paymentService.UpdateOrderStatus(order.ID, "paid", &webhookData.TransactionID)
	if err != nil {
		// Cập nhật log với lỗi update order status
		errorMsg := "Failed to update order status: " + err.Error()
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	// Cộng credit cho user
	creditService := service.NewCreditService()
	amountUSD, _ := order.AmountUSD.Float64()
	err = creditService.AddCredits(order.UserID, amountUSD,
		fmt.Sprintf("Nạp credit qua Sepay - %s", webhookData.OrderCode),
		webhookData.TransactionID)
	if err != nil {
		// Cập nhật log với lỗi add credits
		errorMsg := "Failed to add credits: " + err.Error()
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add credits"})
		return
	}

	// Cập nhật log thành công
	config.Db.Model(&logEntry).Updates(map[string]interface{}{
		"processing_status":  "processed",
		"processing_time_ms": int(time.Since(startTime).Milliseconds()),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":    "Payment processed successfully",
		"order_code": webhookData.OrderCode,
		"status":     "paid",
	})
}
