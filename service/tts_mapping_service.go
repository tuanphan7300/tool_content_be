package service

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// TTSMapping lưu thông tin chi tiết về mỗi TTS segment
type TTSMapping struct {
	SegmentIndex   int           `json:"segment_index"`
	StartTime      float64       `json:"start_time"`
	EndTime        float64       `json:"end_time"`
	Text           string        `json:"text"`
	GoogleAPIResp  string        `json:"google_api_resp"`
	AudioDuration  float64       `json:"audio_duration"`
	PauseBefore    float64       `json:"pause_before"`
	PauseAfter     float64       `json:"pause_after"`
	AdjustedPath   string        `json:"adjusted_path"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          error         `json:"error,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	CompletedAt    time.Time     `json:"completed_at"`
}

// TTSMappingService quản lý tất cả TTS mappings
type TTSMappingService struct {
	mappings map[string]map[int]*TTSMapping // jobID -> segmentIndex -> mapping
	mutex    sync.RWMutex
}

var (
	ttsMappingService *TTSMappingService
	mappingMutex      sync.Mutex
)

// InitTTSMappingService khởi tạo TTS Mapping Service
func InitTTSMappingService() *TTSMappingService {
	mappingMutex.Lock()
	defer mappingMutex.Unlock()

	if ttsMappingService == nil {
		ttsMappingService = &TTSMappingService{
			mappings: make(map[string]map[int]*TTSMapping),
		}
		log.Println("TTS Mapping Service initialized successfully")
	}

	return ttsMappingService
}

// GetTTSMappingService trả về instance của TTS Mapping Service
func GetTTSMappingService() *TTSMappingService {
	if ttsMappingService == nil {
		return InitTTSMappingService()
	}
	return ttsMappingService
}

// CreateJobMapping tạo mapping cho một job mới
func (s *TTSMappingService) CreateJobMapping(jobID string, entries []SRTEntry) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.mappings[jobID] == nil {
		s.mappings[jobID] = make(map[int]*TTSMapping)
	}

	// Tạo mapping cho từng segment
	for i, entry := range entries {
		pauseBefore := 0.0
		pauseAfter := 0.0

		// Tính toán khoảng nghỉ trước segment
		if i > 0 {
			prevEntry := entries[i-1]
			gap := entry.Start - prevEntry.End
			if gap > 0.1 { // Chỉ tính pause nếu gap > 100ms
				pauseBefore = gap
			}
		}

		// Tính toán khoảng nghỉ sau segment
		if i < len(entries)-1 {
			nextEntry := entries[i+1]
			gap := nextEntry.Start - entry.End
			if gap > 0.1 { // Chỉ tính pause nếu gap > 100ms
				pauseAfter = gap
			}
		}

		mapping := &TTSMapping{
			SegmentIndex: i,
			StartTime:    entry.Start,
			EndTime:      entry.End,
			Text:         entry.Text,
			PauseBefore:  pauseBefore,
			PauseAfter:   pauseAfter,
			CreatedAt:    time.Now(),
		}

		s.mappings[jobID][i] = mapping
	}

	log.Printf("Created TTS mapping for job %s with %d segments", jobID, len(entries))
}

// UpdateSegmentMapping cập nhật mapping cho một segment
func (s *TTSMappingService) UpdateSegmentMapping(jobID string, segmentIndex int, updates map[string]interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.mappings[jobID] == nil {
		return fmt.Errorf("job mapping not found: %s", jobID)
	}

	mapping, exists := s.mappings[jobID][segmentIndex]
	if !exists {
		return fmt.Errorf("segment mapping not found: job=%s, segment=%d", jobID, segmentIndex)
	}

	// Cập nhật các trường
	for key, value := range updates {
		switch key {
		case "google_api_resp":
			if str, ok := value.(string); ok {
				mapping.GoogleAPIResp = str
			}
		case "audio_duration":
			if duration, ok := value.(float64); ok {
				mapping.AudioDuration = duration
			}
		case "adjusted_path":
			if path, ok := value.(string); ok {
				mapping.AdjustedPath = path
			}
		case "processing_time":
			if procTime, ok := value.(time.Duration); ok {
				mapping.ProcessingTime = procTime
			}
		case "error":
			if err, ok := value.(error); ok {
				mapping.Error = err
			}
		}
	}

	mapping.CompletedAt = time.Now()

	log.Printf("Updated segment %d mapping for job %s", segmentIndex, jobID)
	return nil
}

// GetSegmentMapping lấy mapping của một segment
func (s *TTSMappingService) GetSegmentMapping(jobID string, segmentIndex int) (*TTSMapping, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.mappings[jobID] == nil {
		return nil, fmt.Errorf("job mapping not found: %s", jobID)
	}

	mapping, exists := s.mappings[jobID][segmentIndex]
	if !exists {
		return nil, fmt.Errorf("segment mapping not found: job=%s, segment=%d", jobID, segmentIndex)
	}

	return mapping, nil
}

// GetJobMapping lấy tất cả mappings của một job
func (s *TTSMappingService) GetJobMapping(jobID string) (map[int]*TTSMapping, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.mappings[jobID] == nil {
		return nil, fmt.Errorf("job mapping not found: %s", jobID)
	}

	// Tạo copy để tránh race condition
	result := make(map[int]*TTSMapping)
	for k, v := range s.mappings[jobID] {
		result[k] = v
	}

	return result, nil
}

// GetJobProgress lấy tiến độ xử lý của một job
func (s *TTSMappingService) GetJobProgress(jobID string) map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.mappings[jobID] == nil {
		return map[string]interface{}{
			"error": "job mapping not found",
		}
	}

	totalSegments := len(s.mappings[jobID])
	completedSegments := 0
	failedSegments := 0
	processingSegments := 0

	for _, mapping := range s.mappings[jobID] {
		if mapping.Error != nil {
			failedSegments++
		} else if mapping.AdjustedPath != "" {
			completedSegments++
		} else {
			processingSegments++
		}
	}

	progressPercentage := 0.0
	if totalSegments > 0 {
		progressPercentage = float64(completedSegments) / float64(totalSegments) * 100
	}

	return map[string]interface{}{
		"job_id":              jobID,
		"total_segments":      totalSegments,
		"completed_segments":  completedSegments,
		"failed_segments":     failedSegments,
		"processing_segments": processingSegments,
		"progress_percentage": progressPercentage,
		"status":              s.getJobStatus(completedSegments, failedSegments, totalSegments),
	}
}

// getJobStatus xác định trạng thái của job
func (s *TTSMappingService) getJobStatus(completed, failed, total int) string {
	if failed > 0 && completed == 0 {
		return "failed"
	} else if completed == total {
		return "completed"
	} else if completed > 0 || failed > 0 {
		return "processing"
	} else {
		return "pending"
	}
}

// CleanupJobMapping xóa mapping của một job
func (s *TTSMappingService) CleanupJobMapping(jobID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.mappings, jobID)
	log.Printf("Cleaned up TTS mapping for job %s", jobID)
}

// GetAllJobMappings lấy tất cả job mappings (cho admin)
func (s *TTSMappingService) GetAllJobMappings() map[string]map[int]*TTSMapping {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]map[int]*TTSMapping)
	for jobID, segments := range s.mappings {
		result[jobID] = make(map[int]*TTSMapping)
		for segmentIndex, mapping := range segments {
			result[jobID][segmentIndex] = mapping
		}
	}

	return result
}

// GetJobStatistics lấy thống kê tổng quan
func (s *TTSMappingService) GetJobStatistics() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	totalJobs := len(s.mappings)
	totalSegments := 0
	totalCompleted := 0
	totalFailed := 0
	totalProcessing := 0

	for _, segments := range s.mappings {
		totalSegments += len(segments)
		for _, mapping := range segments {
			if mapping.Error != nil {
				totalFailed++
			} else if mapping.AdjustedPath != "" {
				totalCompleted++
			} else {
				totalProcessing++
			}
		}
	}

	return map[string]interface{}{
		"total_jobs":       totalJobs,
		"total_segments":   totalSegments,
		"total_completed":  totalCompleted,
		"total_failed":     totalFailed,
		"total_processing": totalProcessing,
		"success_rate":     s.calculateSuccessRate(totalCompleted, totalFailed, totalSegments),
		"average_job_size": s.calculateAverageJobSize(totalSegments, totalJobs),
	}
}

// calculateSuccessRate tính tỷ lệ thành công
func (s *TTSMappingService) calculateSuccessRate(completed, failed, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(completed) / float64(total) * 100
}

// calculateAverageJobSize tính kích thước trung bình của job
func (s *TTSMappingService) calculateAverageJobSize(totalSegments, totalJobs int) float64 {
	if totalJobs == 0 {
		return 0.0
	}
	return float64(totalSegments) / float64(totalJobs)
}
