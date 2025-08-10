package handler

import (
	"net/http"
	"strconv"

	"creator-tool-backend/model"
	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	feedbackService *service.FeedbackService
}

func NewFeedbackHandler(feedbackService *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
	}
}

// CreateFeedback tạo feedback mới
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req model.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	feedback, err := h.feedbackService.CreateFeedback(userID.(uint), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feedback"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Feedback đã được gửi thành công",
		"data":    feedback,
	})
}

// GetUserFeedbacks lấy danh sách feedback của user
func (h *FeedbackHandler) GetUserFeedbacks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	feedbacks, err := h.feedbackService.GetUserFeedbacks(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feedbacks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": feedbacks,
	})
}

// GetAllFeedbacks (Admin only) lấy tất cả feedback
func (h *FeedbackHandler) GetAllFeedbacks(c *gin.Context) {
	feedbacks, err := h.feedbackService.GetAllFeedbacks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feedbacks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": feedbacks,
	})
}

// UpdateFeedback (Admin only) cập nhật feedback
func (h *FeedbackHandler) UpdateFeedback(c *gin.Context) {
	feedbackIDStr := c.Param("id")
	feedbackID, err := strconv.ParseUint(feedbackIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback ID"})
		return
	}

	var req model.UpdateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	err = h.feedbackService.UpdateFeedback(uint(feedbackID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback đã được cập nhật thành công",
	})
}

// GetFeedbackByID lấy chi tiết feedback theo ID
func (h *FeedbackHandler) GetFeedbackByID(c *gin.Context) {
	feedbackIDStr := c.Param("id")
	feedbackID, err := strconv.ParseUint(feedbackIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback ID"})
		return
	}

	feedback, err := h.feedbackService.GetFeedbackByID(uint(feedbackID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feedback not found"})
		return
	}

	// Kiểm tra quyền truy cập
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Chỉ cho phép user xem feedback của chính mình hoặc admin
	if feedback.UserID != userID.(uint) {
		// Kiểm tra xem có phải admin không
		userRole, exists := c.Get("user_role")
		if !exists || userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": feedback,
	})
}
