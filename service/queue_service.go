package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type AudioProcessingJob struct {
	ID              string  `json:"id"`
	AudioPath       string  `json:"audio_path"`
	FileName        string  `json:"file_name"`
	VideoDir        string  `json:"video_dir"`
	StemType        string  `json:"stem_type"` // "vocals" or "no_vocals"
	CreatedAt       int64   `json:"created_at"`
	UserID          uint    `json:"user_id"`
	VideoID         uint    `json:"video_id"`
	Priority        int     `json:"priority"`     // 1-10, 10 là cao nhất
	MaxDuration     float64 `json:"max_duration"` // Giới hạn thời gian xử lý
	JobType         string  `json:"job_type"`     // "demucs", "burn-sub"...
	SubtitlePath    string  `json:"subtitle_path"`
	OutputPath      string  `json:"output_path"`
	SubtitleColor   string  `json:"subtitle_color"`
	SubtitleBgColor string  `json:"subtitle_bgcolor"`
}

type QueueService struct {
	redisClient *redis.Client
	ctx         context.Context
}

var (
	queueService *QueueService
	redisClient  *redis.Client
)

// InitQueueService khởi tạo Redis connection và queue service
func InitQueueService() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Có thể config từ env
		Password: "",               // Có thể config từ env
		DB:       0,
	})

	ctx := context.Background()

	// Test connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	queueService = &QueueService{
		redisClient: redisClient,
		ctx:         ctx,
	}

	log.Println("Queue service initialized successfully")
	return nil
}

// GetQueueService trả về instance của queue service
func GetQueueService() *QueueService {
	return queueService
}

// EnqueueJob thêm job vào queue
func (qs *QueueService) EnqueueJob(job *AudioProcessingJob) error {
	job.CreatedAt = time.Now().Unix()

	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %v", err)
	}

	// Thêm vào queue với priority
	queueKey := fmt.Sprintf("audio_processing_queue:%d", job.Priority)
	err = qs.redisClient.LPush(qs.ctx, queueKey, jobData).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %v", err)
	}

	log.Printf("Job enqueued: %s, priority: %d", job.ID, job.Priority)
	return nil
}

// DequeueJob lấy job từ queue
func (qs *QueueService) DequeueJob() (*AudioProcessingJob, error) {
	// Lấy từ queue priority cao nhất trước
	for priority := 10; priority >= 1; priority-- {
		queueKey := fmt.Sprintf("audio_processing_queue:%d", priority)

		// Lấy job từ queue (blocking với timeout 1s)
		result, err := qs.redisClient.BRPop(qs.ctx, time.Second, queueKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue // Queue này empty, thử queue khác
			}
			return nil, fmt.Errorf("failed to dequeue job: %v", err)
		}

		if len(result) >= 2 {
			var job AudioProcessingJob
			err = json.Unmarshal([]byte(result[1]), &job)
			if err != nil {
				log.Printf("Failed to unmarshal job: %v", err)
				continue
			}
			return &job, nil
		}
	}

	return nil, nil // Không có job nào
}

// GetQueueStatus trả về thông tin về queue
func (qs *QueueService) GetQueueStatus() (map[string]int64, error) {
	status := make(map[string]int64)

	for priority := 1; priority <= 10; priority++ {
		queueKey := fmt.Sprintf("audio_processing_queue:%d", priority)
		count, err := qs.redisClient.LLen(qs.ctx, queueKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get queue length: %v", err)
		}
		status[fmt.Sprintf("priority_%d", priority)] = count
	}

	return status, nil
}

// GetJobStatus trả về trạng thái của job
func (qs *QueueService) GetJobStatus(jobID string) (string, error) {
	status, err := qs.redisClient.Get(qs.ctx, fmt.Sprintf("job_status:%s", jobID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "not_found", nil
		}
		return "", fmt.Errorf("failed to get job status: %v", err)
	}
	return status, nil
}

// UpdateJobStatus cập nhật trạng thái job
func (qs *QueueService) UpdateJobStatus(jobID, status string) error {
	key := fmt.Sprintf("job_status:%s", jobID)
	err := qs.redisClient.Set(qs.ctx, key, status, time.Hour).Err() // TTL 1 giờ
	if err != nil {
		return fmt.Errorf("failed to update job status: %v", err)
	}
	return nil
}

// StoreJobResult lưu kết quả job
func (qs *QueueService) StoreJobResult(jobID, resultPath string) error {
	key := fmt.Sprintf("job_result:%s", jobID)
	err := qs.redisClient.Set(qs.ctx, key, resultPath, time.Hour*24).Err() // TTL 24 giờ
	if err != nil {
		return fmt.Errorf("failed to store job result: %v", err)
	}
	return nil
}

// GetJobResult lấy kết quả job
func (qs *QueueService) GetJobResult(jobID string) (string, error) {
	result, err := qs.redisClient.Get(qs.ctx, fmt.Sprintf("job_result:%s", jobID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get job result: %v", err)
	}
	return result, nil
}
