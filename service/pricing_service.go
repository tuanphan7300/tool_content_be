package service

import (
	"creator-tool-backend/config"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PricingService quản lý tính toán chi phí theo tài liệu chính thức
type PricingService struct{}

// NewPricingService tạo instance mới của PricingService
func NewPricingService() *PricingService {
	return &PricingService{}
}

// CalculateWhisperCost tính chi phí Whisper theo thời gian audio (phút)
func (s *PricingService) CalculateWhisperCost(durationMinutes float64) (float64, error) {
	// Lấy service name từ service_config
	serviceName, _, err := s.GetActiveServiceForType("speech_to_text")
	if err != nil {
		return 0, fmt.Errorf("no active speech-to-text service: %v", err)
	}

	pricing, err := s.getServicePricing(serviceName)
	if err != nil {
		return 0, err
	}

	cost := durationMinutes * pricing.PricePerUnit
	return cost, nil
}

// CalculateLLMCost tính chi phí mô hình LLM (Gemini/GPT) theo số token thực tế, trả về cả model_api_name
func (s *PricingService) CalculateLLMCost(text string, serviceName string) (float64, int, string, error) {
	pricing, err := s.getServicePricing(serviceName)
	if err != nil {
		return 0, 0, "", err
	}
	modelAPIName := pricing.ModelAPIName

	// Tính tokens với giới hạn tối đa để tránh overflow
	textLength := len([]rune(text))
	tokens := textLength / 4
	if tokens < 1 {
		tokens = 1
	}

	// Log để debug
	log.Printf("CalculateLLMCost: text_length=%d, tokens=%d, service_name=%s", textLength, tokens, serviceName)

	// Giới hạn tokens tối đa để tránh lỗi database
	const maxTokens = 1000000 // 1 triệu tokens tối đa
	if tokens > maxTokens {
		log.Printf("WARNING: tokens %d exceeds max %d, capping to max", tokens, maxTokens)
		tokens = maxTokens
	}

	cost := float64(tokens) * pricing.PricePerUnit
	return cost, tokens, modelAPIName, nil
}

// getServicePricingByPreferredNames thử lấy pricing theo danh sách tên ưu tiên
func (s *PricingService) getServicePricingByPreferredNames(preferredNames []string) (*config.ServicePricing, error) {
	var lastErr error
	for _, name := range preferredNames {
		pricing, err := s.getServicePricing(name)
		if err == nil {
			return pricing, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no pricing found for preferred names")
}

// CalculateLLMCostSplit tính chi phí input và output riêng cho LLM theo token
//   - baseServiceName: ví dụ "gemini_2.0_flash" hoặc "gpt_3.5_turbo"
//   - Quy ước DB: tạo thêm 2 bản ghi pricing với suffix "_input" và "_output"
//     Nếu không có, hàm sẽ fallback về baseServiceName để tránh lỗi (tính như input)
func (s *PricingService) CalculateLLMCostSplit(inputText, outputText, baseServiceName string) (float64, float64, int, int, string, error) {
	// Lấy model_api_name từ bản ghi base (nếu có)
	var modelAPIName string
	if basePricing, err := s.getServicePricing(baseServiceName); err == nil {
		modelAPIName = basePricing.ModelAPIName
	}

	// Lấy pricing cho input và output
	inputPricing, err := s.getServicePricingByPreferredNames([]string{baseServiceName + "_input", baseServiceName})
	if err != nil {
		return 0, 0, 0, 0, modelAPIName, fmt.Errorf("input pricing not found for %s: %v", baseServiceName, err)
	}
	outputPricing, err := s.getServicePricingByPreferredNames([]string{baseServiceName + "_output", baseServiceName})
	if err != nil {
		// Fallback: nếu thiếu _output và base, dùng lại inputPricing để tránh fail toàn bộ
		outputPricing = inputPricing
	}

	// Ước tính tokens
	inputTokens := len([]rune(inputText)) / 4
	if inputTokens < 1 {
		inputTokens = 1
	}
	outputTokens := len([]rune(outputText)) / 4
	if outputTokens < 1 {
		outputTokens = 1
	}

	inputCost := float64(inputTokens) * inputPricing.PricePerUnit
	outputCost := float64(outputTokens) * outputPricing.PricePerUnit

	return inputCost, outputCost, inputTokens, outputTokens, modelAPIName, nil
}

// CalculateTTSCost tính chi phí TTS theo số ký tự
func (s *PricingService) CalculateTTSCost(text string, useWavenet bool) (float64, error) {
	serviceName, _, err := s.GetActiveServiceForType("text_to_speech")
	if err != nil {
		return 0, fmt.Errorf("no active text-to-speech service: %v", err)
	}

	pricing, err := s.getServicePricing(serviceName)
	if err != nil {
		return 0, err
	}

	characters := len([]rune(text))
	cost := float64(characters) * pricing.PricePerUnit
	return cost, nil
}

// CalculateGPTCost tính chi phí GPT theo số token
func (s *PricingService) CalculateGPTCost(text string) (float64, int, error) {
	pricing, err := s.getServicePricing("gpt_3.5_turbo")
	if err != nil {
		return 0, 0, err
	}

	// GPT-3.5 Turbo: $0.002 per 1K tokens
	// Ước tính: 1 token ≈ 4 ký tự
	tokens := len([]rune(text)) / 4
	if tokens < 1 {
		tokens = 1
	}

	cost := float64(tokens) * pricing.PricePerUnit
	return cost, tokens, nil
}

// getServicePricing lấy thông tin pricing của service
func (s *PricingService) getServicePricing(serviceName string) (*config.ServicePricing, error) {
	var pricing config.ServicePricing
	err := config.Db.Where("service_name = ? AND is_active = ?", serviceName, true).First(&pricing).Error
	if err != nil {
		return nil, fmt.Errorf("service pricing not found for %s: %v", serviceName, err)
	}
	return &pricing, nil
}

// DeductUserCredits trừ credit của user
func (s *PricingService) DeductUserCredits(userID uint, cost float64, service, description string, videoID *uint, pricingType string, unitsUsed float64) error {
	// Bắt đầu transaction
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Kiểm tra credit của user
	var userCredits config.UserCredits
	err := tx.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Tạo record mới nếu chưa có
			userCredits = config.UserCredits{
				UserID:       userID,
				TotalCredits: 0,
				UsedCredits:  0,
			}
			err = tx.Create(&userCredits).Error
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create user credits: %v", err)
			}
		} else {
			tx.Rollback()
			return fmt.Errorf("failed to get user credits: %v", err)
		}
	}

	// Kiểm tra đủ credit không
	availableCredits := userCredits.TotalCredits - userCredits.UsedCredits
	if availableCredits < cost {
		tx.Rollback()
		return fmt.Errorf("insufficient credits: available %.2f, required %.2f", availableCredits, cost)
	}

	// Cập nhật used credits
	err = tx.Model(&userCredits).Update("used_credits", userCredits.UsedCredits+cost).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update user credits: %v", err)
	}

	// Lưu transaction
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "deduct",
		Amount:            cost,
		Service:           service,
		Description:       description,
		PricingType:       pricingType,
		UnitsUsed:         unitsUsed,
		VideoID:           videoID,
		TransactionStatus: "completed",
		ReferenceID:       "",
		CreatedAt:         time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	// Commit transaction
	return tx.Commit().Error
}

// AddUserCredits thêm credit cho user
func (s *PricingService) AddUserCredits(userID uint, amount float64, description string) error {
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
				UserID:       userID,
				TotalCredits: amount,
				UsedCredits:  0,
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

	// Lưu transaction
	transaction := config.CreditTransaction{
		UserID:          userID,
		TransactionType: "add",
		Amount:          amount,
		Service:         "topup",
		Description:     description,
		PricingType:     "credit",
		UnitsUsed:       0,
		CreatedAt:       time.Now(),
	}

	err = tx.Create(&transaction).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	return tx.Commit().Error
}

// GetUserCreditBalance lấy số dư credit của user
func (s *PricingService) GetUserCreditBalance(userID uint) (float64, error) {
	var userCredits config.UserCredits
	err := config.Db.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil // User chưa có credit
		}
		return 0, fmt.Errorf("failed to get user credits: %v", err)
	}

	return userCredits.TotalCredits - userCredits.UsedCredits, nil
}

// GetUserTier lấy thông tin tier của user
func (s *PricingService) GetUserTier(userID uint) (*config.PricingTier, error) {
	var userCredits config.UserCredits
	err := config.Db.Where("user_id = ?", userID).First(&userCredits).Error
	if err != nil {
		return nil, fmt.Errorf("user credits not found: %v", err)
	}

	var tier config.PricingTier
	err = config.Db.Where("id = ? AND is_active = ?", userCredits.TierID, true).First(&tier).Error
	if err != nil {
		return nil, fmt.Errorf("pricing tier not found: %v", err)
	}

	return &tier, nil
}

// GetServiceMarkup lấy markup của service
func (s *PricingService) GetServiceMarkup(serviceName string) (*config.ServiceMarkup, error) {
	var markup config.ServiceMarkup
	err := config.Db.Where("service_name = ?", serviceName).First(&markup).Error
	if err != nil {
		return nil, fmt.Errorf("service markup not found for %s: %v", serviceName, err)
	}

	return &markup, nil
}

// CalculateUserPrice tính giá với markup cho user
func (s *PricingService) CalculateUserPrice(baseCost float64, service string, userID uint) (float64, error) {
	// Lấy user tier
	tier, err := s.GetUserTier(userID)
	if err != nil {
		// Fallback to default free tier
		tier = &config.PricingTier{
			ID:         1,
			Name:       "free",
			BaseMarkup: 20.00,
		}
	}

	// Lấy service markup
	serviceMarkup, err := s.GetServiceMarkup(service)
	if err != nil {
		// Fallback to default markup
		serviceMarkup = &config.ServiceMarkup{
			ServiceName:   service,
			BaseMarkup:    20.00,
			PremiumMarkup: 0.00,
		}
	}

	// Tính total markup
	totalMarkup := tier.BaseMarkup + serviceMarkup.BaseMarkup + serviceMarkup.PremiumMarkup

	// Tính final price
	finalPrice := baseCost * (1 + totalMarkup/100)

	return finalPrice, nil
}

// EstimateProcessVideoCost ước tính chi phí cho process-video
func (s *PricingService) EstimateProcessVideoCost(durationMinutes float64, transcriptLength int, srtLength int) (map[string]float64, error) {
	estimates := make(map[string]float64)

	// Whisper cost
	whisperCost, err := s.CalculateWhisperCost(durationMinutes)
	if err != nil {
		return nil, err
	}
	estimates["whisper"] = whisperCost

	// Lấy service name cho Gemini từ database
	geminiServiceName, _, err := s.GetActiveServiceForType("srt_translation")
	if err != nil {
		return nil, fmt.Errorf("failed to get active Gemini service: %v", err)
	}

	// Gemini/GPT cost (dịch SRT) - ước tính input + output
	// Prompt bao gồm: context analysis + instructions + SRT content (input)
	// Output ước tính gần bằng độ dài SRT đích
	// Context analysis: ~1000 ký tự cho prompt phân tích
	contextAnalysisLength := 1000
	promptLength := contextAnalysisLength + 500 + srtLength // context analysis + instructions + SRT content
	inCost, outCost, _, _, _, err := s.CalculateLLMCostSplit(strings.Repeat("a", promptLength), strings.Repeat("a", srtLength), geminiServiceName)
	if err != nil {
		return nil, err
	}
	gemiCost := inCost + outCost
	estimates["gemini"] = gemiCost

	// TTS cost (sử dụng Wavenet cho chất lượng tốt)
	ttsCost, err := s.CalculateTTSCost(strings.Repeat("a", transcriptLength), true)
	if err != nil {
		return nil, err
	}
	estimates["tts"] = ttsCost

	// Total cost
	total := whisperCost + gemiCost + ttsCost
	estimates["total"] = total

	return estimates, nil
}

// EstimateProcessVideoCostWithMarkup ước tính chi phí với markup cho user
func (s *PricingService) EstimateProcessVideoCostWithMarkup(durationMinutes float64, transcriptLength int, srtLength int, userID uint) (map[string]float64, error) {
	estimates := make(map[string]float64)

	// Whisper cost
	whisperCost, err := s.CalculateWhisperCost(durationMinutes)
	if err != nil {
		return nil, err
	}
	whisperPrice, err := s.CalculateUserPrice(whisperCost, "whisper", userID)
	if err != nil {
		return nil, err
	}
	estimates["whisper"] = whisperPrice
	estimates["whisper_base"] = whisperCost

	// Lấy service name cho Gemini từ database
	geminiServiceName, _, err := s.GetActiveServiceForType("srt_translation")
	if err != nil {
		return nil, fmt.Errorf("failed to get active Gemini service: %v", err)
	}

	// Gemini/GPT cost (dịch SRT) - ước tính input + output
	// Prompt bao gồm: context analysis + instructions + SRT content (input)
	// Output ước tính gần bằng độ dài SRT đích
	// Context analysis: ~1000 ký tự cho prompt phân tích
	contextAnalysisLength := 1000
	promptLength := contextAnalysisLength + 500 + srtLength // context analysis + instructions + SRT content
	inCost, outCost, _, _, _, err := s.CalculateLLMCostSplit(strings.Repeat("a", promptLength), strings.Repeat("a", srtLength), geminiServiceName)
	if err != nil {
		return nil, err
	}
	gemiCost := inCost + outCost
	geminiPrice, err := s.CalculateUserPrice(gemiCost, "gemini", userID)
	if err != nil {
		return nil, err
	}
	estimates["gemini"] = geminiPrice
	estimates["gemini_base"] = gemiCost

	// TTS cost (sử dụng Wavenet cho chất lượng tốt)
	ttsCost, err := s.CalculateTTSCost(strings.Repeat("a", transcriptLength), true)
	if err != nil {
		return nil, err
	}
	ttsPrice, err := s.CalculateUserPrice(ttsCost, "tts", userID)
	if err != nil {
		return nil, err
	}
	estimates["tts"] = ttsPrice
	estimates["tts_base"] = ttsCost

	// Total cost
	totalBase := whisperCost + gemiCost + ttsCost
	totalPrice := whisperPrice + geminiPrice + ttsPrice

	estimates["total_base"] = totalBase
	estimates["total"] = totalPrice
	estimates["markup_amount"] = totalPrice - totalBase
	estimates["markup_percentage"] = ((totalPrice - totalBase) / totalBase) * 100

	return estimates, nil
}

// GetGeminiModelAPIName lấy model_api_name từ DB theo service_name
func (s *PricingService) GetGeminiModelAPIName(serviceName string) (string, error) {
	var pricing config.ServicePricing
	err := config.Db.Where("service_name = ? AND is_active = ?", serviceName, true).First(&pricing).Error
	if err != nil {
		return "", fmt.Errorf("service pricing not found for %s: %v", serviceName, err)
	}
	if pricing.ModelAPIName != "" {
		return pricing.ModelAPIName, nil
	}
	return "", fmt.Errorf("model_api_name not set for %s", serviceName)
}

// GetActiveGeminiServiceName trả về service_name và model_api_name của Gemini model đang active
func (s *PricingService) GetActiveGeminiServiceName() (string, string, error) {
	var pricing config.ServicePricing
	err := config.Db.Where("service_name LIKE ? AND is_active = ?", "gemini_%", true).Order("id ASC").First(&pricing).Error
	if err != nil {
		return "", "", fmt.Errorf("no active Gemini model found: %v", err)
	}
	return pricing.ServiceName, pricing.ModelAPIName, nil
}

// GetActiveServiceForType trả về service_name và model_api_name của dịch vụ active cho một service_type
func (s *PricingService) GetActiveServiceForType(serviceType string) (string, string, error) {
	// Bước 1: Lấy service_name đang active từ service_config
	var sc config.ServiceConfig
	if err := config.Db.Where("service_type = ? AND is_active = 1", serviceType).Order("service_name").First(&sc).Error; err != nil {
		return "", "", fmt.Errorf("no active service for type %s: %v", serviceType, err)
	}

	// Bước 2: Tìm model_api_name tương ứng trong service_pricings
	// Ưu tiên theo thứ tự: base → _input → _output
	var pricing config.ServicePricing
	// base
	if err := config.Db.Where("service_name = ? AND is_active = 1", sc.ServiceName).First(&pricing).Error; err == nil {
		return sc.ServiceName, pricing.ModelAPIName, nil
	}
	// _input
	if err := config.Db.Where("service_name = ? AND is_active = 1", sc.ServiceName+"_input").First(&pricing).Error; err == nil {
		return sc.ServiceName, pricing.ModelAPIName, nil
	}
	// _output
	if err := config.Db.Where("service_name = ? AND is_active = 1", sc.ServiceName+"_output").First(&pricing).Error; err == nil {
		return sc.ServiceName, pricing.ModelAPIName, nil
	}

	return "", "", fmt.Errorf("pricing not found for active service %s (type %s)", sc.ServiceName, serviceType)
}
