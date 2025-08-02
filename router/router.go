package router

import (
	"creator-tool-backend/handler"
	"creator-tool-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Add database middleware to all routes
	r.Use(middleware.DatabaseMiddleware())

	// Public routes
	r.POST("/register", handler.RegisterHandler)
	r.POST("/login", handler.LoginHandler)
	r.GET("/ping", handler.PingPongHandler)

	// Health check routes
	r.GET("/health", handler.HealthCheckHandler)

	// Google OAuth routes
	r.GET("/auth/google/login", handler.GoogleLoginHandler)
	r.GET("/auth/google/callback", handler.GoogleCallbackHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/user/profile", handler.GetUserProfileHandler)
		protected.POST("/upload", handler.UploadHandler)
		protected.POST("/process", middleware.ProcessAnyStatusMiddleware(), handler.ProcessHandler)
		protected.POST("/generate-caption", handler.GenerateCaptionHandler)
		protected.POST("/tiktok-optimize", handler.TikTokOptimizerHandler)
		protected.GET("/caption/:id", handler.CaptionHandler)
		protected.GET("/suggest/:id", handler.SuggestHandler)
		protected.POST("/save-history", handler.SaveHistory)
		protected.GET("/history", handler.GetHistory)
		protected.GET("/history/:id", handler.GetHistoryByID)
		protected.DELETE("/history/:id", handler.DeleteHistory)
		protected.DELETE("/history", handler.DeleteHistories)
		protected.GET("/user/video-count", handler.GetUserVideoCount)
		protected.GET("/user/video-stats", handler.GetUserVideoStats)
		protected.POST("/process-voice", handler.ProcessVoiceHandler)
		protected.POST("/process-background", handler.ProcessBackgroundMusicHandler)
		protected.POST("/process-video", middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("process-video"), handler.ProcessVideoHandler)
		protected.POST("/text-to-speech", handler.TextToSpeechHandler)
		protected.POST("/burn-sub", middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("burn-sub"), handler.BurnSubHandler)
		protected.POST("/create-subtitle", middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("create-subtitle"), handler.CreateSubtitleHandler)

		// Credit endpoints (new system)
		protected.GET("/credit/balance", handler.GetCreditBalance)
		protected.GET("/credit/history", handler.GetCreditHistory)
		//protected.POST("/credit/add", handler.AddCredits)
		protected.POST("/credit/estimate", handler.EstimateCost)

		// Legacy estimate endpoint
		protected.POST("/estimate-cost", handler.EstimateProcessVideoCostHandler)

		// Queue management endpoints
		protected.GET("/queue/status", handler.GetQueueStatus)
		protected.GET("/queue/worker/status", handler.GetWorkerStatus)
		protected.GET("/queue/job/:job_id/status", handler.GetJobStatus)
		protected.GET("/queue/job/:job_id/result", handler.GetJobResult)
		protected.GET("/queue/job/:job_id/wait", handler.WaitForJobCompletion)
		protected.POST("/queue/worker/start", handler.StartWorkerService)
		protected.POST("/queue/worker/stop", handler.StopWorkerService)
	}

	// Payment routes
	payment := r.Group("/")
	payment.Use(middleware.AuthMiddleware())
	{
		payment.POST("/payment/create-order", handler.CreatePaymentOrder)
		payment.GET("/payment/order/:order_code", handler.GetPaymentOrder)
		payment.GET("/payment/orders", handler.GetUserPaymentOrders)
		payment.POST("/payment/order/:order_code/cancel", handler.CancelPaymentOrder)
		payment.GET("/payment/order/:order_code/status", handler.GetPaymentOrderStatus)
	}

	// API v1 group
	apiV1 := r.Group("/api/v1")
	{
		// Sepay webhook (public route - không cần auth)
		apiV1.POST("/webhook/sepay", handler.SepayWebhookHandler)
	}

	// Admin routes
	admin := r.Group("/admin")
	{
		admin.POST("/login", handler.AdminLoginHandler)

		// Protected admin routes
		adminProtected := admin.Group("/")
		adminProtected.Use(middleware.AdminAuthMiddleware())
		{
			adminProtected.GET("/dashboard", handler.AdminDashboardHandler)
			adminProtected.GET("/users", handler.AdminUsersHandler)
			adminProtected.GET("/process-status", handler.AdminProcessStatusHandler)
			adminProtected.POST("/process-status/:id", handler.AdminUpdateProcessStatusHandler)
			adminProtected.GET("/upload-history", handler.AdminUploadHistoryHandler)

			// Service config management
			adminProtected.GET("/service-config", handler.AdminServiceConfigHandler)
			adminProtected.POST("/service-config", handler.AdminAddServiceConfigHandler)
			adminProtected.PUT("/service-config", handler.AdminUpdateServiceConfigHandler)
			adminProtected.DELETE("/service-config/:id", handler.AdminDeleteServiceConfigHandler)

			// Sepay webhook logs
			adminProtected.GET("/sepay/webhook-logs", handler.GetSepayWebhookLogs)
		}
		admin.GET("/payment/email-logs", handler.GetPaymentEmailLogs)
	}

	// Serve static files
	r.Static("/storage", "./storage")
}
