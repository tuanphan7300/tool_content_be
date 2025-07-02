# Multi-stage build để tối ưu image size
FROM golang:1.23-alpine AS builder

# Cài đặt dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary với optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main .

# Production stage
FROM alpine:latest

# Cài đặt runtime dependencies
RUN apk --no-cache add ca-certificates tzdata ffmpeg python3 py3-pip

# Cài đặt Demucs
RUN pip3 install demucs

# Tạo non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary từ builder stage
COPY --from=builder /app/main .

# Copy pretrained models (nếu có)
COPY --chown=appuser:appgroup pretrained_models/ ./pretrained_models/

# Tạo storage directory
RUN mkdir -p storage && chown -R appuser:appgroup storage

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8888

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/ping || exit 1

# Run binary
CMD ["./main"]
