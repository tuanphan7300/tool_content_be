package handler

import (
	"creator-tool-backend/config"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ServiceConfig represents the service_config table structure
type ServiceConfig struct {
	ID          uint   `json:"id"`
	ServiceType string `json:"service_type"`
	ServiceName string `json:"service_name"`
	IsActive    bool   `json:"is_active"`
	ConfigJSON  string `json:"config_json"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AdminServiceConfigHandler returns all service configurations
func AdminServiceConfigHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var serviceConfigs []config.ServiceConfig
	err := db.Order("service_type, is_active DESC, created_at DESC").Find(&serviceConfigs).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải cấu hình dịch vụ"})
		return
	}

	// Group by service_type for easier frontend handling
	type ServiceTypeGroup struct {
		ServiceType string          `json:"service_type"`
		Services    []ServiceConfig `json:"services"`
	}

	serviceGroups := make(map[string]*ServiceTypeGroup)

	for _, sc := range serviceConfigs {
		if _, exists := serviceGroups[sc.ServiceType]; !exists {
			serviceGroups[sc.ServiceType] = &ServiceTypeGroup{
				ServiceType: sc.ServiceType,
				Services:    []ServiceConfig{},
			}
		}

		serviceConfig := ServiceConfig{
			ID:          sc.ID,
			ServiceType: sc.ServiceType,
			ServiceName: sc.ServiceName,
			IsActive:    sc.IsActive,
			ConfigJSON:  sc.ConfigJSON,
			CreatedAt:   sc.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   sc.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		serviceGroups[sc.ServiceType].Services = append(serviceGroups[sc.ServiceType].Services, serviceConfig)
	}

	// Convert map to slice
	var result []ServiceTypeGroup
	for _, group := range serviceGroups {
		result = append(result, *group)
	}

	c.JSON(http.StatusOK, gin.H{
		"service_configs": result,
	})
}

// AdminUpdateServiceConfigHandler updates service configuration
func AdminUpdateServiceConfigHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		ID       uint `json:"id" binding:"required"`
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Get the service config to update
	var serviceConfig config.ServiceConfig
	if err := db.First(&serviceConfig, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cấu hình dịch vụ"})
		return
	}

	// If we're activating this service, deactivate all others of the same type
	if req.IsActive {
		// Deactivate all services of the same type
		if err := db.Model(&config.ServiceConfig{}).
			Where("service_type = ? AND id != ?", serviceConfig.ServiceType, req.ID).
			Update("is_active", 0).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cấu hình dịch vụ"})
			return
		}
	}

	// Update the target service
	isActiveInt := 0
	if req.IsActive {
		isActiveInt = 1
	}

	if err := db.Model(&serviceConfig).Update("is_active", isActiveInt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cấu hình dịch vụ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật cấu hình dịch vụ thành công"})
}

// AdminAddServiceConfigHandler adds new service configuration
func AdminAddServiceConfigHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		ServiceType string `json:"service_type" binding:"required"`
		ServiceName string `json:"service_name" binding:"required"`
		ConfigJSON  string `json:"config_json"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Check if service already exists
	var existingService config.ServiceConfig
	if err := db.Where("service_type = ? AND service_name = ?", req.ServiceType, req.ServiceName).First(&existingService).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dịch vụ này đã tồn tại"})
		return
	}

	// Create new service config
	newServiceConfig := config.ServiceConfig{
		ServiceType: req.ServiceType,
		ServiceName: req.ServiceName,
		IsActive:    false, // Default to inactive
		ConfigJSON:  req.ConfigJSON,
	}

	if err := db.Create(&newServiceConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo cấu hình dịch vụ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tạo cấu hình dịch vụ thành công"})
}

// AdminDeleteServiceConfigHandler deletes service configuration
func AdminDeleteServiceConfigHandler(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id := c.Param("id")
	serviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	// Check if service is active
	var serviceConfig config.ServiceConfig
	if err := db.First(&serviceConfig, serviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cấu hình dịch vụ"})
		return
	}

	if serviceConfig.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể xóa dịch vụ đang hoạt động"})
		return
	}

	if err := db.Delete(&serviceConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa cấu hình dịch vụ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Xóa cấu hình dịch vụ thành công"})
}
