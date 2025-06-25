package router

import (
	"creator-tool-backend/handler"
	"creator-tool-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Public routes
	r.POST("/register", handler.RegisterHandler)
	r.POST("/login", handler.LoginHandler)
	r.GET("/ping", handler.PingPongHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/upload", handler.UploadHandler)
		protected.POST("/process", handler.ProcessHandler)
		protected.GET("/caption/:id", handler.CaptionHandler)
		protected.GET("/suggest/:id", handler.SuggestHandler)
		protected.POST("/save-history", handler.SaveHistory)
		protected.GET("/history", handler.GetHistory)
		protected.GET("/history/:id", handler.GetHistoryByID)
		protected.POST("/process-voice", handler.ProcessVoiceHandler)
		protected.POST("/process-background", handler.ProcessBackgroundMusicHandler)
		protected.POST("/process-video", handler.ProcessVideoHandler)
		protected.POST("/text-to-speech", handler.TextToSpeechHandler)
		protected.GET("/token/balance", handler.GetTokenBalance)
		protected.GET("/token/history", handler.GetTokenHistory)
		protected.POST("/token/add", handler.AddToken)
		protected.POST("/token/deduct", handler.DeductToken)
	}

	// Serve static files
	r.Static("/storage", "./storage")
}
