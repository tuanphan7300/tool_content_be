# Tool Content Backend

A Go-based backend service for content processing with AI capabilities.

## Features

- Text-to-Speech processing
- Voice separation and processing
- AI-powered content generation
- User authentication and management
- File upload and processing

## Prerequisites

- Go 1.19 or higher
- PostgreSQL database
- OpenAI API key
- Google Cloud credentials for Text-to-Speech

## Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd tool_content_be
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your actual credentials
   ```

4. **Set up credentials**
   - Create a `data/` directory
   - Place your Google Cloud service account JSON file in `data/google_cloud_tts_api.json`
   - Update the `.env` file with your OpenAI API key and other credentials

5. **Set up pretrained models**
   - Create a `pretrained_models/` directory
   - Download the required model files (see Model Setup section)

6. **Run the application**
   ```bash
   go run main.go
   ```

## Model Setup

The application requires pretrained models for voice separation. These files are large (>50MB) and are not included in the repository.

### Required Models

Place the following files in `pretrained_models/2stems/`:
- `model.data-00000-of-00001` (75MB)
- `model.index` (5KB)
- `model.meta` (787KB)
- `checkpoint` (67B)

### Download Instructions

1. Download the HTDemucs 2-stems model from the official repository
2. Extract the files to `pretrained_models/2stems/`
3. Ensure all files are present and have the correct names

## Environment Variables

See `env.example` for all required environment variables.

### Required Credentials

- **OpenAI API Key**: For AI content generation
- **Google Cloud Service Account**: For Text-to-Speech functionality
- **Database Credentials**: PostgreSQL connection details
- **JWT Secret**: For user authentication

## File Structure

```
tool_content_be/
├── config/          # Configuration files
├── handler/         # HTTP handlers
├── service/         # Business logic services
├── middleware/      # HTTP middleware
├── router/          # Route definitions
├── storage/         # File storage (gitignored)
├── data/           # Credentials and data (gitignored)
├── pretrained_models/ # AI models (gitignored)
└── util/           # Utility functions
```

## Security Notes

- Never commit `.env` files or credential files
- The `data/`, `storage/`, and `pretrained_models/` directories are gitignored
- Use environment variables for all sensitive configuration
- Keep your API keys secure and rotate them regularly

## Docker

To run with Docker:

```bash
docker-compose up -d
```

## API Documentation

The API endpoints are defined in the `router/` directory. Main endpoints include:

- `/api/auth/*` - Authentication endpoints
- `/api/upload` - File upload
- `/api/process` - Content processing
- `/api/tts` - Text-to-Speech
- `/api/history` - Processing history

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure all tests pass
5. Submit a pull request

## License

[Add your license information here] 

Phương án tối ưu thời gian xử lý
1. Parallel Processing (Song song hóa)
   └─────┘
2. Streaming Processing (Xử lý theo chunk)
   Chia video thành chunks 1-2 phút
   Xử lý song song từng chunk
   Merge kết quả cuối cùng
   Giảm thời gian từ 15-23 phút xuống 4-6 phút
3. Caching Strategy (Cache thông minh)
   )
4. Pre-processing Optimization
   Audio compression trước khi Whisper
   Video downscale cho processing
   Batch processing cho multiple requests
   Queue management với priority
5. Service Optimization
   Whisper Optimization:
   Sử dụng Whisper API với streaming
   Parallel transcription cho segments
   Local Whisper model cho simple cases
   TTS Optimization:
   Batch TTS generation
   Pre-generated voice models
   Streaming TTS synthesis
   Background Extraction:
   Lightweight Demucs model
   FFmpeg-based separation (nhanh hơn)
   Skip background nếu không cần
6. Smart Processing Pipeline
   Option A: Fast Mode (3-5 phút)
   )
   Option B: Quality Mode (6-8 phút)
   )
7. Infrastructure Optimization
   Server Scaling:
   Multiple worker nodes
   Load balancing
   GPU acceleration cho Demucs
   SSD storage cho I/O
   Queue Management:
   Priority queue cho VIP users
   Batch processing
   Resource allocation
8. User Experience Optimization
   Progress Tracking:
   └─────────────────────────────────────┘
   Preview Generation:
   Generate preview sau mỗi step
   Allow user to download intermediate results
   Background processing cho final video
9. Smart Fallbacks
   Progressive Enhancement:
   )
   Quality vs Speed Options:
   Fast Mode: 3-5 phút, chất lượng cơ bản
   Standard Mode: 6-8 phút, chất lượng tốt
   Premium Mode: 10-12 phút, chất lượng cao nhất
10. Resource Optimization
    Memory Management:
    Streaming file processing
    Temporary file cleanup
    Memory pooling
    Network Optimization:
    CDN cho file delivery
    Compression cho API calls
    Connection pooling
    🎯 Kết quả mong đợi:
    Before Optimization:
    Video 7 phút: 15-23 phút
    User experience: Chờ lâu, không biết tiến độ
    Resource usage: Inefficient
    After Optimization:
    Video 7 phút: 4-8 phút (giảm 60-70%)
    User experience: Real-time progress, preview
    Resource usage: Optimized, scalable
    Implementation Priority:
    Parallel Processing (Impact cao, effort thấp)
    Caching Strategy (Impact cao, effort trung bình)
    Streaming Processing (Impact cao, effort cao)
    Infrastructure Scaling (Impact trung bình, effort cao)
    Bạn muốn tôi đi sâu vào phương án nào cụ thể không?
1. Parallel Processing (Song song hóa)
   cách này thì bạn định triển khai như thế nào? chưa code nhé.

🚀 Parallel Processing Strategy - Triển khai chi tiết
1. Phân tích Dependencies (Quan hệ phụ thuộc)
   Dependency Graph hiện tại:
   Final
   Dependencies thực tế:
   Translation cần Whisper output
   TTS cần Translation output
   Background có thể chạy song song với TTS
   Merge cần cả TTS và Background
   Subtitle cần Merge output
2. Parallel Processing Architecture
   Phase 1: Independent Processing (0-3 phút)
   ──────────────┴─────────────────┴─────────────────┘
   Phase 2: Translation & TTS (3-6 phút)
   ──────────────┴─────────────────┴─────────────────┘
   Phase 3: Video Processing (6-8 phút)
   ──────────────┴─────────────────┴─────────────────┘
3. Implementation Strategy
   A. Worker Pool Architecture
   ──────────────────────────────────────────────────┘
   B. Task Queue System
   ──────────────────────────────────────────────────┘
4. Detailed Parallel Flow
   Step 1: Initial Setup (0-30s)
   ──────────────────────────────────────────────────┘
   Step 2: Independent Processing (30s-3min)
   ──────────────────────────────────────────────────┘
   Step 3: Dependent Processing (3-6min)
   ──────────────────────────────────────────────────┘
   Step 4: Final Processing (6-8min)
   ──────────────────────────────────────────────────┘
5. Resource Allocation Strategy
   CPU/Memory Distribution:
   ──────────────────────────────────────────────────┘
   GPU Allocation (nếu có):
   ──────────────────────────────────────────────────┘
6. Synchronization Points
   Critical Path:
   Block
   Non-blocking Paths:
   )
7. Error Handling & Recovery
   Partial Failure Handling:
   ──────────────────────────────────────────────────┘
   Retry Logic:
   ──────────────────────────────────────────────────┘
8. Progress Tracking
   Real-time Progress:
   ──────────────────────────────────────────────────┘
9. Expected Performance Improvement
   Before Parallel:
   phút
   After Parallel:
   )
10. Implementation Phases
    Phase 1: Basic Parallel (2-3 tuần)
    Worker pool setup
    Independent task parallelization
    Basic synchronization
    Phase 2: Advanced Parallel (3-4 tuần)
    Dependency management
    Error recovery
    Progress tracking
    Phase 3: Optimization (2-3 tuần)
    Resource optimization
    Performance tuning
    Monitoring & alerting
    Kết quả mong đợi: Giảm thời gian từ 15-23 phút xuống 6-8 phút (60-70% improvement)
    tiếng việt nhé. nói lại giúp tôi.

🚀 Chiến lược Xử lý Song song (Parallel Processing) - Triển khai chi tiết
1. Phân tích Quan hệ Phụ thuộc
   Sơ đồ phụ thuộc hiện tại:
   Final
   Quan hệ phụ thuộc thực tế:
   Dịch thuật cần kết quả từ Whisper
   TTS cần kết quả từ Dịch thuật
   Tách nhạc nền có thể chạy song song với TTS
   Merge cần cả TTS và Nhạc nền
   Burn subtitle cần kết quả Merge
2. Kiến trúc Xử lý Song song
   Giai đoạn 1: Xử lý Độc lập (0-3 phút)
   ──────────────┴─────────────────┴─────────────────┘
   Giai đoạn 2: Dịch thuật & TTS (3-6 phút)
   ──────────────┴─────────────────┴─────────────────┘
   Giai đoạn 3: Xử lý Video (6-8 phút)
   ──────────────┴─────────────────┴─────────────────┘
3. Chiến lược Triển khai
   A. Kiến trúc Worker Pool
   ──────────────────────────────────────────────────┘
   B. Hệ thống Queue tác vụ
   ──────────────────────────────────────────────────┘
4. Luồng Xử lý Song song Chi tiết
   Bước 1: Chuẩn bị ban đầu (0-30 giây)
   ──────────────────────────────────────────────────┘
   Bước 2: Xử lý Độc lập (30 giây-3 phút)
   ──────────────────────────────────────────────────┘
   Bước 3: Xử lý Phụ thuộc (3-6 phút)
   ──────────────────────────────────────────────────┘
   Bước 4: Xử lý Cuối cùng (6-8 phút)
   ──────────────────────────────────────────────────┘
5. Chiến lược Phân bổ Tài nguyên
   Phân bổ CPU/Bộ nhớ:
   ──────────────────────────────────────────────────┘
   Phân bổ GPU (nếu có):
   ──────────────────────────────────────────────────┘
6. Điểm Đồng bộ hóa
   Đường dẫn quan trọng:
   Chặn
   Đường dẫn không chặn:
   )
7. Xử lý Lỗi & Khôi phục
   Xử lý Lỗi Từng phần:
   ──────────────────────────────────────────────────┘
   Logic Thử lại:
   ──────────────────────────────────────────────────┘
8. Theo dõi Tiến độ
   Tiến độ Thời gian thực:
   ──────────────────────────────────────────────────┘
9. Cải thiện Hiệu suất Mong đợi
   Trước khi Song song:
   phút
   Sau khi Song song:
   )
10. Các Giai đoạn Triển khai
    Giai đoạn 1: Song song Cơ bản (2-3 tuần)
    Thiết lập worker pool
    Song song hóa tác vụ độc lập
    Đồng bộ hóa cơ bản
    Giai đoạn 2: Song song Nâng cao (3-4 tuần)
    Quản lý phụ thuộc
    Khôi phục lỗi
    Theo dõi tiến độ
    Giai đoạn 3: Tối ưu hóa (2-3 tuần)
    Tối ưu tài nguyên
    Điều chỉnh hiệu suất
    Giám sát & cảnh báo
    Kết quả mong đợi: Giảm thời gian từ 15-23 phút xuống 6-8 phút (cải thiện 60-70%)