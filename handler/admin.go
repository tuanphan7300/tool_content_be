package handler

import (
	"creator-tool-backend/config"
	"net/http"
	"strconv"
	"time"

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

// AdminLoginHandler handles admin login
func AdminLoginHandler(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get database connection
	db := c.MustGet("db").(*gorm.DB)

	// Query admin user using GORM
	var admin config.AdminUser
	err := db.Where("username = ? AND is_active = ?", req.Username, true).First(&admin).Error

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if account is locked
	if admin.LockedUntil != nil && admin.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is temporarily locked"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Update last login
	now := time.Now()
	db.Model(&admin).Update("last_login", now)

	// Generate JWT token
	token, expiresAt, err := generateAdminJWT(admin.ID, admin.Username, admin.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user stats"})
		return
	}

	// Get active users (logged in last 30 days)
	err = db.Model(&config.Users{}).Where("updated_at > ?", time.Now().AddDate(0, 0, -30)).Count(&stats.ActiveUsers).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active user stats"})
		return
	}

	// Get process stats
	err = db.Model(&config.UserProcessStatus{}).Count(&stats.TotalProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get process stats"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "processing").Count(&stats.ProcessingProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get processing stats"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "completed").Count(&stats.CompletedProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get completed stats"})
		return
	}

	err = db.Model(&config.UserProcessStatus{}).Where("status = ?", "failed").Count(&stats.FailedProcesses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get failed stats"})
		return
	}

	// Get upload stats
	err = db.Model(&config.CaptionHistory{}).Count(&stats.TotalUploads).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get upload stats"})
		return
	}

	// Get total credits
	err = db.Model(&config.UserCredits{}).Select("COALESCE(SUM(total_credits), 0)").Scan(&stats.TotalCredits).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get credit stats"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch process statuses"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// Update process status using GORM
	var process config.UserProcessStatus
	err := db.First(&process, processID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Process not found"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update process status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Process status updated successfully"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch upload history"})
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

// Helper function to generate admin JWT token
func generateAdminJWT(adminID int, username, role string) (string, time.Time, error) {
	// This should use the same JWT secret as regular users
	// Implementation depends on your JWT setup
	// For now, return a placeholder
	expiresAt := time.Now().Add(24 * time.Hour)
	return "admin_token_placeholder", expiresAt, nil
}
