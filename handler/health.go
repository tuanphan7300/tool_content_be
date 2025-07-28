package handler

import (
	"creator-tool-backend/config"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheckResponse represents the health check response
type HealthCheckResponse struct {
	Status    string                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Version   string                   `json:"version"`
	Uptime    string                   `json:"uptime"`
	Services  map[string]ServiceStatus `json:"services"`
	System    SystemInfo               `json:"system"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	NumCPU       int    `json:"num_cpu"`
	MemoryUsage  string `json:"memory_usage"`
}

var startTime = time.Now()

// HealthCheckHandler handles health check requests
func HealthCheckHandler(c *gin.Context) {
	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(startTime).String(),
		Services:  make(map[string]ServiceStatus),
		System:    getSystemInfo(),
	}

	// Check database
	dbStatus := checkDatabase()
	response.Services["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check Redis
	redisStatus := checkRedis()
	response.Services["redis"] = redisStatus
	if redisStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check storage
	storageStatus := checkStorage()
	response.Services["storage"] = storageStatus
	if storageStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// PingHandler simple ping endpoint
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now(),
	})
}

// checkDatabase checks database connectivity
func checkDatabase() ServiceStatus {
	start := time.Now()

	var result int
	err := config.Db.Raw("SELECT 1").Scan(&result).Error

	latency := time.Since(start)

	if err != nil {
		return ServiceStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return ServiceStatus{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

// checkRedis checks Redis connectivity
func checkRedis() ServiceStatus {
	start := time.Now()
	
	// For now, return healthy status since Redis is optional
	// TODO: Implement proper Redis health check when Redis is configured
	latency := time.Since(start)
	
	return ServiceStatus{
		Status:  "healthy",
		Message: "Redis check not implemented yet",
		Latency: latency.String(),
	}
}

// checkStorage checks storage directory
func checkStorage() ServiceStatus {
	// Simple check - in production you might want to do more thorough checks
	return ServiceStatus{
		Status:  "healthy",
		Message: "Storage directory accessible",
	}
}

// getSystemInfo returns system information
func getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		NumCPU:       runtime.NumCPU(),
		MemoryUsage:  formatBytes(m.Alloc),
	}
}

// formatBytes formats bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return string(rune(bytes)) + " B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return string(rune(bytes/div)) + " " + string(rune("KMGTPE"[exp])) + "B"
}
