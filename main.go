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

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://inis-hvnh.site", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.SetupRoutes(r)
	r.Run(":8888") // chạy ở localhost:8080
}
