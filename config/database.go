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
var err error

func ConnectDatabase() {
	// Cấu hình kết nối MySQL
	configg := InfaConfig{}
	configg.LoadConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/tool?charset=utf8mb4&parseTime=True&loc=Local",
		configg.DB_USER,
		configg.DB_PASSWORD,
		configg.DB_HOST,
		configg.DB_PORT,
	)

	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}

	//Tự động migrate bảng (tạo bảng nếu chưa có)
	err := Db.AutoMigrate(&CaptionHistory{})
	if err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}
}

// Cấu trúc CaptionHistory lưu trong DB
type CaptionHistory struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	VideoFilename string         `json:"video_filename"`
	Transcript    string         `json:"transcript"`
	Suggestion    string         `json:"suggestion"`
	Segments      datatypes.JSON `json:"segments"`
	SegmentsVi    datatypes.JSON `json:"segments_vi"`
	Timestamps    datatypes.JSON `json:"timestamps"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Users struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}
