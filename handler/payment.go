package handler

import (
	"creator-tool-backend/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
			"id":            order.ID,
			"order_code":    order.OrderCode,
			"amount_vnd":    order.AmountVND.String(),
			"amount_usd":    order.AmountUSD.String(),
			"qr_code_url":   order.QRCodeURL,
			"expires_at":    order.ExpiresAt,
			"bank_account":  order.BankAccount,
			"bank_name":     order.BankName,
			"order_status":  order.OrderStatus,
			"created_at":    order.CreatedAt,
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
			"id":            order.ID,
			"order_code":    order.OrderCode,
			"amount_vnd":    order.AmountVND.String(),
			"amount_usd":    order.AmountUSD.String(),
			"qr_code_url":   order.QRCodeURL,
			"expires_at":    order.ExpiresAt,
			"bank_account":  order.BankAccount,
			"bank_name":     order.BankName,
			"order_status":  order.OrderStatus,
			"paid_at":       order.PaidAt,
			"created_at":    order.CreatedAt,
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