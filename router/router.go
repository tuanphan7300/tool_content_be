package router

import (
	"creator-tool-backend/config"
	"creator-tool-backend/handler"
	"creator-tool-backend/middleware"
	"creator-tool-backend/service"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Add database middleware to all routes
	r.Use(middleware.DatabaseMiddleware())

	// Initialize services
	db := config.Db
	feedbackService := service.NewFeedbackService(db)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)

	// Public routes
	r.POST("/register", handler.RegisterHandler)
	r.POST("/login", handler.LoginHandler)
	r.GET("/ping", handler.PingPongHandler)

	// Test voice cache (public)
	r.GET("/test-voices", handler.TestVoiceCacheHandler)

	// Serve voice preview audio files
	r.Static("/voice-preview", "./storage/voice_preview")

	// Serve voice samples
	r.Static("/voice-samples", "./storage/voice_samples")

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
		protected.POST("/tiktok-optimize", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), handler.TikTokOptimizerHandler)
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
		protected.POST("/process-video-async", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("process-video"), handler.ProcessVideoAsyncHandler)
		protected.GET("/process/:process_id/progress", handler.GetProcessingProgressHandler)

		// Optimized TTS endpoints
		protected.POST("/optimized-tts", handler.OptimizedTTSHandler)
		protected.GET("/optimized-tts/:job_id/progress", handler.GetOptimizedTTSProgress)
		protected.GET("/optimized-tts/:job_id/result", handler.GetOptimizedTTSResult)
		protected.DELETE("/optimized-tts/:job_id", handler.CancelOptimizedTTSJob)
		protected.GET("/optimized-tts/stats", handler.GetOptimizedTTSStatistics)

		// Voice selection endpoints
		protected.GET("/voices", handler.GetAvailableVoicesHandler)
		protected.POST("/voice-preview", handler.VoicePreviewHandler)
		protected.POST("/voices/refresh", handler.RefreshVoiceSamplesHandler)
		protected.POST("/burn-sub", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("burn-sub"), handler.BurnSubHandler)
		protected.POST("/create-subtitle", middleware.FileValidationMiddleware(), middleware.ProcessAnyStatusMiddleware(), middleware.ProcessStatusMiddleware("create-subtitle"), handler.CreateSubtitleHandler)

		// Credit endpoints (new system)
		protected.GET("/credit/balance", handler.GetCreditBalance)
		protected.GET("/credit/history", handler.GetCreditHistory)
		//protected.POST("/credit/add", handler.AddCredits)
		protected.POST("/credit/estimate", handler.EstimateCost)

		// Legacy estimate endpoint
		protected.POST("/estimate-cost", handler.EstimateProcessVideoCostHandler)

		// Feedback endpoints
		protected.POST("/feedback", feedbackHandler.CreateFeedback)
		protected.GET("/feedback", feedbackHandler.GetUserFeedbacks)
		protected.GET("/feedback/:id", feedbackHandler.GetFeedbackByID)
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

			// Pricing tiers management
			adminProtected.GET("/pricing-tiers", handler.AdminPricingTiersHandler)
			adminProtected.POST("/pricing-tiers", handler.AdminAddPricingTierHandler)
			adminProtected.PUT("/pricing-tiers", handler.AdminUpdatePricingTierHandler)
			adminProtected.DELETE("/pricing-tiers/:id", handler.AdminDeletePricingTierHandler)

			// Service markups management
			adminProtected.GET("/service-markups", handler.AdminServiceMarkupsHandler)
			adminProtected.POST("/service-markups", handler.AdminAddServiceMarkupHandler)
			adminProtected.PUT("/service-markups", handler.AdminUpdateServiceMarkupHandler)
			adminProtected.DELETE("/service-markups/:service_name", handler.AdminDeleteServiceMarkupHandler)

			// Sepay webhook logs
			adminProtected.GET("/sepay/webhook-logs", handler.GetSepayWebhookLogs)

			// Payment management
			adminProtected.GET("/payments", handler.GetAdminPaymentOrders)
			adminProtected.GET("/payments/stats", handler.GetAdminPaymentStats)
			adminProtected.POST("/payments/:id/cancel", handler.CancelAdminPaymentOrder)

			// Feedback management
			adminProtected.GET("/feedbacks", feedbackHandler.GetAllFeedbacks)
			adminProtected.PUT("/feedbacks/:id", feedbackHandler.UpdateFeedback)
		}
		admin.GET("/payment/email-logs", handler.GetPaymentEmailLogs)
		admin.GET("/credit-usage", handler.AdminCreditUsageListHandler)
		admin.GET("/credit-usage/:video_id", handler.AdminCreditUsageDetailHandler)
	}

	// Download endpoint với authentication
	protected.GET("/api/download/*filepath", handler.DownloadFileHandler)

	// Serve static files (fallback cho development)
	r.Static("/storage", "./storage")
}
