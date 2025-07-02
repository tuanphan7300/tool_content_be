# Queue System cho Demucs Processing

## 🎯 **Mục đích**

Giải quyết vấn đề bottleneck khi có nhiều người dùng cùng sử dụng Demucs (AI model tách âm thanh) đồng thời.

## 🏗️ **Kiến trúc**

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │───▶│   API       │───▶│   Queue     │
│   (Frontend)│    │   (Go)      │    │   (Redis)   │
└─────────────┘    └─────────────┘    └─────────────┘
                           │                   │
                           ▼                   ▼
                   ┌─────────────┐    ┌─────────────┐
                   │   Worker    │◀───│   Job       │
                   │   Service   │    │   Processing│
                   │   (Go)      │    │   (Demucs)  │
                   └─────────────┘    └─────────────┘
```

## 🚀 **Setup**

### 1. **Cài đặt Redis**

#### Cách 1: Docker (Khuyến nghị)
```bash
# Chạy Redis với docker-compose
docker-compose -f docker-compose.redis.yml up -d

# Kiểm tra Redis đang chạy
docker ps | grep redis
```

#### Cách 2: Cài đặt trực tiếp
```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt-get install redis-server
sudo systemctl start redis-server
```

### 2. **Cài đặt Dependencies**
```bash
cd tool_content_be
go mod tidy
```

### 3. **Chạy ứng dụng**
```bash
go run main.go
```

## 📊 **Cách hoạt động**

### **Trước khi có Queue System**
```
User 1 ──▶ Demucs (5 phút)
User 2 ──▶ Demucs (5 phút) ──▶ BLOCKED ❌
User 3 ──▶ Demucs (5 phút) ──▶ BLOCKED ❌
```

### **Sau khi có Queue System**
```
User 1 ──▶ Queue ──▶ Worker 1 ──▶ Demucs ✅
User 2 ──▶ Queue ──▶ Worker 2 ──▶ Demucs ✅  
User 3 ──▶ Queue ──▶ Waiting... ──▶ Demucs ✅
```

## 🔧 **Cấu hình**

### **Worker Configuration**
- **Max Workers**: Số CPU cores (tối đa 4)
- **Max Concurrent**: 2 Demucs processes cùng lúc
- **Priority Levels**: 1-10 (10 cao nhất)

### **Job Configuration**
- **Timeout**: 2x audio duration + 60s buffer
- **Priority**: 5 (mặc định)
- **Retry**: Không (fail fast)

## 📡 **API Endpoints**

### **Queue Management**
```bash
# Xem trạng thái queue
GET /queue/status

# Xem trạng thái worker
GET /queue/worker/status

# Xem trạng thái job
GET /queue/job/{job_id}/status

# Lấy kết quả job
GET /queue/job/{job_id}/result

# Chờ job hoàn thành
GET /queue/job/{job_id}/wait?timeout=300

# Khởi động worker (admin)
POST /queue/worker/start

# Dừng worker (admin)
POST /queue/worker/stop
```

### **Response Examples**

#### Queue Status
```json
{
  "queue_status": {
    "priority_1": 0,
    "priority_2": 0,
    "priority_3": 0,
    "priority_4": 0,
    "priority_5": 2,
    "priority_6": 0,
    "priority_7": 0,
    "priority_8": 0,
    "priority_9": 0,
    "priority_10": 0
  }
}
```

#### Worker Status
```json
{
  "is_running": true,
  "max_workers": 4,
  "max_concurrent": 2,
  "active_workers": 1,
  "queue_status": {
    "priority_5": 2
  }
}
```

#### Job Status
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing"
}
```

## 🔄 **Workflow**

### **1. User upload video**
```bash
POST /process-video
```

### **2. API tạo job**
```go
jobID, err := service.ExtractBackgroundMusicAsync(
    audioPath, fileName, videoDir, userID, videoID
)
```

### **3. Job được thêm vào queue**
```go
job := &AudioProcessingJob{
    ID:          jobID,
    AudioPath:   audioPath,
    StemType:    "no_vocals",
    Priority:    5,
    MaxDuration: duration*2 + 60,
}
queueService.EnqueueJob(job)
```

### **4. Worker xử lý job**
```go
// Worker lấy job từ queue
job := queueService.DequeueJob()

// Chạy Demucs với timeout
result, err := runDemucs(ctx, job)

// Lưu kết quả
queueService.StoreJobResult(jobID, resultPath)
```

### **5. Client poll kết quả**
```bash
GET /queue/job/{job_id}/status
# hoặc
GET /queue/job/{job_id}/wait?timeout=300
```

## 📈 **Monitoring**

### **Redis Commander**
Truy cập: http://localhost:8081
- Xem queue status
- Monitor Redis performance
- Debug job data

### **Logs**
```bash
# Xem worker logs
tail -f logs/worker.log

# Xem queue logs  
tail -f logs/queue.log
```

## 🚨 **Troubleshooting**

### **Redis Connection Failed**
```bash
# Kiểm tra Redis
redis-cli ping

# Restart Redis
docker-compose -f docker-compose.redis.yml restart
```

### **Worker Not Processing**
```bash
# Kiểm tra worker status
curl http://localhost:8888/queue/worker/status

# Restart worker
curl -X POST http://localhost:8888/queue/worker/stop
curl -X POST http://localhost:8888/queue/worker/start
```

### **Job Stuck**
```bash
# Kiểm tra job status
curl http://localhost:8888/queue/job/{job_id}/status

# Xem queue
curl http://localhost:8888/queue/status
```

## 🔒 **Security**

- **Authentication**: Tất cả endpoints yêu cầu JWT token
- **Rate Limiting**: Giới hạn 10 requests/phút/user
- **Resource Limits**: Max 2 Demucs processes
- **Timeout**: Job timeout tự động

## 📊 **Performance**

### **Before Queue System**
- **Concurrent Users**: 1-2
- **Response Time**: 5-10 phút
- **Resource Usage**: 100% CPU khi busy

### **After Queue System**
- **Concurrent Users**: 10-20
- **Response Time**: 5-10 phút (queued)
- **Resource Usage**: 50-70% CPU (controlled)

## 🎯 **Benefits**

1. **Scalability**: Hỗ trợ nhiều user cùng lúc
2. **Reliability**: Job không bị mất khi server restart
3. **Monitoring**: Theo dõi được trạng thái xử lý
4. **Resource Control**: Giới hạn tài nguyên sử dụng
5. **User Experience**: Không bị timeout/error

## 🔮 **Future Improvements**

1. **Job Priority**: VIP users có priority cao hơn
2. **Retry Logic**: Tự động retry khi fail
3. **Load Balancing**: Multiple worker instances
4. **Caching**: Cache kết quả cho file giống nhau
5. **WebSocket**: Real-time status updates 