package config

import (
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Db *gorm.DB

func ConnectDatabase() {
	// Cấu hình kết nối MySQL
	configg := InfaConfig{}
	configg.LoadConfig()

	// Set default values if environment variables are not set
	if configg.DB_HOST == "" {
		configg.DB_HOST = "db" // Use service name from docker-compose
	}
	if configg.DB_PORT == "" {
		configg.DB_PORT = "3306"
	}
	if configg.DB_USER == "" {
		configg.DB_USER = "root"
	}
	if configg.DB_PASSWORD == "" {
		configg.DB_PASSWORD = "Root@123"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/tool?charset=utf8mb4&parseTime=True&loc=Local",
		configg.DB_USER,
		configg.DB_PASSWORD,
		configg.DB_HOST,
		configg.DB_PORT,
	)

	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// Cấu hình để tránh lock timeout
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}

	// Cấu hình connection pool và timeout
	sqlDB, err := Db.DB()
	if err != nil {
		log.Fatalf("Error getting underlying sql.DB: %v", err)
	}

	// Cấu hình connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Cấu hình timeout cho lock
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	//Tự động migrate bảng (tạo bảng nếu chưa có)
	//err = Db.AutoMigrate(&CaptionHistory{}, &Users{}, &UserTokens{}, &CreditTransaction{}, &model.Feedback{})
	//if err != nil {
	//	log.Fatalf("Error migrating database: %v", err)
	//}
}

// Cấu trúc CaptionHistory lưu trong DB
type CaptionHistory struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	UserID              uint           `json:"user_id"`
	VideoFilename       string         `json:"video_filename" gorm:"type:varchar(500);index:idx_video_filename,length:255"`
	VideoFilenameOrigin string         `json:"video_filename_origin" gorm:"type:varchar(500)"`
	Transcript          string         `json:"transcript" gorm:"type:text"`
	Suggestion          string         `json:"suggestion" gorm:"type:text"`
	Segments            datatypes.JSON `json:"segments"`
	SegmentsVi          datatypes.JSON `json:"segments_vi"`
	Timestamps          datatypes.JSON `json:"timestamps"`
	BackgroundMusic     string         `json:"background_music" gorm:"type:varchar(500)"`
	SrtFile             string         `json:"srt_file" gorm:"type:varchar(500)"`
	OriginalSrtFile     string         `json:"original_srt_file" gorm:"type:varchar(500)"`
	TTSFile             string         `json:"tts_file" gorm:"type:varchar(500)"`
	MergedVideoFile     string         `json:"merged_video_file" gorm:"type:varchar(500)"`
	ProcessType         string         `json:"process_type" gorm:"type:varchar(64);index"`
	// TikTok Optimizer fields
	HookScore         int            `json:"hook_score" gorm:"type:int;default:0"`
	ViralPotential    int            `json:"viral_potential" gorm:"type:int;default:0"`
	TrendingHashtags  datatypes.JSON `json:"trending_hashtags" gorm:"type:json"`
	SuggestedCaption  string         `json:"suggested_caption" gorm:"type:text"`
	BestPostingTime   string         `json:"best_posting_time" gorm:"type:varchar(64)"`
	OptimizationTips  datatypes.JSON `json:"optimization_tips" gorm:"type:json"`
	EngagementPrompts datatypes.JSON `json:"engagement_prompts" gorm:"type:json"`
	CallToAction      string         `json:"call_to_action" gorm:"type:text"`
	VideoDuration     float64        `json:"video_duration" gorm:"type:decimal(10,2);comment:'Duration in seconds'"`
	DeletedAt         *time.Time     `json:"deleted_at" gorm:"index"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type Users struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	// Google OAuth fields
	GoogleID      string    `json:"google_id" gorm:"index"`
	Name          string    `json:"name"`
	Picture       string    `json:"picture"`
	EmailVerified bool      `json:"email_verified"`
	AuthProvider  string    `json:"auth_provider"` // "local" or "google"
	CreatedAt     time.Time `json:"created_at"`
}

// UserTokens lưu số dư token của user
// GORM sẽ tự động tạo bảng user_tokens
// GORM sẽ tự động tạo bảng token_transactions

// ServicePricing lưu giá các service API
type ServicePricing struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ServiceName  string    `json:"service_name" gorm:"uniqueIndex"`
	PricingType  string    `json:"pricing_type" gorm:"type:enum('per_minute','per_token','per_character')"`
	PricePerUnit float64   `json:"price_per_unit" gorm:"type:decimal(10,6)"`
	Currency     string    `json:"currency" gorm:"default:'USD'"`
	Description  string    `json:"description"`
	ModelAPIName string    `json:"model_api_name" gorm:"type:varchar(100);default:null"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ServicePricing) TableName() string {
	return "service_pricings"
}

// PricingTier lưu thông tin các tier pricing
type PricingTier struct {
	ID                int       `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name" gorm:"uniqueIndex"`
	BaseMarkup        float64   `json:"base_markup" gorm:"type:decimal(5,2)"`
	MonthlyLimit      *int      `json:"monthly_limit"`
	SubscriptionPrice float64   `json:"subscription_price" gorm:"type:decimal(10,2);default:0.00"`
	IsActive          bool      `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ServiceMarkup lưu markup cho từng service
type ServiceMarkup struct {
	ServiceName   string    `json:"service_name" gorm:"primaryKey"`
	BaseMarkup    float64   `json:"base_markup" gorm:"type:decimal(5,2)"`
	PremiumMarkup float64   `json:"premium_markup" gorm:"type:decimal(5,2);default:0.00"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserCredits lưu credit của user (USD)
type UserCredits struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"uniqueIndex"`
	TotalCredits  float64   `json:"total_credits" gorm:"type:decimal(12,6);default:0.000000"`
	UsedCredits   float64   `json:"used_credits" gorm:"type:decimal(12,6);default:0.000000"`
	LockedCredits float64   `json:"locked_credits" gorm:"type:decimal(12,6);default:0.000000"`
	TierID        int       `json:"tier_id" gorm:"default:1"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreditTransaction lưu lịch sử giao dịch credit
type CreditTransaction struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UserID            uint      `json:"user_id" gorm:"index"`
	TransactionType   string    `json:"transaction_type" gorm:"type:enum('add','deduct','lock','unlock','refund')"`
	Amount            float64   `json:"amount" gorm:"type:decimal(12,6);default:0.000000"`
	BaseAmount        float64   `json:"base_amount" gorm:"type:decimal(12,6);default:0.000000"`
	Service           string    `json:"service"`
	Description       string    `json:"description"`
	PricingType       string    `json:"pricing_type"`
	UnitsUsed         float64   `json:"units_used" gorm:"type:decimal(12,6);default:0.000000"`
	VideoID           *uint     `json:"video_id"`
	TransactionStatus string    `json:"transaction_status" gorm:"type:enum('pending','completed','failed','refunded');default:'completed'"`
	ReferenceID       string    `json:"reference_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UserTokens lưu số dư token của user (backward compatibility)
type UserTokens struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"index"`
	TotalTokens int       `json:"total_tokens"`
	UsedTokens  int       `json:"used_tokens"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserProcessStatus theo dõi trạng thái process của user để tránh spam
type UserProcessStatus struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      uint       `json:"user_id" gorm:"index"`
	Status      string     `json:"status" gorm:"type:enum('processing','completed','failed','cancelled');default:'processing'"`
	ProcessType string     `json:"process_type" gorm:"type:enum('process','process-video','process-voice','process-background','tiktok-optimize','create-subtitle')"`
	StartedAt   time.Time  `json:"started_at" gorm:"default:CURRENT_TIMESTAMP"`
	CompletedAt *time.Time `json:"completed_at"`
	VideoID     *uint      `json:"video_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (UserProcessStatus) TableName() string {
	return "user_process_status"
}

// ServiceConfig lưu cấu hình dịch vụ cho từng nghiệp vụ
type ServiceConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ServiceType string    `json:"service_type" gorm:"type:varchar(64);index:idx_service_type_active"`
	ServiceName string    `json:"service_name" gorm:"type:varchar(64)"`
	IsActive    bool      `json:"is_active" gorm:"default:true;index:idx_service_type_active"`
	ConfigJSON  string    `json:"config_json" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ServiceConfig) TableName() string {
	return "service_config"
}

// AdminUser represents admin user structure
type AdminUser struct {
	ID            int            `json:"id" gorm:"primaryKey"`
	Username      string         `json:"username" gorm:"uniqueIndex"`
	PasswordHash  string         `json:"password_hash"`
	Email         string         `json:"email" gorm:"uniqueIndex"`
	Name          string         `json:"name"`
	Role          string         `json:"role" gorm:"type:enum('super_admin','admin','moderator');default:'admin'"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	Permissions   datatypes.JSON `json:"permissions" gorm:"type:json"`
	LoginAttempts int            `json:"login_attempts" gorm:"default:0"`
	LockedUntil   *time.Time     `json:"locked_until"`
	LastLogin     *time.Time     `json:"last_login"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

// PaymentOrder lưu thông tin đơn hàng thanh toán
type PaymentOrder struct {
	ID                uint            `json:"id" gorm:"primaryKey"`
	UserID            uint            `json:"user_id" gorm:"index"`
	OrderCode         string          `json:"order_code" gorm:"uniqueIndex;size:50"`
	AmountVND         decimal.Decimal `json:"amount_vnd" gorm:"type:decimal(12,0)"`
	AmountUSD         decimal.Decimal `json:"amount_usd" gorm:"type:decimal(10,2)"`
	ExchangeRate      decimal.Decimal `json:"exchange_rate" gorm:"type:decimal(10,4)"`
	BankAccount       string          `json:"bank_account" gorm:"size:50"`
	BankName          string          `json:"bank_name" gorm:"size:100"`
	QRCodeURL         *string         `json:"qr_code_url" gorm:"size:500"`
	QRCodeData        *string         `json:"qr_code_data" gorm:"type:text"`
	OrderStatus       string          `json:"order_status" gorm:"type:enum('pending','paid','expired','cancelled');default:'pending'"`
	PaymentMethod     string          `json:"payment_method" gorm:"type:enum('qr_code','bank_transfer');default:'qr_code'"`
	ExpiresAt         time.Time       `json:"expires_at"`
	PaidAt            *time.Time      `json:"paid_at"`
	TransactionID     *string         `json:"transaction_id" gorm:"size:100"`
	EmailConfirmation *string         `json:"email_confirmation" gorm:"type:text"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// BankAccount lưu thông tin tài khoản ngân hàng
type BankAccount struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	BankName      string          `json:"bank_name" gorm:"size:100"`
	AccountNumber string          `json:"account_number" gorm:"size:50;uniqueIndex"`
	AccountName   string          `json:"account_name" gorm:"size:200"`
	BankCode      string          `json:"bank_code" gorm:"size:20"`
	CardBin       string          `json:"card_bin" gorm:"size:20"`
	IsActive      bool            `json:"is_active" gorm:"default:true"`
	DailyLimit    decimal.Decimal `json:"daily_limit" gorm:"type:decimal(12,0);default:100000000"`
	MonthlyLimit  decimal.Decimal `json:"monthly_limit" gorm:"type:decimal(12,0);default:1000000000"`
	CreatedAt     time.Time       `json:"created_at"`
}

// EmailTemplate lưu template mail xác nhận thanh toán
type EmailTemplate struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	BankName       string    `json:"bank_name" gorm:"size:100"`
	TemplateName   string    `json:"template_name" gorm:"size:100"`
	SubjectPattern *string   `json:"subject_pattern" gorm:"size:200"`
	SenderPattern  *string   `json:"sender_pattern" gorm:"size:200"`
	AmountPattern  *string   `json:"amount_pattern" gorm:"size:100"`
	AccountPattern *string   `json:"account_pattern" gorm:"size:100"`
	ContentPattern *string   `json:"content_pattern" gorm:"type:text"`
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time `json:"created_at"`
}

// PaymentLog lưu log thanh toán
type PaymentLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	OrderID   uint      `json:"order_id" gorm:"index"`
	LogType   string    `json:"log_type" gorm:"type:enum('order_created','qr_generated','payment_detected','payment_confirmed','payment_failed')"`
	Message   string    `json:"message" gorm:"type:text"`
	Metadata  *string   `json:"metadata" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`
}

// PaymentEmailLog lưu log các email thanh toán đã đọc từ IMAP
// Status: matched, unmatched, error
// ErrorMessage: nếu có lỗi khi xử lý
// TransactionID: nếu parse được
// OrderID: nếu match được đơn hàng
// EmailContent: nội dung mail
// Sender: địa chỉ gửi
// Subject: tiêu đề mail
// ReceivedAt: thời gian nhận mail
// CreatedAt: thời gian lưu log

type PaymentEmailLog struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	OrderCode     string    `json:"order_code"`
	OrderID       *uint     `json:"order_id"`
	Amount        string    `json:"amount"`
	BankAccount   string    `json:"bank_account"`
	TransactionID string    `json:"transaction_id"`
	Sender        string    `json:"sender"`
	Subject       string    `json:"subject"`
	EmailContent  string    `json:"email_content" gorm:"type:text"`
	Status        string    `json:"status" gorm:"type:enum('matched','unmatched','error');default:'unmatched'"`
	ErrorMessage  string    `json:"error_message" gorm:"type:text"`
	ReceivedAt    time.Time `json:"received_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// SepayWebhookLog lưu log tất cả webhook từ Sepay
// ProcessingStatus: received, validated, processed, failed, ignored
// RawPayload: toàn bộ payload JSON gốc
// Headers: headers của request
// IPAddress: IP của Sepay
// UserAgent: User-Agent của request
// ProcessingTimeMs: thời gian xử lý tính bằng milliseconds
// ErrorMessage: nếu có lỗi khi xử lý

type SepayWebhookLog struct {
	ID               uint             `json:"id" gorm:"primaryKey"`
	OrderCode        *string          `json:"order_code" gorm:"size:50"`
	Amount           *decimal.Decimal `json:"amount" gorm:"type:decimal(12,0)"`
	Status           *string          `json:"status" gorm:"size:50"`
	TransactionID    *string          `json:"transaction_id" gorm:"size:100"`
	Signature        *string          `json:"signature" gorm:"size:255"`
	Timestamp        *int64           `json:"timestamp"`
	RawPayload       datatypes.JSON   `json:"raw_payload" gorm:"type:json"`
	Headers          datatypes.JSON   `json:"headers" gorm:"type:json"`
	IPAddress        *string          `json:"ip_address" gorm:"size:45"`
	UserAgent        *string          `json:"user_agent" gorm:"size:500"`
	ProcessingStatus string           `json:"processing_status" gorm:"type:enum('received','validated','processed','failed','ignored');default:'received'"`
	ErrorMessage     *string          `json:"error_message" gorm:"type:text"`
	ProcessingTimeMs *int             `json:"processing_time_ms"`
	CreatedAt        time.Time        `json:"created_at"`
}
