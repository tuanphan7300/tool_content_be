#!/bin/bash

# Quick VPS Deployment Script
# Chạy nhanh để deploy lên VPS

set -e

echo "🚀 Quick VPS Deployment for Tool Creator"
echo "========================================"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root"
   exit 1
fi

# Configuration
read -p "🌐 Enter your domain (e.g., api.yourdomain.com): " DOMAIN
read -p "🔑 Enter database password: " DB_PASSWORD
read -p "🔐 Enter JWT secret: " JWT_SECRET
read -p "🤖 Enter OpenAI API key: " OPENAI_API_KEY
read -p "🤖 Enter Gemini API key: " GEMINI_API_KEY

echo "📦 Installing dependencies..."

# Update system
apt update && apt upgrade -y

# Install packages
apt install -y curl wget git nginx mysql-server redis-server supervisor certbot python3-certbot-nginx python3 python3-pip ffmpeg build-essential

# Install Go
GO_VERSION="1.23"
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Configure MySQL
mysql -e "CREATE DATABASE IF NOT EXISTS tool_creator CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -e "CREATE USER IF NOT EXISTS 'tool_creator'@'localhost' IDENTIFIED BY '$DB_PASSWORD';"
mysql -e "GRANT ALL PRIVILEGES ON tool_creator.* TO 'tool_creator'@'localhost';"
mysql -e "FLUSH PRIVILEGES;"

# Install Demucs
python3 -m venv /opt/demucs
source /opt/demucs/bin/activate
pip install demucs

# Setup application
mkdir -p /opt/tool-creator
cd /opt/tool-creator

# Copy application files (assuming current directory)
cp -r . /opt/tool-creator/
mkdir -p storage pretrained_models logs

# Create .env file
cat > /opt/tool-creator/.env <<EOF
DB_HOST=localhost
DB_PORT=3306
DB_USER=tool_creator
DB_PASSWORD=$DB_PASSWORD
DB_NAME=tool_creator
REDIS_HOST=localhost
REDIS_PORT=6379
OPENAI_API_KEY=$OPENAI_API_KEY
GEMINI_API_KEY=$GEMINI_API_KEY
JWT_SECRET=$JWT_SECRET
PORT=8888
ENVIRONMENT=production
STORAGE_PATH=/opt/tool-creator/storage
MODELS_PATH=/opt/tool-creator/pretrained_models
QUEUE_WORKERS=2
QUEUE_TIMEOUT=1800
EOF

# Build application
cd /opt/tool-creator
export PATH=$PATH:/usr/local/go/bin
go mod download
go build -o tool-creator-api .

# Configure Supervisor
cat > /etc/supervisor/conf.d/tool-creator.conf <<EOF
[program:tool-creator-api]
command=/opt/tool-creator/tool-creator-api
directory=/opt/tool-creator
user=www-data
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/opt/tool-creator/logs/api.log

[program:tool-creator-worker]
command=/opt/tool-creator/tool-creator-api --worker
directory=/opt/tool-creator
user=www-data
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/opt/tool-creator/logs/worker.log
EOF

# Configure Nginx
cat > /etc/nginx/sites-available/tool-creator <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;
    
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    
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

# Enable site
rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/

# Set permissions
chown -R www-data:www-data /opt/tool-creator
chmod 600 /opt/tool-creator/.env

# Start services
systemctl start supervisor
systemctl enable supervisor
supervisorctl reread
supervisorctl update
supervisorctl start tool-creator-api
supervisorctl start tool-creator-worker

# Get SSL certificate
certbot --nginx -d $DOMAIN --non-interactive --agree-tos --email admin@$DOMAIN

# Setup auto-renewal
echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -

# Create monitoring script
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

echo ""
echo "✅ Deployment completed!"
echo ""
echo "🌐 Your API is available at: https://$DOMAIN/api"
echo "📊 Monitor: /opt/tool-creator/monitor.sh"
echo "📝 Logs: /opt/tool-creator/logs/"
echo ""
echo "🛠️ Useful commands:"
echo "  Check status: supervisorctl status"
echo "  View logs: tail -f /opt/tool-creator/logs/api.log"
echo "  Monitor: /opt/tool-creator/monitor.sh"
echo ""
echo "⚠️ Don't forget to:"
echo "  1. Update DNS records to point to this server"
echo "  2. Test the API endpoints"
echo "  3. Monitor system resources" 