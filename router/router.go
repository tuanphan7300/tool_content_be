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
		protected.POST("/tiktok-optimize", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), handler.TikTokOptimizerHandler)
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
		protected.POST("/process-video", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("process-video"), handler.ProcessVideoHandler)
		protected.POST("/process-video-parallel", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("process-video"), handler.ProcessVideoParallelHandler)
		protected.GET("/process/:process_id/progress", handler.GetProcessingProgressHandler)

		// Cache management endpoints
		protected.GET("/cache/stats", handler.GetCacheStatsHandler)
		protected.POST("/cache/cleanup", handler.CleanupCacheHandler)
		protected.DELETE("/cache/entry/:key", handler.DeleteCacheEntryHandler)
		protected.GET("/cache/entry/:key", handler.GetCacheEntryHandler)
		protected.DELETE("/cache/all", handler.ClearAllCacheHandler)
		protected.PUT("/cache/entry/:key/ttl", handler.SetCacheTTLHandler)
		protected.POST("/text-to-speech", handler.TextToSpeechHandler)
		protected.POST("/burn-sub", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("burn-sub"), handler.BurnSubHandler)
		protected.POST("/create-subtitle", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("create-subtitle"), handler.CreateSubtitleHandler)

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
	apiV1 := r.Group("/v1")
	{
		// Sepay webhook (cần xác thực API Key)
		apiV1.POST("/webhook/sepay", middleware.SepayAuthMiddleware(), handler.SepayWebhookHandler)
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

			// Payment management
			adminProtected.GET("/payments", handler.GetAdminPaymentOrders)
			adminProtected.GET("/payments/stats", handler.GetAdminPaymentStats)
			adminProtected.POST("/payments/:id/cancel", handler.CancelAdminPaymentOrder)
		}
		admin.GET("/payment/email-logs", handler.GetPaymentEmailLogs)
	}

	// Serve static files
	r.Static("/storage", "./storage")
}
