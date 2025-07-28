#!/bin/bash

# ========================================
# BACKEND SYSTEMD SERVICE SETUP SCRIPT
# ========================================
# This script sets up systemd service for the backend

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Setting up Backend Systemd Service${NC}"
echo "========================================"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Get application directory
echo -e "${YELLOW}Enter the full path to your backend application:${NC}"
read APP_DIR

if [ -z "$APP_DIR" ]; then
    echo -e "${RED}❌ Application directory is required${NC}"
    exit 1
fi

if [ ! -d "$APP_DIR" ]; then
    echo -e "${RED}❌ Directory $APP_DIR does not exist${NC}"
    exit 1
fi

# Get user to run the service
echo -e "${YELLOW}Enter the user to run the service (default: www-data):${NC}"
read SERVICE_USER
SERVICE_USER=${SERVICE_USER:-www-data}

# Check if user exists
if ! id "$SERVICE_USER" &>/dev/null; then
    echo -e "${YELLOW}Creating user $SERVICE_USER...${NC}"
    useradd -r -s /bin/false $SERVICE_USER
fi

# Create systemd service file
echo -e "${YELLOW}📝 Creating systemd service file...${NC}"
cat > /etc/systemd/system/tool-creator-backend.service << EOF
[Unit]
Description=Tool Creator Backend Service
After=network.target mysql.service redis.service
Wants=mysql.service redis.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$APP_DIR
Environment=PATH=$APP_DIR:/usr/local/go/bin:/usr/bin:/bin
Environment=GIN_MODE=release
ExecStart=$APP_DIR/main
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=tool-creator-backend

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$APP_DIR/storage $APP_DIR/data

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Environment variables
EnvironmentFile=$APP_DIR/.env

[Install]
WantedBy=multi-user.target
EOF

# Set proper permissions
chown root:root /etc/systemd/system/tool-creator-backend.service
chmod 644 /etc/systemd/system/tool-creator-backend.service

# Create log directory
mkdir -p /var/log/tool-creator
chown $SERVICE_USER:$SERVICE_USER /var/log/tool-creator

# Set application directory permissions
chown -R $SERVICE_USER:$SERVICE_USER $APP_DIR
chmod -R 755 $APP_DIR

# Create storage directories
mkdir -p $APP_DIR/storage
mkdir -p $APP_DIR/data
chown -R $SERVICE_USER:$SERVICE_USER $APP_DIR/storage
chown -R $SERVICE_USER:$SERVICE_USER $APP_DIR/data
chmod -R 755 $APP_DIR/storage
chmod -R 755 $APP_DIR/data

# Reload systemd
echo -e "${YELLOW}🔄 Reloading systemd...${NC}"
systemctl daemon-reload

# Enable service
echo -e "${YELLOW}✅ Enabling service...${NC}"
systemctl enable tool-creator-backend

# Test service
echo -e "${YELLOW}🧪 Testing service configuration...${NC}"
systemctl status tool-creator-backend --no-pager

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Service configuration is valid${NC}"
else
    echo -e "${RED}❌ Service configuration has errors${NC}"
    exit 1
fi

# Start service
echo -e "${YELLOW}🚀 Starting service...${NC}"
systemctl start tool-creator-backend

# Check service status
sleep 3
if systemctl is-active --quiet tool-creator-backend; then
    echo -e "${GREEN}✅ Service started successfully!${NC}"
else
    echo -e "${RED}❌ Service failed to start${NC}"
    echo -e "${YELLOW}Checking logs...${NC}"
    journalctl -u tool-creator-backend -n 20 --no-pager
    exit 1
fi

# Create log rotation
echo -e "${YELLOW}📝 Setting up log rotation...${NC}"
cat > /etc/logrotate.d/tool-creator-backend << EOF
/var/log/tool-creator/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 $SERVICE_USER $SERVICE_USER
    postrotate
        systemctl reload tool-creator-backend
    endscript
}
EOF

# Create monitoring script
echo -e "${YELLOW}📊 Creating monitoring script...${NC}"
cat > /usr/local/bin/monitor-tool-creator.sh << 'EOF'
#!/bin/bash

# Check if service is running
if ! systemctl is-active --quiet tool-creator-backend; then
    echo "$(date): Tool Creator Backend service is down, restarting..." >> /var/log/tool-creator/monitor.log
    systemctl restart tool-creator-backend
fi

# Check health endpoint
if ! curl -s -f http://localhost:8888/health > /dev/null; then
    echo "$(date): Health check failed, restarting service..." >> /var/log/tool-creator/monitor.log
    systemctl restart tool-creator-backend
fi
EOF

chmod +x /usr/local/bin/monitor-tool-creator.sh

# Add monitoring to crontab
(crontab -l 2>/dev/null; echo "*/5 * * * * /usr/local/bin/monitor-tool-creator.sh") | crontab -

echo ""
echo -e "${GREEN}🎉 Backend service setup completed successfully!${NC}"
echo ""
echo -e "${BLUE}📝 Service Information:${NC}"
echo "========================================"
echo "Service Name: tool-creator-backend"
echo "User: $SERVICE_USER"
echo "Working Directory: $APP_DIR"
echo "Logs: journalctl -u tool-creator-backend"
echo "Status: systemctl status tool-creator-backend"
echo ""
echo -e "${BLUE}🔧 Useful Commands:${NC}"
echo "Start: systemctl start tool-creator-backend"
echo "Stop: systemctl stop tool-creator-backend"
echo "Restart: systemctl restart tool-creator-backend"
echo "Status: systemctl status tool-creator-backend"
echo "Logs: journalctl -u tool-creator-backend -f"
echo ""
echo -e "${YELLOW}⚠️  Next steps:${NC}"
echo "1. Test the API endpoints"
echo "2. Configure Nginx reverse proxy"
echo "3. Setup SSL certificate"
echo "4. Test the complete application" 