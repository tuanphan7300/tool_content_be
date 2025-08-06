package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry đại diện cho một entry trong cache
type CacheEntry struct {
	Key            string        `json:"key"`
	Value          string        `json:"value"`
	Type           string        `json:"type"`
	CreatedAt      time.Time     `json:"created_at"`
	ExpiresAt      time.Time     `json:"expires_at"`
	FileSize       int64         `json:"file_size"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// CacheService quản lý cache cho các kết quả xử lý
type CacheService struct {
	CacheDir string
}

// NewCacheService tạo service cache mới
func NewCacheService() *CacheService {
	cacheDir := "./cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("Failed to create cache directory: %v", err)
	}

	return &CacheService{
		CacheDir: cacheDir,
	}
}

// GenerateCacheKey tạo key cache từ input
func (c *CacheService) GenerateCacheKey(input string, cacheType string) string {
	hash := md5.Sum([]byte(input + cacheType))
	return hex.EncodeToString(hash[:])
}

// Get lấy giá trị từ cache
func (c *CacheService) Get(key string) (*CacheEntry, error) {
	cacheFile := filepath.Join(c.CacheDir, key+".json")

	// Kiểm tra file tồn tại
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache entry not found")
	}

	// Đọc file cache
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %v", err)
	}

	// Kiểm tra expiration
	if time.Now().After(entry.ExpiresAt) {
		// Cache đã hết hạn, xóa file
		os.Remove(cacheFile)
		return nil, fmt.Errorf("cache entry expired")
	}

	// Kiểm tra file value tồn tại
	if _, err := os.Stat(entry.Value); os.IsNotExist(err) {
		// File không tồn tại, xóa cache entry
		os.Remove(cacheFile)
		return nil, fmt.Errorf("cached file not found")
	}

	return &entry, nil
}

// Set lưu giá trị vào cache
func (c *CacheService) Set(key string, value string, cacheType string, ttl time.Duration) error {
	// Kiểm tra file value tồn tại
	fileInfo, err := os.Stat(value)
	if err != nil {
		return fmt.Errorf("value file not found: %v", err)
	}

	entry := CacheEntry{
		Key:       key,
		Value:     value,
		Type:      cacheType,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		FileSize:  fileInfo.Size(),
	}

	// Lưu entry vào file
	cacheFile := filepath.Join(c.CacheDir, key+".json")
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %v", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	log.Printf("Cached %s: %s -> %s", cacheType, key, value)
	return nil
}

// Delete xóa entry khỏi cache
func (c *CacheService) Delete(key string) error {
	cacheFile := filepath.Join(c.CacheDir, key+".json")

	// Đọc entry để lấy file value
	if entry, err := c.Get(key); err == nil {
		// Xóa file value
		if err := os.Remove(entry.Value); err != nil {
			log.Printf("Failed to remove cached file: %v", err)
		}
	}

	// Xóa file cache
	return os.Remove(cacheFile)
}

// CleanupExpired dọn dẹp cache đã hết hạn
func (c *CacheService) CleanupExpired() error {
	files, err := os.ReadDir(c.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %v", err)
	}

	var deletedCount int
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		cacheFile := filepath.Join(c.CacheDir, file.Name())
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			continue
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		// Kiểm tra expiration
		if time.Now().After(entry.ExpiresAt) {
			c.Delete(entry.Key)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("Cleaned up %d expired cache entries", deletedCount)
	}

	return nil
}

// GetStats lấy thống kê cache
func (c *CacheService) GetStats() map[string]interface{} {
	files, err := os.ReadDir(c.CacheDir)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	var totalSize int64
	var entryCount int
	typeStats := make(map[string]int)

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		cacheFile := filepath.Join(c.CacheDir, file.Name())
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			continue
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		// Kiểm tra expiration
		if time.Now().After(entry.ExpiresAt) {
			continue
		}

		entryCount++
		totalSize += entry.FileSize
		typeStats[entry.Type]++
	}

	return map[string]interface{}{
		"total_entries": entryCount,
		"total_size":    totalSize,
		"type_stats":    typeStats,
		"cache_dir":     c.CacheDir,
	}
}

// CacheWhisperResult cache kết quả Whisper
func (c *CacheService) CacheWhisperResult(audioPath string, result *WhisperResult) error {
	key := c.GenerateCacheKey(audioPath, "whisper")
	return c.Set(key, result.SRTPath, "whisper", 24*time.Hour) // Cache 24 giờ
}

// GetCachedWhisperResult lấy kết quả Whisper từ cache
func (c *CacheService) GetCachedWhisperResult(audioPath string) (*WhisperResult, error) {
	key := c.GenerateCacheKey(audioPath, "whisper")
	entry, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	// Đọc SRT content
	content, err := os.ReadFile(entry.Value)
	if err != nil {
		return nil, err
	}

	// Parse SRT để tạo segments (đơn giản hóa)
	segments := []Segment{{Start: 0, End: 0, Text: string(content)}}

	return &WhisperResult{
		Transcript: string(content),
		Segments:   segments,
		SRTPath:    entry.Value,
	}, nil
}

// CacheBackgroundResult cache kết quả background extraction
func (c *CacheService) CacheBackgroundResult(audioPath string, backgroundPath string) error {
	key := c.GenerateCacheKey(audioPath, "background")
	return c.Set(key, backgroundPath, "background", 12*time.Hour) // Cache 12 giờ
}

// GetCachedBackgroundResult lấy kết quả background từ cache
func (c *CacheService) GetCachedBackgroundResult(audioPath string) (string, error) {
	key := c.GenerateCacheKey(audioPath, "background")
	entry, err := c.Get(key)
	if err != nil {
		return "", err
	}

	return entry.Value, nil
}
