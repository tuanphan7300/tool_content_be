package main

import (
	"creator-tool-backend/config"
	"creator-tool-backend/router"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"time"
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

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.SetupRoutes(r)
	r.Run(":8080") // chạy ở localhost:8080
}
