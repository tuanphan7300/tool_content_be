package service

import (
	"fmt"

	"creator-tool-backend/model"

	"gorm.io/gorm"
)

type FeedbackService struct {
	db *gorm.DB
}

func NewFeedbackService(db *gorm.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

func (s *FeedbackService) CreateFeedback(userID uint, req model.CreateFeedbackRequest) (*model.Feedback, error) {
	feedback := &model.Feedback{
		UserID:   userID,
		Subject:  req.Subject,
		Message:  req.Message,
		Status:   "pending",
		Priority: "medium",
	}

	if err := s.db.Create(feedback).Error; err != nil {
		return nil, fmt.Errorf("failed to create feedback: %v", err)
	}

	return feedback, nil
}

func (s *FeedbackService) GetUserFeedbacks(userID uint) ([]model.Feedback, error) {
	var feedbacks []model.Feedback

	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&feedbacks).Error; err != nil {
		return nil, fmt.Errorf("failed to query user feedbacks: %v", err)
	}

	return feedbacks, nil
}

func (s *FeedbackService) GetAllFeedbacks() ([]model.FeedbackResponse, error) {
	var feedbacks []model.FeedbackResponse

	// Sử dụng GORM joins thay vì raw SQL
	if err := s.db.Table("tool_feedbacks").
		Select("tool_feedbacks.*, users.name as user_name").
		Joins("LEFT JOIN users ON tool_feedbacks.user_id = users.id").
		Order("tool_feedbacks.created_at DESC").
		Scan(&feedbacks).Error; err != nil {
		return nil, fmt.Errorf("failed to query all feedbacks: %v", err)
	}

	return feedbacks, nil
}

func (s *FeedbackService) UpdateFeedback(feedbackID uint, req model.UpdateFeedbackRequest) error {
	updates := map[string]interface{}{
		"status":         req.Status,
		"priority":       req.Priority,
		"admin_response": req.AdminResponse,
	}

	if req.Status == "resolved" {
		updates["resolved_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}

	if err := s.db.Model(&model.Feedback{}).Where("id = ?", feedbackID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update feedback: %v", err)
	}

	return nil
}

func (s *FeedbackService) GetFeedbackByID(feedbackID uint) (*model.Feedback, error) {
	var feedback model.Feedback

	if err := s.db.Where("id = ?", feedbackID).First(&feedback).Error; err != nil {
		return nil, fmt.Errorf("failed to get feedback: %v", err)
	}

	return &feedback, nil
}
