# 🐳 Docker Deployment Guide

## 📋 **Ưu điểm của Docker**

- ✅ **Consistent environment**: Chạy giống nhau ở mọi nơi
- ✅ **Easy scaling**: Dễ dàng scale up/down
- ✅ **Isolation**: Mỗi service chạy độc lập
- ✅ **Easy updates**: Update từng service riêng biệt
- ✅ **Rollback**: Dễ dàng rollback về version cũ

## ⚠️ **Nhược điểm**

- ❌ **Phức tạp hơn**: Cần hiểu Docker concepts
- ❌ **Resource overhead**: Docker engine tiêu tốn tài nguyên
- ❌ **Learning curve**: Cần thời gian học Docker

## 🚀 **Docker Deployment Steps**

### **1. Cài đặt Docker trên VPS**

```bash
# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Add user to docker group
usermod -aG docker $USER

# Start Docker
systemctl start docker
systemctl enable docker
```

### **2. Tạo Dockerfile cho API**

```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata ffmpeg python3 py3-pip

# Install Demucs
RUN pip3 install demucs

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/main .
COPY --chown=appuser:appgroup pretrained_models/ ./pretrained_models/

RUN mkdir -p storage && chown -R appuser:appgroup storage

USER appuser
EXPOSE 8888

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/ping || exit 1

CMD ["./main"]
```

### **3. Tạo docker-compose.yml**

```yaml
version: '3.8'

services:
  # API Service
  api:
    build: .
    container_name: tool-creator-api
    restart: unless-stopped
    ports:
      - "8888:8888"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=tool_creator
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=tool_creator
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - PORT=8888
      - ENVIRONMENT=production
    volumes:
      - app_storage:/app/storage
      - app_models:/app/pretrained_models
    depends_on:
      - mysql
      - redis
    networks:
      - app-network

  # Worker Service
  worker:
    build: .
    container_name: tool-creator-worker
    restart: unless-stopped
    command: ["./main", "--worker"]
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=tool_creator
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=tool_creator
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - PORT=8888
      - ENVIRONMENT=production
    volumes:
      - app_storage:/app/storage
      - app_models:/app/pretrained_models
    depends_on:
      - mysql
      - redis
    networks:
      - app-network

  # MySQL Database
  mysql:
    image: mysql:8.0
    container_name: tool-creator-mysql
    restart: unless-stopped
    environment:
      - MYSQL_ROOT_PASSWORD=${DB_ROOT_PASSWORD}
      - MYSQL_DATABASE=tool_creator
      - MYSQL_USER=tool_creator
      - MYSQL_PASSWORD=${DB_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "3306:3306"
    networks:
      - app-network
    command: --default-authentication-plugin=mysql_native_password

  # Redis Cache
  redis:
    image: redis:7-alpine
    container_name: tool-creator-redis
    restart: unless-stopped
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    networks:
      - app-network
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru

  # Nginx Reverse Proxy
  nginx:
    image: nginx:alpine
    container_name: tool-creator-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
      - app_storage:/var/www/static
    depends_on:
      - api
    networks:
      - app-network

volumes:
  mysql_data:
  redis_data:
  app_storage:
  app_models:

networks:
  app-network:
    driver: bridge
```

### **4. Tạo file .env**

```bash
# .env
DB_PASSWORD=your-secure-password
DB_ROOT_PASSWORD=your-root-password
JWT_SECRET=your-jwt-secret
OPENAI_API_KEY=your-openai-key
GEMINI_API_KEY=your-gemini-key
DOMAIN=your-domain.com
```

### **5. Tạo nginx.conf**

```nginx
events {
    worker_connections 1024;
}

http {
    upstream api_backend {
        server api:8888;
    }

    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;

        location /api/ {
            proxy_pass http://api_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 300s;
            client_max_body_size 100M;
        }

        location /static/ {
            alias /var/www/static/;
            expires 1y;
        }
    }
}
```

### **6. Deploy với Docker**

```bash
# Build và start services
docker-compose up -d --build

# Check status
docker-compose ps

# View logs
docker-compose logs -f api
docker-compose logs -f worker

# Scale workers
docker-compose up -d --scale worker=3

# Update application
git pull
docker-compose up -d --build api worker

# Backup
docker-compose exec mysql mysqldump -u tool_creator -p tool_creator > backup.sql
```

### **7. SSL với Docker**

```bash
# Create SSL directory
mkdir -p ssl

# Get SSL certificate
docker run --rm -v $(pwd)/ssl:/etc/letsencrypt -v $(pwd)/ssl:/var/lib/letsencrypt certbot/certbot certonly --standalone -d your-domain.com --email your-email@domain.com --agree-tos --no-eff-email

# Setup auto-renewal
echo "0 12 * * * docker run --rm -v $(pwd)/ssl:/etc/letsencrypt -v $(pwd)/ssl:/var/lib/letsencrypt certbot/certbot renew --quiet" | crontab -
```

## 🔧 **Docker Commands Cheat Sheet**

```bash
# Build image
docker build -t tool-creator .

# Run container
docker run -d -p 8888:8888 --name api tool-creator

# Stop container
docker stop api

# Remove container
docker rm api

# View logs
docker logs -f api

# Execute command in container
docker exec -it api sh

# Copy file from container
docker cp api:/app/logs/app.log ./app.log

# View resource usage
docker stats

# Clean up unused resources
docker system prune -a
```

## 📊 **Monitoring với Docker**

```bash
# Create monitoring script
cat > monitor-docker.sh <<'EOF'
#!/bin/bash
echo "=== Docker Containers ==="
docker ps

echo -e "\n=== Resource Usage ==="
docker stats --no-stream

echo -e "\n=== Container Logs ==="
docker logs --tail 10 tool-creator-api
EOF

chmod +x monitor-docker.sh
```

## 🚀 **Docker vs Manual Install**

| Aspect | Manual Install | Docker |
|--------|----------------|--------|
| **Setup Time** | 2-3 hours | 30 minutes |
| **Complexity** | Medium | High |
| **Resource Usage** | Low | Medium |
| **Scaling** | Manual | Easy |
| **Updates** | Manual | Automated |
| **Learning Curve** | Low | High |
| **Debugging** | Easy | Medium |

## 🎯 **Khuyến nghị**

### **Dùng Docker khi:**
- ✅ Có kinh nghiệm với Docker
- ✅ Cần deploy nhanh
- ✅ Cần scale dễ dàng
- ✅ Có team development

### **Dùng Manual Install khi:**
- ✅ Mới bắt đầu deploy
- ✅ Muốn hiểu rõ hệ thống
- ✅ Cần tối ưu performance
- ✅ Server resources hạn chế

## 📝 **Troubleshooting Docker**

```bash
# Container không start
docker logs container-name

# Port conflict
docker ps -a
netstat -tlnp | grep :8888

# Volume issues
docker volume ls
docker volume inspect volume-name

# Network issues
docker network ls
docker network inspect app-network

# Resource issues
docker stats
df -h
free -h
```

## 🔄 **CI/CD với Docker**

```yaml
# .github/workflows/docker-deploy.yml
name: Docker Deploy

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Deploy to server
      uses: appleboy/ssh-action@v0.1.5
      with:
        host: ${{ secrets.HOST }}
        username: ${{ secrets.USERNAME }}
        key: ${{ secrets.KEY }}
        script: |
          cd /opt/tool-creator
          git pull
          docker-compose down
          docker-compose up -d --build
          docker system prune -f
```

Docker deployment phù hợp cho những ai đã có kinh nghiệm và muốn có một hệ thống scalable, nhưng với người mới bắt đầu, tôi khuyến nghị dùng **Manual Install** trước để hiểu rõ hệ thống! 🎯 