package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// TTSRateLimiter quản lý rate limit cho Google TTS API
type TTSRateLimiter struct {
	redisClient *redis.Client
	rateLimit   int           // 900 requests/phút
	windowSize  time.Duration // 1 phút
	ctx         context.Context
}

// TTSRequestRecord lưu thông tin về mỗi TTS request
type TTSRequestRecord struct {
	Timestamp   int64  `json:"timestamp"`
	UserID      uint   `json:"user_id"`
	SegmentText string `json:"segment_text"`
	RequestID   string `json:"request_id"`
}

var (
	ttsRateLimiter *TTSRateLimiter
	ttsRedisClient *redis.Client
)

// InitTTSRateLimiter khởi tạo TTS Rate Limiter
func InitTTSRateLimiter(redisAddr, redisPassword string) error {
	// Sử dụng Redis client từ queue service nếu có
	if ttsRedisClient == nil {
		ttsRedisClient = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       0,
		})
	}

	ctx := context.Background()

	// Test connection
	_, err := ttsRedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis for TTS rate limiter: %v", err)
	}

	ttsRateLimiter = &TTSRateLimiter{
		redisClient: ttsRedisClient,
		rateLimit:   900,         // 900 requests/phút
		windowSize:  time.Minute, // 1 phút
		ctx:         ctx,
	}

	log.Println("TTS Rate Limiter initialized successfully")
	return nil
}

// GetTTSRateLimiter trả về instance của TTS Rate Limiter
func GetTTSRateLimiter() *TTSRateLimiter {
	return ttsRateLimiter
}

// CanMakeRequest kiểm tra xem có thể gọi TTS API không
func (r *TTSRateLimiter) CanMakeRequest() bool {
	currentTime := time.Now()
	windowStart := currentTime.Add(-r.windowSize)

	// Đếm requests trong window hiện tại
	count, err := r.redisClient.ZCount(
		r.ctx,
		"tts_requests",
		fmt.Sprintf("%d", windowStart.Unix()),
		fmt.Sprintf("%d", currentTime.Unix()),
	).Result()

	if err != nil {
		log.Printf("Error checking TTS rate limit: %v", err)
		return false
	}

	return count < int64(r.rateLimit)
}

// ReserveSlot đặt chỗ cho TTS request
func (r *TTSRateLimiter) ReserveSlot(userID uint, segmentText, requestID string) bool {
	if !r.CanMakeRequest() {
		return false
	}

	currentTime := time.Now()

	// Thêm timestamp vào sorted set
	err := r.redisClient.ZAdd(
		r.ctx,
		"tts_requests",
		&redis.Z{Score: float64(currentTime.Unix()), Member: currentTime.UnixNano()},
	).Err()

	if err != nil {
		log.Printf("Error reserving TTS slot: %v", err)
		return false
	}

	// Set TTL để tự động cleanup
	r.redisClient.Expire(r.ctx, "tts_requests", r.windowSize)

	// Log request
	log.Printf("TTS request reserved for user %d: %s", userID, segmentText[:min(len(segmentText), 50)])

	return true
}

// GetCurrentUsage trả về thông tin sử dụng hiện tại
func (r *TTSRateLimiter) GetCurrentUsage() map[string]interface{} {
	currentTime := time.Now()
	windowStart := currentTime.Add(-r.windowSize)

	// Đếm requests trong window hiện tại
	count, err := r.redisClient.ZCount(
		r.ctx,
		"tts_requests",
		fmt.Sprintf("%d", windowStart.Unix()),
		fmt.Sprintf("%d", currentTime.Unix()),
	).Result()

	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	usagePercentage := float64(count) / float64(r.rateLimit) * 100

	return map[string]interface{}{
		"current_requests": count,
		"max_requests":     r.rateLimit,
		"usage_percentage": usagePercentage,
		"window_start":     windowStart,
		"window_end":       currentTime,
		"remaining":        r.rateLimit - int(count),
	}
}

// WaitForSlot chờ cho đến khi có slot available
func (r *TTSRateLimiter) WaitForSlot(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if r.CanMakeRequest() {
			return true
		}
		<-ticker.C
	}

	return false
}

// CleanupOldRequests xóa các requests cũ
func (r *TTSRateLimiter) CleanupOldRequests() error {
	currentTime := time.Now()
	cutoffTime := currentTime.Add(-r.windowSize)

	// Xóa các requests cũ hơn 1 phút
	removed, err := r.redisClient.ZRemRangeByScore(
		r.ctx,
		"tts_requests",
		"0",
		fmt.Sprintf("%d", cutoffTime.Unix()),
	).Result()

	if err != nil {
		return fmt.Errorf("failed to cleanup old TTS requests: %v", err)
	}

	if removed > 0 {
		log.Printf("Cleaned up %d old TTS requests", removed)
	}

	return nil
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
