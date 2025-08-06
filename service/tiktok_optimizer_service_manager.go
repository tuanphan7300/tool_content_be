package service

import (
	"creator-tool-backend/config"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// TikTokServiceManager quản lý TikTok Optimizer với service_config và tính phí
type TikTokServiceManager struct {
	db *gorm.DB
}

// NewTikTokServiceManager tạo instance mới
func NewTikTokServiceManager(db *gorm.DB) *TikTokServiceManager {
	return &TikTokServiceManager{
		db: db,
	}
}

// TikTokServiceConfig cấu hình dịch vụ TikTok Optimizer
type TikTokServiceConfig struct {
	ServiceType string `json:"service_type"`
	ServiceName string `json:"service_name"`
	IsActive    bool   `json:"is_active"`
	ConfigJSON  string `json:"config_json"`
}

// TikTokConfig chi tiết cấu hình TikTok
type TikTokConfig struct {
	UseAI              bool     `json:"use_ai"`              // Bật/tắt AI
	SupportedLanguages []string `json:"supported_languages"` // Danh sách ngôn ngữ hỗ trợ
	AICostMultiplier   float64  `json:"ai_cost_multiplier"`  // Hệ số nhân chi phí AI
	MaxTokensPerCall   int      `json:"max_tokens_per_call"` // Số token tối đa mỗi lần gọi
}

// GetTikTokServiceConfig lấy cấu hình dịch vụ TikTok Optimizer
func (t *TikTokServiceManager) GetTikTokServiceConfig() (*TikTokServiceConfig, error) {
	var serviceConfig config.ServiceConfig
	err := t.db.Where("service_type = ? AND is_active = ?", "tiktok-optimizer", true).First(&serviceConfig).Error
	if err != nil {
		log.Printf("TikTok Optimizer service config not found %s", err)
		return &TikTokServiceConfig{}, err
	}

	return &TikTokServiceConfig{
		ServiceType: serviceConfig.ServiceType,
		ServiceName: serviceConfig.ServiceName,
		IsActive:    serviceConfig.IsActive,
		ConfigJSON:  serviceConfig.ConfigJSON,
	}, nil
}

// ParseTikTokConfig parse cấu hình JSON thành struct
func (t *TikTokServiceManager) ParseTikTokConfig(configJSON string) (*TikTokConfig, error) {
	var config TikTokConfig
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TikTok config: %v", err)
	}

	// Set defaults nếu không có
	if len(config.SupportedLanguages) == 0 {
		config.SupportedLanguages = []string{"vi", "en"}
	}
	if config.AICostMultiplier == 0 {
		config.AICostMultiplier = 1.0 // Không có multiplier vì chỉ dùng AI
	}
	if config.MaxTokensPerCall == 0 {
		config.MaxTokensPerCall = 2000
	}

	return &config, nil
}

// IsLanguageSupported kiểm tra xem ngôn ngữ có được hỗ trợ không
func (t *TikTokServiceManager) IsLanguageSupported(language string, config *TikTokConfig) bool {
	for _, lang := range config.SupportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// CalculateTikTokCost tính chi phí cho TikTok Optimizer
func (t *TikTokServiceManager) CalculateTikTokCost(transcript string, targetLanguage string, tikTokConfig *TikTokConfig) (float64, string, error) {
	// Lấy pricing cho TikTok Optimizer
	var pricing config.ServicePricing
	err := t.db.Where("service_name = ? AND is_active = ?", "tiktok-optimizer", true).First(&pricing).Error
	if err != nil {
		// Fallback: Sử dụng default pricing nếu không tìm thấy
		log.Printf("TikTok Optimizer pricing not found, using default pricing: %v", err)
		pricing = config.ServicePricing{
			ServiceName:  "tiktok-optimizer",
			PricingType:  "per_token",
			PricePerUnit: 0.0001,
			Currency:     "USD",
			Description:  "TikTok Optimizer per token (default)",
			ModelAPIName: "gpt-4",
			IsActive:     true,
		}
	}

	// Tính số token dựa trên transcript
	tokenCount := float64(len(strings.Split(transcript, " "))) * 1.3 // Ước tính token

	// Giới hạn token theo config
	if tokenCount > float64(tikTokConfig.MaxTokensPerCall) {
		tokenCount = float64(tikTokConfig.MaxTokensPerCall)
	}

	// Tính cost cơ bản
	baseCost := float64(tokenCount) * pricing.PricePerUnit

	// Áp dụng hệ số AI
	if tikTokConfig.UseAI {
		baseCost *= tikTokConfig.AICostMultiplier
	}

	return baseCost, pricing.PricingType, nil
}

// CreateOptimizer tạo optimizer dựa trên cấu hình
func (t *TikTokServiceManager) CreateOptimizer(apiKey string) *AITikTokOptimizer {
	return NewAITikTokOptimizer(apiKey)
}

// GenerateOptimizedContentWithConfig tạo nội dung tối ưu với cấu hình đầy đủ
func (t *TikTokServiceManager) GenerateOptimizedContentWithConfig(transcript, category, targetLanguage string, duration float64, apiKey string) (*LocalizedTikTokContent, error) {
	// Lấy cấu hình dịch vụ
	serviceConfig, err := t.GetTikTokServiceConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %v", err)
	}
	_ = serviceConfig
	// Tạo optimizer (chỉ dùng AI)
	optimizer := t.CreateOptimizer(apiKey)

	// Generate content bằng AI
	content, err := optimizer.GenerateLocalizedContent(transcript, category, targetLanguage, duration)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %v", err)
	}

	return content, nil
}

// GetServiceStatus trả về trạng thái dịch vụ
func (t *TikTokServiceManager) GetServiceStatus() (bool, string, error) {
	serviceConfig, err := t.GetTikTokServiceConfig()
	if err != nil {
		return false, "Service not found", err
	}

	if !serviceConfig.IsActive {
		return false, "Service is inactive", nil
	}

	return true, "Service is active", nil
}

// LogUsage log việc sử dụng dịch vụ
func (t *TikTokServiceManager) LogUsage(userID uint, cost float64, language string, method string, videoID *uint) error {
	// Tạo credit transaction
	transaction := config.CreditTransaction{
		UserID:            userID,
		TransactionType:   "deduct",
		Amount:            cost,
		Service:           "tiktok-optimizer",
		Description:       fmt.Sprintf("TikTok Optimizer - %s (%s)", language, method),
		PricingType:       "per_token",
		UnitsUsed:         cost, // Sẽ được tính toán chính xác hơn
		VideoID:           videoID,
		TransactionStatus: "completed",
		ReferenceID:       fmt.Sprintf("tiktok_%d_%s", userID, method),
	}

	return t.db.Create(&transaction).Error
}
