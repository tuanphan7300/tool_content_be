package router

import (
	"creator-tool-backend/handler"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.POST("/upload", handler.UploadHandler)
	r.POST("/process", handler.ProcessHandler)
	r.GET("/caption/:id", handler.CaptionHandler)
	r.GET("/suggest/:id", handler.SuggestHandler)
	r.POST("/save-history", handler.SaveHistory)
	r.GET("/history", handler.GetHistory)
	r.GET("/history/:id", handler.GetHistoryByID)
	r.POST("/register", handler.RegisterHandler)
	r.POST("/login", handler.LoginHandler)
	r.GET("/ping", handler.PingPongHandler)
	r.POST("/process-voice", handler.ProcessVoiceHandler)
	r.POST("/process-background", handler.ProcessBackgroundMusicHandler)
	//r.Use(middleware.AuthMiddleware())
}
