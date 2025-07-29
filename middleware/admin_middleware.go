package middleware

import (
	"creator-tool-backend/config"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminClaims represents JWT claims for admin
type AdminClaims struct {
	AdminID  int    `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	// jwt.RegisteredClaims // TODO: Implement JWT properly
}

// AdminAuthMiddleware checks if user is authenticated as admin
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Yêu cầu header Authorization"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Định dạng header Authorization không hợp lệ"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// TODO: Implement proper JWT validation
		// For now, just check if token exists
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			c.Abort()
			return
		}

		// Temporary: Extract admin info from token (simple format: admin_id:username:role)
		// In production, use proper JWT
		claims := &AdminClaims{
			AdminID:  1,             // TODO: Extract from JWT
			Username: "admin",       // TODO: Extract from JWT
			Role:     "super_admin", // TODO: Extract from JWT
		}

		// Check if admin still exists and is active using GORM
		db := c.MustGet("db").(*gorm.DB)
		var admin config.AdminUser
		err := db.Where("id = ? AND is_active = ?", claims.AdminID, true).First(&admin).Error
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tài khoản admin không tồn tại hoặc không hoạt động"})
			c.Abort()
			return
		}

		// Set admin info in context
		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Set("admin_role", claims.Role)

		c.Next()
	}
}

// AdminPermissionMiddleware checks if admin has required permission
func AdminPermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.MustGet("admin_id").(int)
		adminRole := c.MustGet("admin_role").(string)

		// Super admin has all permissions
		if adminRole == "super_admin" {
			c.Next()
			return
		}

		// Check specific permission using GORM
		db := c.MustGet("db").(*gorm.DB)
		var admin config.AdminUser
		err := db.Select("permissions").Where("id = ?", adminID).First(&admin).Error
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không thể kiểm tra quyền"})
			c.Abort()
			return
		}

		// Parse permissions from JSON
		var permissions []string
		if admin.Permissions != nil {
			permissionsJSON, err := json.Marshal(admin.Permissions)
			if err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Định dạng quyền không hợp lệ"})
				c.Abort()
				return
			}

			if err := json.Unmarshal(permissionsJSON, &permissions); err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Định dạng quyền không hợp lệ"})
				c.Abort()
				return
			}
		}

		// Check if admin has required permission
		hasPermission := false
		for _, permission := range permissions {
			if permission == requiredPermission || permission == "*" {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminRoleMiddleware checks if admin has required role
func AdminRoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminRole := c.MustGet("admin_role").(string)

		hasRole := false
		for _, role := range requiredRoles {
			if adminRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ vai trò"})
			c.Abort()
			return
		}

		c.Next()
	}
}
