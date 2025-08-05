package handler

import (
	"net/http"
	"strconv"
	"time"

	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
)

// GetCacheStatsHandler lấy thống kê cache
func GetCacheStatsHandler(c *gin.Context) {
	cacheService := service.NewCacheService()
	stats := cacheService.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"cache_stats": stats,
	})
}

// CleanupCacheHandler dọn dẹp cache đã hết hạn
func CleanupCacheHandler(c *gin.Context) {
	cacheService := service.NewCacheService()
	err := cacheService.CleanupExpired()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cleanup cache",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache cleanup completed successfully",
	})
}

// DeleteCacheEntryHandler xóa entry cache cụ thể
func DeleteCacheEntryHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache key is required"})
		return
	}

	cacheService := service.NewCacheService()
	err := cacheService.Delete(key)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete cache entry",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache entry deleted successfully",
		"key":     key,
	})
}

// GetCacheEntryHandler lấy thông tin entry cache
func GetCacheEntryHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache key is required"})
		return
	}

	cacheService := service.NewCacheService()
	entry, err := cacheService.Get(key)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Cache entry not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cache_entry": entry,
	})
}

// ClearAllCacheHandler xóa tất cả cache
func ClearAllCacheHandler(c *gin.Context) {
	cacheService := service.NewCacheService()

	// Lấy thống kê trước khi xóa
	stats := cacheService.GetStats()

	// Dọn dẹp tất cả cache
	err := cacheService.CleanupExpired()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clear cache",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All cache cleared successfully",
		"cleared_stats": stats,
	})
}

// SetCacheTTLHandler set TTL cho cache entry
func SetCacheTTLHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache key is required"})
		return
	}

	ttlStr := c.PostForm("ttl_hours")
	if ttlStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TTL hours is required"})
		return
	}

	ttlHours, err := strconv.Atoi(ttlStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TTL format"})
		return
	}

	cacheService := service.NewCacheService()

	// Lấy entry hiện tại
	entry, err := cacheService.Get(key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Cache entry not found",
			"details": err.Error(),
		})
		return
	}

	// Cập nhật TTL
	err = cacheService.Set(key, entry.Value, entry.Type, time.Duration(ttlHours)*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update cache TTL",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Cache TTL updated successfully",
		"key":       key,
		"ttl_hours": ttlHours,
	})
}
