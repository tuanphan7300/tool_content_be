package limit

import (
	"sync"
	"time"
)

type UploadCount struct {
	Count     int
	LastReset time.Time
}

var (
	counts = make(map[string]*UploadCount)
	mu     sync.Mutex
)

func CheckFreeLimit(ip string) bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	count, exists := counts[ip]
	if !exists || now.Sub(count.LastReset) > 24*time.Hour {
		counts[ip] = &UploadCount{Count: 0, LastReset: now}
		count = counts[ip]
	}

	if count.Count >= 3 {
		return false
	}
	count.Count++
	return true
}
