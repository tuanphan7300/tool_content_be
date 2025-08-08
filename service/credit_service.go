package service

import (
	"creator-tool-backend/config"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CreditService quản lý credit system
type CreditService struct{}

// NewCreditService tạo instance mới của CreditService
func NewCreditService() *CreditService {
	return &CreditService{}
}

// GetUserCreditBalance lấy số dư credit của user
func (s *CreditService) GetUserCreditBalance(userID uint) (map[string]float64, error) {
	var userCredits config.UserCredits
	err := config.Db.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Tạo record mới nếu chưa có
			userCredits = config.UserCredits{
				UserID:        userID,
				TotalCredits:  0,
				UsedCredits:   0,
				LockedCredits: 0,
			}
			err = config.Db.Create(&userCredits).Error
			if err != nil {
				return nil, fmt.Errorf("failed to create user credits: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user credits: %v", err)
		}
	}

	availableCredits := userCredits.TotalCredits - userCredits.UsedCredits - userCredits.LockedCredits
	if availableCredits < 0 {
		availableCredits = 0
	}

	return map[string]float64{
		"total_credits":     userCredits.TotalCredits,
		"used_credits":      userCredits.UsedCredits,
		"locked_credits":    userCredits.LockedCredits,
		"available_credits": availableCredits,
	}, nil
}

// LockCredits khóa credit trước khi xử lý
func (s *CreditService) LockCredits(userID uint, amount float64, service, description string, videoID *uint) (uint, error) {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Kiểm tra và cập nhật credit
	var userCredits config.UserCredits
	err := tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			userCredits = config.UserCredits{
				UserID:        userID,
				TotalCredits:  0,
				UsedCredits:   0,
				LockedCredits: 0,
			}
			err = tx.Create(&userCredits).Error
			if err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("failed to create user credits: %v", err)
			}
		} else {
			tx.Rollback()
			return 0, fmt.Errorf("failed to get user credits: %v", err)
		}
	}

	// Kiểm tra đủ credit không
	availableCredits := userCredits.TotalCredits - userCredits.UsedCredits - userCredits.LockedCredits
	if availableCredits < amount {
		tx.Rollback()
		return 0, fmt.Errorf("insufficient credits: available %.2f, required %.2f", availableCredits, amount)
	}

	// Cập nhật locked credits
	err = tx.Model(&userCredits).Update("locked_credits", userCredits.LockedCredits+amount).Error
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to lock credits: %v", err)
	}

	// Tạo transaction record
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "lock",
		Amount:            amount,
		Service:           service,
		Description:       description,
		VideoID:           videoID,
		TransactionStatus: "completed",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create lock transaction: %v", err)
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return 0, fmt.Errorf("failed to commit lock transaction: %v", err)
	}

	return transaction.ID, nil
}

// UnlockCredits mở khóa credit (khi lỗi hoặc hoàn thành)
func (s *CreditService) UnlockCredits(userID uint, amount float64, service, description string, videoID *uint) error {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Cập nhật locked credits
	var userCredits config.UserCredits
	err := tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get user credits: %v", err)
	}

	newLockedCredits := userCredits.LockedCredits - amount
	if newLockedCredits < 0 {
		newLockedCredits = 0
	}

	err = tx.Model(&userCredits).Update("locked_credits", newLockedCredits).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to unlock credits: %v", err)
	}

	// Tạo transaction record
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "unlock",
		Amount:            amount,
		Service:           service,
		Description:       description,
		VideoID:           videoID,
		TransactionStatus: "completed",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create unlock transaction: %v", err)
	}

	return tx.Commit().Error
}

// DeductCredits trừ credit sau khi xử lý thành công (với markup)
func (s *CreditService) DeductCredits(userID uint, baseAmount float64, service, description string, videoID *uint, pricingType string, unitsUsed float64) error {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Tính final amount với markup (chuẩn hóa tên service để áp dụng đúng markup nhóm)
	pricingService := NewPricingService()
	markupService := normalizeServiceForMarkup(service)
	finalAmount, err := pricingService.CalculateUserPrice(baseAmount, markupService, userID)
	if err != nil {
		// Fallback to base amount nếu có lỗi
		finalAmount = baseAmount
		log.Printf("Error calculating markup for %s, using base amount: %v", service, err)
	}

	// Cập nhật used credits và giảm locked credits
	var userCredits config.UserCredits
	err = tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get user credits: %v", err)
	}

	// Giảm locked credits và tăng used credits
	newLockedCredits := userCredits.LockedCredits - finalAmount
	if newLockedCredits < 0 {
		newLockedCredits = 0
	}

	err = tx.Model(&userCredits).Updates(map[string]interface{}{
		"used_credits":   userCredits.UsedCredits + finalAmount,
		"locked_credits": newLockedCredits,
	}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deduct credits: %v", err)
	}

	// Tạo transaction record
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "deduct",
		Amount:            finalAmount,
		BaseAmount:        baseAmount,
		Service:           service,
		Description:       description,
		PricingType:       pricingType,
		UnitsUsed:         unitsUsed,
		VideoID:           videoID,
		TransactionStatus: "completed",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create deduct transaction: %v", err)
	}

	markupAmount := finalAmount - baseAmount
	log.Printf("Deducted %.6f credits (base: %.6f, markup: %.6f) from user %d for %s",
		finalAmount, baseAmount, markupAmount, userID, service)

	return tx.Commit().Error
}

// normalizeServiceForMarkup chuẩn hóa tên service con về nhóm để tính markup đúng
// Ví dụ: "gemini_2.0_flash" -> "gemini", "gpt_4o_mini" -> "gpt"
func normalizeServiceForMarkup(service string) string {
	// Đơn giản hóa theo prefix/phần chứa
	// Giữ nguyên các service cơ bản đã chuẩn hóa
	switch service {
	case "whisper", "tts", "process-video", "create-subtitle", "tiktok-optimizer":
		return service
	}
	// Gom nhóm theo từ khóa
	if strings.Contains(service, "gemini") {
		return "gemini"
	}
	if strings.Contains(service, "gpt") {
		return "gpt"
	}
	// Mặc định trả về chính nó nếu không biết
	return service
}

// AddCredits thêm credit cho user
func (s *CreditService) AddCredits(userID uint, amount float64, description, referenceID string) error {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Tìm hoặc tạo user credits
	var userCredits config.UserCredits
	err := tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			userCredits = config.UserCredits{
				UserID:        userID,
				TotalCredits:  amount,
				UsedCredits:   0,
				LockedCredits: 0,
			}
			err = tx.Create(&userCredits).Error
		} else {
			tx.Rollback()
			return fmt.Errorf("failed to get user credits: %v", err)
		}
	} else {
		// Cập nhật total credits
		err = tx.Model(&userCredits).Update("total_credits", userCredits.TotalCredits+amount).Error
	}

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update user credits: %v", err)
	}

	// Tạo transaction record
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "add",
		Amount:            amount,
		Service:           "topup",
		Description:       description,
		ReferenceID:       referenceID,
		TransactionStatus: "completed",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create add transaction: %v", err)
	}

	return tx.Commit().Error
}

// RefundCredits hoàn tiền khi có lỗi
func (s *CreditService) RefundCredits(userID uint, amount float64, service, description string, videoID *uint) error {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Giảm used credits
	var userCredits config.UserCredits
	err := tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get user credits: %v", err)
	}

	newUsedCredits := userCredits.UsedCredits - amount
	if newUsedCredits < 0 {
		newUsedCredits = 0
	}

	err = tx.Model(&userCredits).Update("used_credits", newUsedCredits).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to refund credits: %v", err)
	}

	// Tạo transaction record
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "refund",
		Amount:            amount,
		Service:           service,
		Description:       description,
		VideoID:           videoID,
		TransactionStatus: "completed",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create refund transaction: %v", err)
	}

	return tx.Commit().Error
}

// GetTransactionHistory lấy lịch sử giao dịch
func (s *CreditService) GetTransactionHistory(userID uint, limit int) ([]config.CreditTransaction, error) {
	var transactions []config.CreditTransaction
	err := config.Db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&transactions).Error

	return transactions, err
}

// EstimateTotalCost ước tính tổng chi phí cho process-video
func (s *CreditService) EstimateTotalCost(durationMinutes float64, transcriptLength int, srtLength int) (map[string]float64, error) {
	pricingService := NewPricingService()
	return pricingService.EstimateProcessVideoCost(durationMinutes, transcriptLength, srtLength)
}
