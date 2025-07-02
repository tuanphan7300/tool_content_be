package handler

import (
	"creator-tool-backend/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetQueueStatus trả về trạng thái của queue
func GetQueueStatus(c *gin.Context) {
	status, err := service.GetQueueStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_status": status,
	})
}

// GetWorkerStatus trả về trạng thái của worker service
func GetWorkerStatus(c *gin.Context) {
	status := service.GetWorkerStatus()
	c.JSON(http.StatusOK, status)
}

// GetJobStatus trả về trạng thái của một job cụ thể
func GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	status, err := service.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"status": status,
	})
}

// GetJobResult trả về kết quả của một job
func GetJobResult(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	result, err := service.GetJobResult(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job result not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"result": result,
	})
}

// WaitForJobCompletion chờ job hoàn thành và trả về kết quả
func WaitForJobCompletion(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Parse timeout từ query parameter
	timeoutStr := c.DefaultQuery("timeout", "300") // Default 5 phút
	timeout, err := time.ParseDuration(timeoutStr + "s")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeout format"})
		return
	}

	// Chờ job hoàn thành
	result, err := service.WaitForJobCompletion(jobID, timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"status": "completed",
		"result": result,
	})
}

// StartWorkerService khởi động worker service (admin only)
func StartWorkerService(c *gin.Context) {
	workerService := service.GetWorkerService()
	if workerService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Worker service not initialized"})
		return
	}

	workerService.Start()
	c.JSON(http.StatusOK, gin.H{"message": "Worker service started"})
}

// StopWorkerService dừng worker service (admin only)
func StopWorkerService(c *gin.Context) {
	workerService := service.GetWorkerService()
	if workerService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Worker service not initialized"})
		return
	}

	workerService.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "Worker service stopped"})
}
