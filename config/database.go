package config

import (
	"fmt"
	"log"
	"time"

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
		configg.DB_PASSWORD = "root"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/tool?charset=utf8mb4&parseTime=True&loc=Local",
		configg.DB_USER,
		configg.DB_PASSWORD,
		configg.DB_HOST,
		configg.DB_PORT,
	)

	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}

	//Tự động migrate bảng (tạo bảng nếu chưa có)
	//err = Db.AutoMigrate(&CaptionHistory{}, &Users{}, &UserTokens{}, &TokenTransaction{})
	//if err != nil {
	//	log.Fatalf("Error migrating database: %v", err)
	//}
}

// Cấu trúc CaptionHistory lưu trong DB
type CaptionHistory struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"user_id"`
	VideoFilename   string         `json:"video_filename"`
	Transcript      string         `json:"transcript"`
	Suggestion      string         `json:"suggestion"`
	Segments        datatypes.JSON `json:"segments"`
	SegmentsVi      datatypes.JSON `json:"segments_vi"`
	Timestamps      datatypes.JSON `json:"timestamps"`
	BackgroundMusic string         `json:"background_music"`
	SrtFile         string         `json:"srt_file"`
	OriginalSrtFile string         `json:"original_srt_file"`
	TTSFile         string         `json:"tts_file"`
	MergedVideoFile string         `json:"merged_video_file"`
	CreatedAt       time.Time      `json:"created_at"`
}

type Users struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserTokens lưu số dư token của user
// GORM sẽ tự động tạo bảng user_tokens
// GORM sẽ tự động tạo bảng token_transactions

type UserTokens struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"index"`
	TotalTokens int       `json:"total_tokens"`
	UsedTokens  int       `json:"used_tokens"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TokenTransaction struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"index"`
	Type        string    `json:"type"` // "add" hoặc "deduct"
	Amount      int       `json:"amount"`
	Description string    `json:"description"`
	Service     string    `json:"service"`
	VideoID     *uint     `json:"video_id"`
	CreatedAt   time.Time `json:"created_at"`
}
