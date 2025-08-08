package handler

import (
	"creator-tool-backend/config"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminUserResponse represents admin user response (without password)
type AdminUserResponse struct {
	ID            int                    `json:"id"`
	Username      string                 `json:"username"`
	Email         string                 `json:"email"`
	Name          string                 `json:"name"`
	Role          string                 `json:"role"`
	IsActive      bool                   `json:"is_active"`
	Permissions   map[string]interface{} `json:"permissions"`
	LoginAttempts int                    `json:"login_attempts"`
	LockedUntil   *time.Time             `json:"locked_until"`
	LastLogin     *time.Time             `json:"last_login"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// AdminLoginRequest represents admin login request
type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AdminLoginResponse represents admin login response
type AdminLoginResponse struct {
	Token     string            `json:"token"`
	Admin     AdminUserResponse `json:"admin"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// AdminDashboardStats represents dashboard statistics
type AdminDashboardStats struct {
	TotalUsers          int64   `json:"total_users"`
	ActiveUsers         int64   `json:"active_users"`
	TotalProcesses      int64   `json:"total_processes"`
	ProcessingProcesses int64   `json:"processing_processes"`
	CompletedProcesses  int64   `json:"completed_processes"`
	FailedProcesses     int64   `json:"failed_processes"`
	TotalUploads        int64   `json:"total_uploads"`
	TotalCredits        float64 `json:"total_credits"`
}

// AdminUserListItem represents user item for admin list
type AdminUserListItem struct {
	ID            uint      `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	GoogleID      string    `json:"google_id"`
	EmailVerified bool      `json:"email_verified"`
	AuthProvider  string    `json:"auth_provider"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TotalCredits  float64   `json:"total_credits"`
	UsedCredits   float64   `json:"used_credits"`
}

// AdminProcessListItem represents process item for admin list
type AdminProcessListItem struct {
	ID          uint       `json:"id"`
	UserID      uint       `json:"user_id"`
	Status      string     `json:"status"`
	ProcessType string     `json:"process_type"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	VideoID     *uint      `json:"video_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UserEmail   string     `json:"user_email"`
	UserName    string     `json:"user_name"`
}

// AdminUploadListItem represents upload item for admin list
type AdminUploadListItem struct {
	ID                  uint      `json:"id"`
	UserID              uint      `json:"user_id"`
	VideoFilename       string    `json:"video_filename"`
	VideoFilenameOrigin string    `json:"video_filename_origin"`
	ProcessType         string    `json:"process_type"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	UserEmail           string    `json:"user_email"`
	UserName            string    `json:"user_name"`
}

// Credit usage summary for a video (for list API)
type AdminCreditUsageVideoItem struct {
	VideoID         uint      `json:"video_id"`
	VideoFilename   string    `json:"video_filename"`
	UserID          uint      `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserEmail       string    `json:"user_email"`
	TotalCredit     float64   `json:"total_credit"`
	TotalBaseAmount float64   `json:"total_base_amount"`
	TotalProfit     float64   `json:"total_profit"`
	CreatedAt       time.Time `json:"created_at"`
	ProcessType     string    `json:"process_type"`
}

// Credit usage detail for a video (for detail API)
type AdminCreditUsageDetailItem struct {
	Service     string    `json:"service"`
	Amount      float64   `json:"amount"`
	BaseAmount  float64   `json:"base_amount"`
	Profit      float64   `json:"profit"`
	PricingType string    `json:"pricing_type"`
	UnitsUsed   float64   `json:"units_used"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminLoginHandler handles admin login
func AdminLoginHandler(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	// Get database connection
	db := c.MustGet("db").(*gorm.DB)

	// Query admin user using GORM
	var admin config.AdminUser
	err := db.Where("username = ? AND is_active = ?", req.Username, true).First(&admin).Error

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tên đăng nhập hoặc mật khẩu không đúng"})
		return
	}

	// Check if account is locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tài khoản tạm thời bị khóa"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tên đăng nhập hoặc mật khẩu không đúng"})
		return
	}

	// Update last login
	now := time.Now()
	db.Model(&admin).Update("last_login", now)

	// Generate JWT token
	token, expiresAt, err := generateAdminJWT(admin.ID, admin.Username, admin.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	// Convert to response format
	adminResponse := AdminUserResponse{
		ID:            admin.ID,
		Username:      admin.Username,
		Email:         admin.Email,
		Name:          admin.Name,
		Role:          admin.Role,
		IsActive:      admin.IsActive,
		Permissions:   make(map[string]interface{}), // TODO: Parse from admin.Permissions
		LoginAttempts: admin.LoginAttempts,
		LockedUntil:   admin.LockedUntil,
		LastLogin:     &now,
		CreatedAt:     admin.CreatedAt,
		UpdatedAt:     admin.UpdatedAt,
	}

	c.JSON(http.StatusOK, AdminLoginResponse{
		Token:     token,
		Admin:     adminResponse,
		ExpiresAt: expiresAt,
	})
}

// AdminDashboardHandler returns dashboard statistics
func AdminDashboardHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var stats AdminDashboardStats

	// Get total users
	err := db.Model(&config.Users{}).Count(&stats.TotalUsers).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê người dùng"})
		return
	}

	// Get active users (logged in last 30 days)
	err = db.Model(&config.Users{}).Where("created_at > ?", time.Now().AddDate(0, 0, -30)).Count(&stats.ActiveUsers).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê người dùng hoạt động"})
		return
	}

	// Get process stats
	err = db.Model(&config.UserProcessStatus{}).Count(&stats.TotalProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê quy trình"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "processing").Count(&stats.ProcessingProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê đang xử lý"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "completed").Count(&stats.CompletedProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê hoàn thành"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "failed").Count(&stats.FailedProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê thất bại"})
		return
	}

	// Get upload stats
	err = db.Model(&config.CaptionHistory{}).Count(&stats.TotalUploads).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê tải lên"})
		return
	}

	// Get total credits
	err = db.Model(&config.UserCredits{}).Select("COALESCE(SUM(total_credits), 0)").Scan(&stats.TotalCredits).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy thống kê tín dụng"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// AdminUsersHandler returns list of users with pagination
func AdminUsersHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	offset := (page - 1) * limit

	// Get users with credits using separate queries
	var users []config.Users
	query := db.Model(&config.Users{})

	if search != "" {
		query = query.Where("email LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get users with pagination
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách người dùng"})
		return
	}

	// Get credits for each user
	var userList []AdminUserListItem
	for _, user := range users {
		var credits config.UserCredits
		db.Where("user_id = ?", user.ID).First(&credits)

		userList = append(userList, AdminUserListItem{
			ID:            user.ID,
			Email:         user.Email,
			Name:          user.Name,
			GoogleID:      user.GoogleID,
			EmailVerified: user.EmailVerified,
			AuthProvider:  user.AuthProvider,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.CreatedAt, // Users struct doesn't have UpdatedAt, use CreatedAt
			TotalCredits:  credits.TotalCredits,
			UsedCredits:   credits.UsedCredits,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userList,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

// AdminProcessStatusHandler returns list of process statuses
func AdminProcessStatusHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	processType := c.Query("process_type")

	offset := (page - 1) * limit

	// Get processes with separate user query
	var processes []config.UserProcessStatus
	query := db.Model(&config.UserProcessStatus{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if processType != "" {
		query = query.Where("process_type = ?", processType)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get processes with pagination
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&processes).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải trạng thái quy trình"})
		return
	}

	// Get user info for each process
	var processList []AdminProcessListItem
	for _, process := range processes {
		var user config.Users
		db.First(&user, process.UserID)

		processList = append(processList, AdminProcessListItem{
			ID:          process.ID,
			UserID:      process.UserID,
			Status:      process.Status,
			ProcessType: process.ProcessType,
			StartedAt:   process.StartedAt,
			CompletedAt: process.CompletedAt,
			VideoID:     process.VideoID,
			CreatedAt:   process.CreatedAt,
			UpdatedAt:   process.UpdatedAt,
			UserEmail:   user.Email,
			UserName:    user.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"processes": processList,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

// AdminUpdateProcessStatusHandler updates process status
func AdminUpdateProcessStatusHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	processID := c.Param("id")
	status := c.PostForm("status")

	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái là bắt buộc"})
		return
	}

	// Validate status
	validStatuses := []string{"processing", "completed", "failed", "cancelled"}
	isValid := false
	for _, s := range validStatuses {
		if s == status {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ"})
		return
	}

	// Update process status using GORM
	var process config.UserProcessStatus
	err := db.First(&process, processID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy quy trình"})
		return
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Set completed_at if status is final
	if status == "completed" || status == "failed" || status == "cancelled" {
		now := time.Now()
		updates["completed_at"] = &now
	} else {
		updates["completed_at"] = nil
	}

	err = db.Model(&process).Updates(updates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái quy trình"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật trạng thái quy trình thành công"})
}

// AdminUploadHistoryHandler returns upload history
func AdminUploadHistoryHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	processType := c.Query("process_type")
	search := c.Query("search")

	offset := (page - 1) * limit

	// Get uploads with separate user query
	var uploads []config.CaptionHistory
	query := db.Model(&config.CaptionHistory{})

	if processType != "" {
		query = query.Where("process_type = ?", processType)
	}

	if search != "" {
		query = query.Where("video_filename LIKE ? OR video_filename_origin LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get uploads with pagination
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&uploads).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải lịch sử tải lên"})
		return
	}

	// Get user info for each upload
	var uploadList []AdminUploadListItem
	for _, upload := range uploads {
		var user config.Users
		db.First(&user, upload.UserID)

		uploadList = append(uploadList, AdminUploadListItem{
			ID:                  upload.ID,
			UserID:              upload.UserID,
			VideoFilename:       upload.VideoFilename,
			VideoFilenameOrigin: upload.VideoFilenameOrigin,
			ProcessType:         upload.ProcessType,
			CreatedAt:           upload.CreatedAt,
			UpdatedAt:           upload.UpdatedAt,
			UserEmail:           user.Email,
			UserName:            user.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploadList,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

// GET /admin/payment/email-logs
func GetPaymentEmailLogs(c *gin.Context) {
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	status := c.Query("status")
	orderCode := c.Query("order_code")

	var logs []config.PaymentEmailLog
	db := config.Db.Order("created_at desc")
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if orderCode != "" {
		db = db.Where("order_code = ?", orderCode)
	}
	db = db.Offset((page - 1) * limit).Limit(limit)
	db.Find(&logs)

	c.JSON(200, gin.H{
		"logs":  logs,
		"page":  page,
		"limit": limit,
	})
}

// GetSepayWebhookLogs lấy danh sách log webhook từ Sepay
func GetSepayWebhookLogs(c *gin.Context) {
	// Lấy query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")
	orderCode := c.Query("order_code")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	// Build query
	query := config.Db.Model(&config.SepayWebhookLog{})

	// Apply filters
	if status != "" {
		query = query.Where("processing_status = ?", status)
	}
	if orderCode != "" {
		query = query.Where("order_code LIKE ?", "%"+orderCode+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get logs
	var logs []config.SepayWebhookLog
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get webhook logs"})
		return
	}

	// Format response
	var formattedLogs []gin.H
	for _, log := range logs {
		formattedLogs = append(formattedLogs, gin.H{
			"id":                 log.ID,
			"order_code":         log.OrderCode,
			"amount":             log.Amount,
			"status":             log.Status,
			"transaction_id":     log.TransactionID,
			"signature":          log.Signature,
			"timestamp":          log.Timestamp,
			"raw_payload":        log.RawPayload,
			"headers":            log.Headers,
			"ip_address":         log.IPAddress,
			"user_agent":         log.UserAgent,
			"processing_status":  log.ProcessingStatus,
			"error_message":      log.ErrorMessage,
			"processing_time_ms": log.ProcessingTimeMs,
			"created_at":         log.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        formattedLogs,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// GetAdminPaymentOrders lấy danh sách đơn hàng thanh toán cho admin
func GetAdminPaymentOrders(c *gin.Context) {
	// Lấy query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")
	orderCode := c.Query("order_code")
	userEmail := c.Query("user_email")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	// Build query với join user - select rõ ràng các field
	query := config.Db.Table("payment_orders").
		Select(`
			payment_orders.id,
			payment_orders.user_id,
			payment_orders.order_code,
			payment_orders.amount_vnd,
			payment_orders.amount_usd,
			payment_orders.bank_account,
			payment_orders.bank_name,
			payment_orders.order_status,
			payment_orders.payment_method,
			payment_orders.expires_at,
			payment_orders.paid_at,
			payment_orders.transaction_id,
			payment_orders.created_at,
			users.email as user_email,
			users.name as user_name
		`).
		Joins("LEFT JOIN users ON payment_orders.user_id = users.id")

	// Apply filters
	if status != "" {
		query = query.Where("payment_orders.order_status = ?", status)
	}
	if orderCode != "" {
		query = query.Where("payment_orders.order_code LIKE ?", "%"+orderCode+"%")
	}
	if userEmail != "" {
		query = query.Where("users.email LIKE ?", "%"+userEmail+"%")
	}
	if dateFrom != "" {
		query = query.Where("payment_orders.created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("payment_orders.created_at <= ?", dateTo+" 23:59:59")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Define struct để map dữ liệu chính xác
	type PaymentOrderRow struct {
		ID            uint       `json:"id"`
		UserID        uint       `json:"user_id"`
		OrderCode     string     `json:"order_code"`
		AmountVND     string     `json:"amount_vnd"` // MySQL decimal trả về string
		AmountUSD     string     `json:"amount_usd"` // MySQL decimal trả về string
		BankAccount   string     `json:"bank_account"`
		BankName      string     `json:"bank_name"`
		OrderStatus   string     `json:"order_status"`
		PaymentMethod string     `json:"payment_method"`
		ExpiresAt     time.Time  `json:"expires_at"`
		PaidAt        *time.Time `json:"paid_at"`
		TransactionID *string    `json:"transaction_id"`
		CreatedAt     time.Time  `json:"created_at"`
		UserEmail     *string    `json:"user_email"`
		UserName      *string    `json:"user_name"`
	}

	// Get orders
	var orders []PaymentOrderRow
	err := query.Order("payment_orders.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment orders"})
		return
	}

	// Debug log để xem dữ liệu thực tế
	log.Printf("Raw orders from DB: %+v", orders)

	// Format response
	var formattedOrders []gin.H
	for _, order := range orders {
		// Convert amount_vnd to float64
		var amountVND float64
		if order.AmountVND != "" {
			if f, err := strconv.ParseFloat(order.AmountVND, 64); err == nil {
				amountVND = f
			}
		}

		// Convert amount_usd to float64
		var amountUSD float64
		if order.AmountUSD != "" {
			if f, err := strconv.ParseFloat(order.AmountUSD, 64); err == nil {
				amountUSD = f
			}
		}

		formattedOrders = append(formattedOrders, gin.H{
			"id":             order.ID,
			"user_id":        order.UserID,
			"user_email":     order.UserEmail,
			"user_name":      order.UserName,
			"order_code":     order.OrderCode,
			"amount_vnd":     amountVND,
			"amount_usd":     amountUSD,
			"bank_account":   order.BankAccount,
			"bank_name":      order.BankName,
			"order_status":   order.OrderStatus,
			"payment_method": order.PaymentMethod,
			"expires_at":     order.ExpiresAt,
			"paid_at":        order.PaidAt,
			"transaction_id": order.TransactionID,
			"created_at":     order.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"orders":      formattedOrders,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// GetAdminPaymentStats lấy thống kê thanh toán cho admin
func GetAdminPaymentStats(c *gin.Context) {
	// Thống kê theo trạng thái
	type StatusStat struct {
		OrderStatus string `json:"order_status"`
		Count       int64  `json:"count"`
		TotalAmount string `json:"total_amount"`
	}
	var statusStats []StatusStat
	err := config.Db.Table("payment_orders").
		Select("order_status, COUNT(*) as count, COALESCE(SUM(amount_vnd), 0) as total_amount").
		Group("order_status").
		Find(&statusStats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment stats"})
		return
	}

	// Thống kê theo ngày (7 ngày gần nhất)
	type DailyStat struct {
		Date        string `json:"date"`
		Count       int64  `json:"count"`
		TotalAmount string `json:"total_amount"`
	}
	var dailyStats []DailyStat
	err = config.Db.Table("payment_orders").
		Select("DATE(created_at) as date, COUNT(*) as count, COALESCE(SUM(amount_vnd), 0) as total_amount").
		Where("created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)").
		Group("DATE(created_at)").
		Order("date DESC").
		Find(&dailyStats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get daily stats"})
		return
	}

	// Tổng quan
	var totalOrders int64
	var totalAmountStr string
	var pendingOrders int64
	var completedOrders int64

	config.Db.Table("payment_orders").Count(&totalOrders)
	config.Db.Table("payment_orders").Select("COALESCE(SUM(amount_vnd), 0) as total").Scan(&totalAmountStr)
	config.Db.Table("payment_orders").Where("order_status = 'pending'").Count(&pendingOrders)
	config.Db.Table("payment_orders").Where("order_status = 'paid'").Count(&completedOrders)

	c.JSON(http.StatusOK, gin.H{
		"status_stats":     statusStats,
		"daily_stats":      dailyStats,
		"total_orders":     totalOrders,
		"total_amount":     totalAmountStr,
		"pending_orders":   pendingOrders,
		"completed_orders": completedOrders,
	})
}

// CancelAdminPaymentOrder hủy đơn hàng thanh toán (admin)
func CancelAdminPaymentOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Parse order ID
	id, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Tìm đơn hàng
	var order config.PaymentOrder
	err = config.Db.First(&order, id).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Kiểm tra trạng thái đơn hàng
	if order.OrderStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled"})
		return
	}

	// Cập nhật trạng thái thành cancelled
	paymentService := service.NewPaymentOrderService()
	err = paymentService.UpdateOrderStatus(uint(id), "cancelled", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Order cancelled successfully",
		"order_status": "cancelled",
	})
}

// GET /admin/credit-usage
func AdminCreditUsageListHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Parse date range filters
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	// Lấy tất cả process status có video_id
	var processStatuses []config.UserProcessStatus
	query := db.Model(&config.UserProcessStatus{}).
		Where("video_id IS NOT NULL")

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Apply date range filters
	if fromDate != "" {
		query = query.Where("created_at >= ?", fromDate)
	}
	if toDate != "" {
		query = query.Where("created_at <= ?", toDate+" 23:59:59")
	}

	// Get total count for pagination
	var total int64
	query.Count(&total)

	query = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset)

	if err := query.Find(&processStatuses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách process status"})
		return
	}

	// Lấy video details
	var videoIDs []uint
	for _, ps := range processStatuses {
		if ps.VideoID != nil {
			videoIDs = append(videoIDs, *ps.VideoID)
		}
	}

	var videos []config.CaptionHistory
	if len(videoIDs) > 0 {
		db.Where("id IN (?)", videoIDs).Find(&videos)
	}

	// Create video map for quick lookup
	videoMap := make(map[uint]config.CaptionHistory)
	for _, v := range videos {
		videoMap[v.ID] = v
	}

	// Preload all users for mapping
	var users []config.Users
	db.Find(&users)
	userMap := map[uint]config.Users{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	// For each process status, calculate credit usage
	var result []AdminCreditUsageVideoItem
	for _, ps := range processStatuses {
		if ps.VideoID == nil {
			continue
		}

		video, exists := videoMap[*ps.VideoID]
		if !exists {
			continue
		}

		// Tính tổng credit cho video này
		var txs []config.CreditTransaction
		db.Where("video_id = ? AND transaction_type = ?", *ps.VideoID, "deduct").Find(&txs)

		totalCredit := 0.0
		totalBase := 0.0
		for _, t := range txs {
			totalCredit += t.Amount
			totalBase += t.BaseAmount
		}

		// Đặc biệt xử lý cho burn-sub process type
		// burn-sub có thể có video_id = NULL trong credit_transaction
		if ps.ProcessType == "burn-sub" && totalCredit == 0 {
			// Tìm giao dịch burn-sub có video_id = NULL trong cùng thời gian
			var burnSubTxs []config.CreditTransaction
			db.Where("service = ? AND transaction_type = ? AND video_id IS NULL AND created_at BETWEEN ? AND ?",
				"burn-sub", "deduct", ps.CreatedAt.Add(-time.Minute), ps.CreatedAt.Add(time.Minute)).
				Find(&burnSubTxs)

			for _, t := range burnSubTxs {
				totalCredit += t.Amount
				totalBase += t.BaseAmount
			}
		}

		// Chỉ thêm vào kết quả nếu có giao dịch credit
		if totalCredit > 0 {
			profit := totalCredit - totalBase
			u := userMap[ps.UserID]

			result = append(result, AdminCreditUsageVideoItem{
				VideoID:         *ps.VideoID,
				VideoFilename:   video.VideoFilename,
				UserID:          ps.UserID,
				UserName:        u.Name,
				UserEmail:       u.Email,
				TotalCredit:     totalCredit,
				TotalBaseAmount: totalBase,
				TotalProfit:     profit,
				CreatedAt:       ps.CreatedAt,
				ProcessType:     ps.ProcessType,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": result,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (int(total) + limit - 1) / limit,
		},
	})
}

// GET /admin/credit-usage/:video_id
func AdminCreditUsageDetailHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	videoIDStr := c.Param("video_id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video ID không hợp lệ"})
		return
	}

	// Lấy thông tin video
	var video config.CaptionHistory
	if err := db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video không tồn tại"})
		return
	}

	// Lấy process status để xác định process type
	var processStatus config.UserProcessStatus
	db.Where("video_id = ?", videoID).Order("created_at DESC").First(&processStatus)

	// Lấy tất cả giao dịch credit cho video này
	var txs []config.CreditTransaction
	db.Where("video_id = ? AND transaction_type = ?", videoID, "deduct").Order("created_at").Find(&txs)

	// Nếu là burn-sub và không có giao dịch, tìm giao dịch có video_id = NULL
	if processStatus.ProcessType == "burn-sub" && len(txs) == 0 {
		// Tìm giao dịch burn-sub có video_id = NULL trong khoảng thời gian gần với process status
		var burnSubTxs []config.CreditTransaction
		db.Where("service = ? AND transaction_type = ? AND video_id IS NULL AND created_at BETWEEN ? AND ?",
			"burn-sub", "deduct", processStatus.CreatedAt.Add(-time.Minute), processStatus.CreatedAt.Add(time.Minute)).
			Order("created_at").
			Find(&burnSubTxs)

		txs = burnSubTxs
	}

	var details []AdminCreditUsageDetailItem
	for _, t := range txs {
		profit := t.Amount - t.BaseAmount
		details = append(details, AdminCreditUsageDetailItem{
			Service:     t.Service,
			Amount:      t.Amount,
			BaseAmount:  t.BaseAmount,
			Profit:      profit,
			PricingType: t.PricingType,
			UnitsUsed:   t.UnitsUsed,
			Description: t.Description,
			CreatedAt:   t.CreatedAt,
		})
	}

	// Tính tổng
	totalCredit := 0.0
	totalBaseAmount := 0.0
	totalProfit := 0.0
	for _, t := range txs {
		totalCredit += t.Amount
		totalBaseAmount += t.BaseAmount
		totalProfit += (t.Amount - t.BaseAmount)
	}

	c.JSON(http.StatusOK, gin.H{
		"video_info": gin.H{
			"video_id":       video.ID,
			"video_filename": video.VideoFilename,
			"process_type":   processStatus.ProcessType,
			"created_at":     video.CreatedAt,
		},
		"details": details,
		"summary": gin.H{
			"total_credit":      totalCredit,
			"total_base_amount": totalBaseAmount,
			"total_profit":      totalProfit,
			"transaction_count": len(txs),
		},
	})
}

// Helper function to generate admin JWT token
func generateAdminJWT(adminID int, username, role string) (string, time.Time, error) {
	// This should use the same JWT secret as regular users
	// Implementation depends on your JWT setup
	// For now, return a placeholder
	expiresAt := time.Now().Add(24 * time.Hour)
	return "admin_token_placeholder", expiresAt, nil
}
