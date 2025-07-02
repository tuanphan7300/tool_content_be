#!/bin/bash

# VPS Single Server Deployment Script
# Chạy tất cả services trên 1 server để tiết kiệm chi phí

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN=${DOMAIN:-"your-domain.com"}
DB_PASSWORD=${DB_PASSWORD:-"your-secure-password"}
JWT_SECRET=${JWT_SECRET:-"your-jwt-secret"}
OPENAI_API_KEY=${OPENAI_API_KEY:-"your-openai-key"}
GEMINI_API_KEY=${GEMINI_API_KEY:-"your-gemini-key"}

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root"
    fi
}

# Update system
update_system() {
    log "Updating system packages..."
    apt update && apt upgrade -y
    log "System updated successfully"
}

# Install dependencies
install_dependencies() {
    log "Installing system dependencies..."
    
    # Install required packages
    apt install -y \
        curl \
        wget \
        git \
        unzip \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg \
        lsb-release \
        nginx \
        certbot \
        python3-certbot-nginx \
        supervisor \
        htop \
        vim \
        ufw \
        fail2ban \
        logrotate \
        cron \
        build-essential \
        python3 \
        python3-pip \
        python3-venv \
        ffmpeg \
        redis-server \
        mysql-server \
        mysql-client \
        nodejs \
        npm
    
    log "Dependencies installed successfully"
}

# Configure firewall
configure_firewall() {
    log "Configuring firewall..."
    
    ufw --force enable
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow ssh
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 8888/tcp  # API port
    ufw allow 3306/tcp  # MySQL (internal only)
    ufw allow 6379/tcp  # Redis (internal only)
    
    log "Firewall configured successfully"
}

# Configure MySQL
configure_mysql() {
    log "Configuring MySQL..."
    
    # Secure MySQL installation
    mysql_secure_installation <<EOF

y
0
$DB_PASSWORD
$DB_PASSWORD
y
y
y
y
EOF
    
    # Create database and user
    mysql -u root -p$DB_PASSWORD <<EOF
CREATE DATABASE IF NOT EXISTS tool_creator CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'tool_creator'@'localhost' IDENTIFIED BY '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON tool_creator.* TO 'tool_creator'@'localhost';
FLUSH PRIVILEGES;
EOF
    
    # Configure MySQL for better performance
    cat > /etc/mysql/conf.d/tool-creator.cnf <<EOF
[mysqld]
innodb_buffer_pool_size = 256M
innodb_log_file_size = 64M
innodb_flush_log_at_trx_commit = 2
innodb_flush_method = O_DIRECT
max_connections = 100
query_cache_size = 32M
query_cache_type = 1
tmp_table_size = 64M
max_heap_table_size = 64M
EOF
    
    systemctl restart mysql
    log "MySQL configured successfully"
}

# Configure Redis
configure_redis() {
    log "Configuring Redis..."
    
    # Configure Redis for better performance
    cat > /etc/redis/redis.conf <<EOF
bind 127.0.0.1
port 6379
timeout 0
tcp-keepalive 300
daemonize yes
supervised systemd
pidfile /var/run/redis/redis-server.pid
loglevel notice
logfile /var/log/redis/redis-server.log
databases 16
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /var/lib/redis
maxmemory 256mb
maxmemory-policy allkeys-lru
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
EOF
    
    systemctl restart redis-server
    log "Redis configured successfully"
}

# Install Go
install_go() {
    log "Installing Go..."
    
    GO_VERSION="1.23"
    wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    source /etc/profile
    
    log "Go installed successfully"
}

# Install Demucs
install_demucs() {
    log "Installing Demucs..."
    
    # Create virtual environment
    python3 -m venv /opt/demucs
    source /opt/demucs/bin/activate
    
    # Install Demucs
    pip install demucs
    
    # Download models
    demucs --download
    
    log "Demucs installed successfully"
}

# Setup application
setup_application() {
    log "Setting up application..."
    
    # Create application directory
    mkdir -p /opt/tool-creator
    cd /opt/tool-creator
    
    # Clone repository (replace with your repo)
    # git clone https://github.com/your-username/tool-creator.git .
    
    # Create necessary directories
    mkdir -p storage pretrained_models logs
    
    # Set permissions
    chown -R www-data:www-data /opt/tool-creator
    chmod -R 755 /opt/tool-creator
    
    log "Application setup completed"
}

# Build application
build_application() {
    log "Building application..."
    
    cd /opt/tool-creator
    
    # Set Go environment
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=/opt/tool-creator
    export GOCACHE=/opt/tool-creator/.cache
    
    # Download dependencies
    go mod download
    
    # Build application
    go build -o tool-creator-api .
    
    log "Application built successfully"
}

# Create environment file
create_env_file() {
    log "Creating environment configuration..."
    
    cat > /opt/tool-creator/.env <<EOF
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=tool_creator
DB_PASSWORD=$DB_PASSWORD
DB_NAME=tool_creator

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379

# API Keys
OPENAI_API_KEY=$OPENAI_API_KEY
GEMINI_API_KEY=$GEMINI_API_KEY

# JWT Configuration
JWT_SECRET=$JWT_SECRET

# Application Configuration
PORT=8888
ENVIRONMENT=production
LOG_LEVEL=info

# File Storage
STORAGE_PATH=/opt/tool-creator/storage
MODELS_PATH=/opt/tool-creator/pretrained_models

# Queue Configuration
QUEUE_WORKERS=2
QUEUE_TIMEOUT=1800
EOF
    
    chown www-data:www-data /opt/tool-creator/.env
    chmod 600 /opt/tool-creator/.env
    
    log "Environment configuration created"
}

# Configure Supervisor
configure_supervisor() {
    log "Configuring Supervisor..."
    
    # API Service
    cat > /etc/supervisor/conf.d/tool-creator-api.conf <<EOF
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
environment=HOME="/opt/tool-creator"
EOF
    
    # Worker Service
    cat > /etc/supervisor/conf.d/tool-creator-worker.conf <<EOF
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
environment=HOME="/opt/tool-creator"
EOF
    
    # Reload supervisor
    supervisorctl reread
    supervisorctl update
    
    log "Supervisor configured successfully"
}

# Configure Nginx
configure_nginx() {
    log "Configuring Nginx..."
    
    # Remove default site
    rm -f /etc/nginx/sites-enabled/default
    
    # Create application site
    cat > /etc/nginx/sites-available/tool-creator <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    
    # Redirect to HTTPS
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;
    
    # SSL Configuration (will be configured by certbot)
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
    
    # Rate limiting
    limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone \$binary_remote_addr zone=upload:10m rate=2r/s;
    
    # API endpoints
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        
        proxy_pass http://127.0.0.1:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }
    
    # Upload endpoint (higher rate limit)
    location /api/upload {
        limit_req zone=upload burst=5 nodelay;
        
        client_max_body_size 100M;
        proxy_pass http://127.0.0.1:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        proxy_read_timeout 600s;
        proxy_connect_timeout 75s;
    }
    
    # Static files
    location /static/ {
        alias /opt/tool-creator/storage/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # Health check
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF
    
    # Enable site
    ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/
    
    # Test configuration
    nginx -t
    
    # Reload nginx
    systemctl reload nginx
    
    log "Nginx configured successfully"
}

# Setup SSL certificate
setup_ssl() {
    log "Setting up SSL certificate..."
    
    # Get SSL certificate
    certbot --nginx -d $DOMAIN --non-interactive --agree-tos --email admin@$DOMAIN
    
    # Setup auto-renewal
    echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
    
    log "SSL certificate configured successfully"
}

# Setup monitoring
setup_monitoring() {
    log "Setting up monitoring..."
    
    # Install monitoring tools
    apt install -y htop iotop nethogs
    
    # Create monitoring script
    cat > /opt/tool-creator/monitor.sh <<'EOF'
#!/bin/bash

# Simple monitoring script
echo "=== System Status ==="
echo "CPU Usage: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)%"
echo "Memory Usage: $(free | grep Mem | awk '{printf("%.2f%%", $3/$2 * 100.0)}')"
echo "Disk Usage: $(df -h / | awk 'NR==2 {print $5}')"
echo "Load Average: $(uptime | awk -F'load average:' '{print $2}')"

echo -e "\n=== Application Status ==="
echo "API Status: $(supervisorctl status tool-creator-api | awk '{print $2}')"
echo "Worker Status: $(supervisorctl status tool-creator-worker | awk '{print $2}')"
echo "MySQL Status: $(systemctl is-active mysql)"
echo "Redis Status: $(systemctl is-active redis-server)"
echo "Nginx Status: $(systemctl is-active nginx)"

echo -e "\n=== Queue Status ==="
redis-cli llen tool_creator:queue:high
redis-cli llen tool_creator:queue:normal
redis-cli llen tool_creator:queue:low

echo -e "\n=== Recent Logs ==="
tail -n 5 /opt/tool-creator/logs/api.log
EOF
    
    chmod +x /opt/tool-creator/monitor.sh
    
    # Setup log rotation
    cat > /etc/logrotate.d/tool-creator <<EOF
/opt/tool-creator/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 www-data www-data
    postrotate
        supervisorctl restart tool-creator-api
        supervisorctl restart tool-creator-worker
    endscript
}
EOF
    
    log "Monitoring setup completed"
}

# Setup backup
setup_backup() {
    log "Setting up backup system..."
    
    # Create backup script
    cat > /opt/tool-creator/backup.sh <<EOF
#!/bin/bash

BACKUP_DIR="/opt/backups"
DATE=\$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p \$BACKUP_DIR

# Backup database
mysqldump -u tool_creator -p$DB_PASSWORD tool_creator > \$BACKUP_DIR/db_\$DATE.sql

# Backup application files
tar -czf \$BACKUP_DIR/app_\$DATE.tar.gz -C /opt tool-creator/storage

# Keep only last 7 days of backups
find \$BACKUP_DIR -name "*.sql" -mtime +7 -delete
find \$BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: \$DATE"
EOF
    
    chmod +x /opt/tool-creator/backup.sh
    
    # Setup daily backup cron
    echo "0 2 * * * /opt/tool-creator/backup.sh >> /opt/tool-creator/logs/backup.log 2>&1" | crontab -
    
    log "Backup system configured"
}

# Final setup
final_setup() {
    log "Performing final setup..."
    
    # Start services
    systemctl start supervisor
    systemctl enable supervisor
    
    # Start application
    supervisorctl start tool-creator-api
    supervisorctl start tool-creator-worker
    
    # Create systemd service for auto-start
    cat > /etc/systemd/system/tool-creator.service <<EOF
[Unit]
Description=Tool Creator Application
After=network.target mysql.service redis-server.service

[Service]
Type=forking
ExecStart=/usr/bin/supervisord -c /etc/supervisor/supervisord.conf
ExecStop=/usr/bin/supervisorctl shutdown
Restart=always

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl enable tool-creator.service
    
    log "Final setup completed"
}

# Display information
display_info() {
    echo -e "${GREEN}"
    echo "=========================================="
    echo "           DEPLOYMENT COMPLETED           "
    echo "=========================================="
    echo -e "${NC}"
    echo "🌐 Domain: https://$DOMAIN"
    echo "🔧 API Endpoint: https://$DOMAIN/api"
    echo "📊 Monitor: /opt/tool-creator/monitor.sh"
    echo "💾 Backup: /opt/tool-creator/backup.sh"
    echo ""
    echo "📁 Application Directory: /opt/tool-creator"
    echo "📝 Logs: /opt/tool-creator/logs/"
    echo "🗄️  Database: tool_creator"
    echo "🔴 Redis: localhost:6379"
    echo ""
    echo "🛠️  Useful Commands:"
    echo "  Check status: supervisorctl status"
    echo "  View logs: tail -f /opt/tool-creator/logs/api.log"
    echo "  Monitor: /opt/tool-creator/monitor.sh"
    echo "  Backup: /opt/tool-creator/backup.sh"
    echo ""
    echo "🔒 Security:"
    echo "  - Firewall enabled (UFW)"
    echo "  - Fail2ban installed"
    echo "  - SSL certificate configured"
    echo "  - Rate limiting enabled"
    echo ""
    echo -e "${YELLOW}⚠️  Don't forget to:${NC}"
    echo "  1. Update DNS records to point to this server"
    echo "  2. Test the application endpoints"
    echo "  3. Monitor system resources"
    echo "  4. Set up monitoring alerts"
    echo ""
}

# Main function
main() {
    echo -e "${BLUE}"
    echo "=========================================="
    echo "    VPS Single Server Deployment Script   "
    echo "=========================================="
    echo -e "${NC}"
    
    check_root
    update_system
    install_dependencies
    configure_firewall
    configure_mysql
    configure_redis
    install_go
    install_demucs
    setup_application
    build_application
    create_env_file
    configure_supervisor
    configure_nginx
    setup_ssl
    setup_monitoring
    setup_backup
    final_setup
    display_info
    
    log "Deployment completed successfully!"
}

# Run main function
main "$@" 