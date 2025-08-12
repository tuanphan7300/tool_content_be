package main

import (
	"creator-tool-backend/config"
	"creator-tool-backend/handler"
	"creator-tool-backend/router"
	"creator-tool-backend/service"
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err.Error())
	}
	infaConfig := config.InfaConfig{}
	infaConfig.LoadConfig()
}

func main() {
	infaConfig := config.InfaConfig{}
	infaConfig.LoadConfig()
	config.ConnectDatabase()
	defer func() {
		db, _ := config.Db.DB()
		db.Close() // Đảm bảo đóng kết nối khi ứng dụng dừng
	}()

	// Khởi tạo Google OAuth
	log.Println("Initializing Google OAuth...")
	handler.InitGoogleOAuth()

	// Khởi tạo queue service
	log.Println("Initializing queue service...")
	err := service.InitQueueService()
	if err != nil {
		log.Printf("Failed to initialize queue service: %v", err)
		log.Println("Continuing without queue service...")
	} else {
		// Khởi tạo worker service
		log.Println("Initializing worker service...")
		workerService := service.InitWorkerService(service.GetQueueService())

		// Khởi động worker service
		workerService.Start()
		defer workerService.Stop()
	}

	// Khởi tạo process status service và cleanup routine
	log.Println("Initializing process status service...")
	processStatusService := service.NewProcessStatusService()

	// Khởi tạo TTS Rate Limiter
	log.Println("Initializing TTS rate limiter...")
	err = service.InitTTSRateLimiter("localhost:6379", "")
	if err != nil {
		log.Printf("Warning: Failed to initialize TTS rate limiter: %v", err)
		log.Println("TTS will continue without rate limiting")
	}

	// Khởi tạo TTS Mapping Service
	log.Println("Initializing TTS mapping service...")
	service.InitTTSMappingService()

	// Chạy background cleanup routine
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Cleanup mỗi 5 phút
		defer ticker.Stop()

		log.Println("Starting background cleanup routine for stale processes...")

		for {
			select {
			case <-ticker.C:
				if err := processStatusService.CleanupStaleProcesses(); err != nil {
					log.Printf("Error cleaning up stale processes: %v", err)
				} else {
					log.Println("Background cleanup completed successfully (stale processes)")
				}
				if err := processStatusService.CleanupOldCaptionHistories(); err != nil {
					log.Printf("Error cleaning up old caption histories: %v", err)
				} else {
					log.Println("Background cleanup completed successfully (old caption histories)")
				}
			}
		}
	}()

	// Khởi động cron job kiểm tra đơn hàng hết hạn
	//go func() {
	//	ticker := time.NewTicker(1 * time.Minute)
	//	defer ticker.Stop()
	//
	//	paymentService := service.NewPaymentOrderService()
	//	for {
	//		select {
	//		case <-ticker.C:
	//			if err := paymentService.CheckExpiredOrders(); err != nil {
	//				log.Printf("Failed to check expired orders: %v", err)
	//			}
	//		}
	//	}
	//}()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://inis-hvnh.site", "https://videotool.com.vn", "http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Range", "If-Range"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length", "Content-Range", "Content-Disposition"},
		MaxAge:           12 * time.Hour,
	}))
	router.SetupRoutes(r)
	r.Run(":8888") // chạy ở localhost:8080
}
