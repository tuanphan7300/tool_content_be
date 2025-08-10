package model

import (
	"time"
)

type Feedback struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint       `json:"user_id" gorm:"not null;index"`
	Subject       string     `json:"subject" gorm:"not null;size:255"`
	Message       string     `json:"message" gorm:"not null;type:text"`
	Status        string     `json:"status" gorm:"not null;default:pending;type:enum('pending','in_progress','resolved','closed')"`
	Priority      string     `json:"priority" gorm:"not null;default:medium;type:enum('low','medium','high','urgent')"`
	AdminResponse *string    `json:"admin_response" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	ResolvedAt    *time.Time `json:"resolved_at"`
}

type CreateFeedbackRequest struct {
	Subject string `json:"subject" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type UpdateFeedbackRequest struct {
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	AdminResponse *string `json:"admin_response"`
}

type FeedbackResponse struct {
	ID            uint       `json:"id"`
	UserID        uint       `json:"user_id"`
	Subject       string     `json:"subject"`
	Message       string     `json:"message"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	AdminResponse *string    `json:"admin_response"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	UserName      string     `json:"user_name"`
}

// TableName specifies the table name for GORM
func (Feedback) TableName() string {
	return "tool_feedbacks"
}
