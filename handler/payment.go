package handler

import (
	"creator-tool-backend/config"
	"creator-tool-backend/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	var webhookDataInterface map[string]interface{}
	if err := c.ShouldBindJSON(&webhookDataInterface); err != nil {
		// Cập nhật log với lỗi parse JSON
		log.Println("Failed to unmarshal webhook data: %v", err)
	}
	log.Println("webhookDataInterface sepay webhook", webhookDataInterface)

	// Parse webhook data theo format thực tế của Sepay
	var webhookData map[string]interface{}
	if err := json.Unmarshal(rawBody, &webhookData); err != nil {
		log.Printf("Failed to unmarshal webhook data: %v", err)
		webhookData = make(map[string]interface{})
	}

	log.Printf("webhookData sepay webhook %v", webhookData)

	// Extract thông tin từ format Sepay
	var orderCode string
	var amount float64
	var transactionID string

	// Lấy order code từ content hoặc code
	if content, ok := webhookData["content"].(string); ok {
		// Tìm order code trong content (format: PAY202508020808253439)
		if strings.Contains(content, "PAY") {
			parts := strings.Split(content, ".")
			for _, part := range parts {
				if strings.HasPrefix(part, "PAY") {
					orderCode = part
					break
				}
			}
		}
	}

	// Nếu không tìm thấy trong content, thử từ code field
	if orderCode == "" {
		if code, ok := webhookData["code"].(string); ok && code != "" {
			orderCode = code
		}
	}

	// Lấy amount
	if transferAmount, ok := webhookData["transferAmount"].(float64); ok {
		amount = transferAmount
	}

	// Lấy transaction ID
	if id, ok := webhookData["id"].(float64); ok {
		transactionID = fmt.Sprintf("%.0f", id)
	}

	// Kiểm tra transferType phải là "in" (tiền vào)
	transferType, _ := webhookData["transferType"].(string)
	if transferType != "in" {
		// Cập nhật log với trạng thái ignored
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "ignored",
			"error_message":      "Transfer type is not 'in': " + transferType,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusOK, gin.H{"message": "Transfer type is not 'in'"})
		return
	}

	// Cập nhật log với thông tin parsed
	config.Db.Model(&logEntry).Updates(map[string]interface{}{
		"order_code":        orderCode,
		"amount":            amount,
		"status":            "success", // Sepay gửi khi thanh toán thành công
		"transaction_id":    transactionID,
		"signature":         "", // Sepay không gửi signature
		"timestamp":         nil,
		"processing_status": "validated",
	})

	// Log thêm thông tin chi tiết
	log.Printf("Sepay webhook processed - Order: %s, Amount: %.0f, Gateway: %v, TransactionID: %s",
		orderCode, amount, webhookData["gateway"], transactionID)

	// Kiểm tra order code có hợp lệ không
	if orderCode == "" {
		// Cập nhật log với lỗi order code không tìm thấy
		errorMsg := "Order code not found in content"
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Order code not found in content"})
		return
	}

	// Tìm đơn hàng theo order code
	paymentService := service.NewPaymentOrderService()
	order, err := paymentService.GetOrderByCode(orderCode)
	if err != nil {
		// Cập nhật log với lỗi order not found
		errorMsg := "Order not found: " + orderCode
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
	if amount != expectedAmount {
		// Cập nhật log với lỗi amount mismatch
		errorMsg := fmt.Sprintf("Amount mismatch: expected %.0f, got %.0f", expectedAmount, amount)
		config.Db.Model(&logEntry).Updates(map[string]interface{}{
			"processing_status":  "failed",
			"error_message":      errorMsg,
			"processing_time_ms": int(time.Since(startTime).Milliseconds()),
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount mismatch"})
		return
	}

	// Cập nhật trạng thái đơn hàng thành paid
	err = paymentService.UpdateOrderStatus(order.ID, "paid", &transactionID)
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
		fmt.Sprintf("Nạp credit qua Sepay - %s", orderCode),
		transactionID)
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
		"order_code": orderCode,
		"status":     "paid",
	})
}
