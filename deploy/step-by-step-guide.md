# 🚀 Hướng dẫn Deploy từ A-Z cho người mới bắt đầu

## 📋 **Chuẩn bị trước khi deploy**

### **1. Chuẩn bị VPS**
- ✅ Đăng ký VPS (Hetzner CX21 - €23/tháng)
- ✅ Có domain name (mua từ Namecheap/GoDaddy)
- ✅ Có API keys (OpenAI, Gemini)

### **2. Chuẩn bị máy local**
- ✅ SSH client (Terminal trên Mac/Linux, PuTTY trên Windows)
- ✅ Code đã test xong
- ✅ Backup code lên GitHub

## 🖥️ **Bước 1: Tạo VPS**

### **Hetzner (Khuyến nghị)**
1. Truy cập https://hetzner.com
2. Đăng ký tài khoản
3. Chọn **CX21** (3CPU, 8GB RAM, 80GB SSD)
4. Chọn **Ubuntu 22.04 LTS**
5. Chọn data center gần nhất (Singapore/Japan)
6. Tạo SSH key hoặc dùng password
7. Thanh toán và chờ 5-10 phút

### **Lấy thông tin server**
- **IP Address**: 123.456.789.123
- **Username**: root
- **Password**: (nếu dùng password)

## 🔑 **Bước 2: Kết nối SSH**

### **Trên Mac/Linux:**
```bash
ssh root@123.456.789.123
```

### **Trên Windows (PuTTY):**
1. Mở PuTTY
2. Nhập IP: 123.456.789.123
3. Port: 22
4. Click Connect
5. Nhập username: root
6. Nhập password

## 📦 **Bước 3: Cài đặt hệ thống**

### **Cách 1: Dùng script tự động (Khuyến nghị)**
```bash
# Tải script về server
wget https://raw.githubusercontent.com/your-repo/tool-creator/main/deploy/quick-deploy.sh

# Chạy script
chmod +x quick-deploy.sh
./quick-deploy.sh
```

### **Cách 2: Cài thủ công từng bước**

#### **3.1 Cập nhật hệ thống**
```bash
apt update && apt upgrade -y
```

#### **3.2 Cài đặt các package cần thiết**
```bash
apt install -y curl wget git nginx mysql-server redis-server supervisor certbot python3-certbot-nginx python3 python3-pip ffmpeg build-essential
```

#### **3.3 Cài đặt Go**
```bash
# Tải Go
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz

# Giải nén
tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz

# Thêm vào PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Kiểm tra
go version
```

#### **3.4 Cài đặt Demucs**
```bash
# Tạo virtual environment
python3 -m venv /opt/demucs
source /opt/demucs/bin/activate

# Cài Demucs
pip install demucs

# Tải models (có thể mất 30-60 phút)
demucs --download
```

## 🗄️ **Bước 4: Cấu hình Database**

### **4.1 Cấu hình MySQL**
```bash
# Tạo database
mysql -e "CREATE DATABASE IF NOT EXISTS tool_creator CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# Tạo user
mysql -e "CREATE USER IF NOT EXISTS 'tool_creator'@'localhost' IDENTIFIED BY 'your-password';"

# Cấp quyền
mysql -e "GRANT ALL PRIVILEGES ON tool_creator.* TO 'tool_creator'@'localhost';"
mysql -e "FLUSH PRIVILEGES;"
```

### **4.2 Chạy migration**
```bash
# Tạo file migration
cat > migration.sql <<EOF
ALTER TABLE caption_history ADD COLUMN video_file_name_origin VARCHAR(255);
EOF

# Chạy migration
mysql -u tool_creator -p tool_creator < migration.sql
```

## 📁 **Bước 5: Deploy code**

### **5.1 Tạo thư mục ứng dụng**
```bash
mkdir -p /opt/tool-creator
cd /opt/tool-creator
```

### **5.2 Upload code lên server**

#### **Cách A: Dùng Git (Khuyến nghị)**
```bash
# Clone từ GitHub
git clone https://github.com/your-username/tool-creator.git .

# Hoặc nếu code chưa push lên GitHub
# Tạo repo trên GitHub trước, rồi:
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/your-username/tool-creator.git
git push -u origin main
```

#### **Cách B: Dùng SCP (nếu code ở local)**
```bash
# Trên máy local, chạy:
scp -r /path/to/your/code root@123.456.789.123:/opt/tool-creator/
```

#### **Cách C: Dùng rsync**
```bash
# Trên máy local, chạy:
rsync -avz --exclude 'node_modules' --exclude '.git' /path/to/your/code/ root@123.456.789.123:/opt/tool-creator/
```

### **5.3 Tạo file cấu hình**
```bash
cat > /opt/tool-creator/.env <<EOF
DB_HOST=localhost
DB_PORT=3306
DB_USER=tool_creator
DB_PASSWORD=your-password
DB_NAME=tool_creator
REDIS_HOST=localhost
REDIS_PORT=6379
OPENAI_API_KEY=your-openai-key
GEMINI_API_KEY=your-gemini-key
JWT_SECRET=your-jwt-secret
PORT=8888
ENVIRONMENT=production
STORAGE_PATH=/opt/tool-creator/storage
MODELS_PATH=/opt/tool-creator/pretrained_models
QUEUE_WORKERS=2
QUEUE_TIMEOUT=1800
EOF
```

### **5.4 Build ứng dụng**
```bash
cd /opt/tool-creator

# Download dependencies
go mod download

# Build
go build -o tool-creator-api .

# Tạo thư mục cần thiết
mkdir -p storage pretrained_models logs

# Set permissions
chown -R www-data:www-data /opt/tool-creator
chmod 600 /opt/tool-creator/.env
```

## ⚙️ **Bước 6: Cấu hình Supervisor**

### **6.1 Tạo file cấu hình**
```bash
cat > /etc/supervisor/conf.d/tool-creator.conf <<EOF
[program:tool-creator-api]
command=/opt/tool-creator/tool-creator-api
directory=/opt/tool-creator
user=www-data
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/opt/tool-creator/logs/api.log
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=10

[program:tool-creator-worker]
command=/opt/tool-creator/tool-creator-api --worker
directory=/opt/tool-creator
user=www-data
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/opt/tool-creator/logs/worker.log
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=10
EOF
```

### **6.2 Khởi động Supervisor**
```bash
# Reload config
supervisorctl reread
supervisorctl update

# Start services
supervisorctl start tool-creator-api
supervisorctl start tool-creator-worker

# Check status
supervisorctl status
```

## 🌐 **Bước 7: Cấu hình Nginx**

### **7.1 Tạo file cấu hình**
```bash
cat > /etc/nginx/sites-available/tool-creator <<EOF
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    
    location /api/ {
        proxy_pass http://127.0.0.1:8888;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        client_max_body_size 100M;
    }
    
    location /static/ {
        alias /opt/tool-creator/storage/;
        expires 1y;
    }
}
EOF
```

### **7.2 Enable site**
```bash
# Remove default site
rm -f /etc/nginx/sites-enabled/default

# Enable our site
ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/

# Test config
nginx -t

# Reload nginx
systemctl reload nginx
```

## 🔒 **Bước 8: Cấu hình SSL**

### **8.1 Lấy SSL certificate**
```bash
# Đảm bảo domain đã trỏ về IP server trước
certbot --nginx -d your-domain.com --non-interactive --agree-tos --email your-email@domain.com
```

### **8.2 Setup auto-renewal**
```bash
# Thêm vào crontab
echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
```

## 📊 **Bước 9: Setup Monitoring**

### **9.1 Tạo script monitoring**
```bash
cat > /opt/tool-creator/monitor.sh <<'EOF'
#!/bin/bash
echo "=== System Status ==="
echo "CPU: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)%"
echo "Memory: $(free | grep Mem | awk '{printf("%.2f%%", $3/$2 * 100.0)}')"
echo "Disk: $(df -h / | awk 'NR==2 {print $5}')"

echo -e "\n=== Application Status ==="
echo "API: $(supervisorctl status tool-creator-api | awk '{print $2}')"
echo "Worker: $(supervisorctl status tool-creator-worker | awk '{print $2}')"
echo "MySQL: $(systemctl is-active mysql)"
echo "Redis: $(systemctl is-active redis-server)"
echo "Nginx: $(systemctl is-active nginx)"

echo -e "\n=== Queue Status ==="
redis-cli llen tool_creator:queue:high
redis-cli llen tool_creator:queue:normal
redis-cli llen tool_creator:queue:low
EOF

chmod +x /opt/tool-creator/monitor.sh
```

## 🧪 **Bước 10: Test ứng dụng**

### **10.1 Test local trên server**
```bash
# Test API
curl http://localhost:8888/ping

# Test queue
curl http://localhost:8888/queue/status
```

### **10.2 Test từ bên ngoài**
```bash
# Test domain
curl https://your-domain.com/api/ping

# Test upload (từ máy local)
curl -X POST https://your-domain.com/api/upload \
  -F "file=@test-video.mp4" \
  -F "user_id=1"
```

## 🔧 **Bước 11: Troubleshooting**

### **11.1 Kiểm tra logs**
```bash
# API logs
tail -f /opt/tool-creator/logs/api.log

# Worker logs
tail -f /opt/tool-creator/logs/worker.log

# Nginx logs
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log

# MySQL logs
tail -f /var/log/mysql/error.log
```

### **11.2 Kiểm tra services**
```bash
# Check all services
systemctl status nginx mysql redis-server supervisor

# Check supervisor processes
supervisorctl status

# Check ports
netstat -tlnp | grep :8888
netstat -tlnp | grep :3306
netstat -tlnp | grep :6379
```

### **11.3 Restart services**
```bash
# Restart application
supervisorctl restart tool-creator-api
supervisorctl restart tool-creator-worker

# Restart system services
systemctl restart nginx
systemctl restart mysql
systemctl restart redis-server
```

## 📝 **Bước 12: Backup & Maintenance**

### **12.1 Tạo script backup**
```bash
cat > /opt/tool-creator/backup.sh <<EOF
#!/bin/bash
BACKUP_DIR="/opt/backups"
DATE=\$(date +%Y%m%d_%H%M%S)

mkdir -p \$BACKUP_DIR

# Backup database
mysqldump -u tool_creator -p your-password tool_creator > \$BACKUP_DIR/db_\$DATE.sql

# Backup files
tar -czf \$BACKUP_DIR/app_\$DATE.tar.gz -C /opt tool-creator/storage

# Keep only 7 days
find \$BACKUP_DIR -name "*.sql" -mtime +7 -delete
find \$BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: \$DATE"
EOF

chmod +x /opt/tool-creator/backup.sh

# Setup daily backup
echo "0 2 * * * /opt/tool-creator/backup.sh >> /opt/tool-creator/logs/backup.log 2>&1" | crontab -
```

## ✅ **Hoàn thành!**

Sau khi hoàn thành tất cả các bước trên:

- 🌐 **API URL**: https://your-domain.com/api
- 📊 **Monitor**: `/opt/tool-creator/monitor.sh`
- 📝 **Logs**: `/opt/tool-creator/logs/`
- 💾 **Backup**: `/opt/tool-creator/backup.sh`

### **Các lệnh hữu ích:**
```bash
# Check status
supervisorctl status

# View logs
tail -f /opt/tool-creator/logs/api.log

# Monitor system
/opt/tool-creator/monitor.sh

# Manual backup
/opt/tool-creator/backup.sh
```

## 🆘 **Khi gặp vấn đề**

1. **Kiểm tra logs trước**
2. **Google error message**
3. **Restart services**
4. **Check disk space**: `df -h`
5. **Check memory**: `free -h`
6. **Check CPU**: `htop`

## 📞 **Hỗ trợ**

- **Stack Overflow**: Tìm câu hỏi tương tự
- **GitHub Issues**: Báo lỗi cho project
- **Server provider support**: Hetzner có support tốt 