#!/bin/bash

# ========================================
# HETZNER SERVER SETUP SCRIPT
# ========================================
# Script này setup server Hetzner cho Tool Creator production

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 HETZNER SERVER SETUP FOR TOOL CREATOR${NC}"
echo "========================================"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Script này phải chạy với quyền root (sudo)${NC}"
    exit 1
fi

# Get server information
echo -e "${YELLOW}Nhập domain của bạn (e.g., toolcreator.com):${NC}"
read DOMAIN

echo -e "${YELLOW}Nhập email admin (cho SSL certificate):${NC}"
read ADMIN_EMAIL

echo -e "${YELLOW}Nhập SSH key public (để bảo mật):${NC}"
read SSH_PUBLIC_KEY

if [ -z "$DOMAIN" ] || [ -z "$ADMIN_EMAIL" ]; then
    echo -e "${RED}❌ Domain và email không được để trống${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}📋 Server Configuration:${NC}"
echo "Domain: $DOMAIN"
echo "Admin Email: $ADMIN_EMAIL"
echo "SSH Key: $SSH_PUBLIC_KEY"
echo ""

echo -e "${YELLOW}Bạn có muốn tiếp tục? (y/N):${NC}"
read -r response
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Setup cancelled."
    exit 0
fi

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ $1 completed successfully${NC}"
    else
        echo -e "${RED}❌ $1 failed${NC}"
        exit 1
    fi
}

# Step 1: System Update
echo ""
echo -e "${BLUE}🔧 Step 1: System Update${NC}"
echo "========================================"

echo -e "${YELLOW}📦 Updating system packages...${NC}"
apt update && apt upgrade -y
check_status "System update"

# Step 2: Install Required Packages
echo ""
echo -e "${BLUE}📦 Step 2: Install Required Packages${NC}"
echo "========================================"

echo -e "${YELLOW}📦 Installing required packages...${NC}"
apt install -y curl wget git nginx mysql-server redis-server supervisor certbot python3-certbot-nginx python3 python3-pip ffmpeg build-essential htop ufw fail2ban unzip zip
check_status "Package installation"

# Step 3: Install Go
echo ""
echo -e "${BLUE}🐹 Step 3: Install Go${NC}"
echo "========================================"

echo -e "${YELLOW}🐹 Installing Go...${NC}"
if ! command -v go &> /dev/null; then
    wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    source /etc/profile
    rm go1.23.linux-amd64.tar.gz
    echo -e "${GREEN}✅ Go installed successfully${NC}"
else
    echo -e "${GREEN}✅ Go already installed${NC}"
fi

# Step 4: Install Node.js
echo ""
echo -e "${BLUE}📦 Step 4: Install Node.js${NC}"
echo "========================================"

echo -e "${YELLOW}📦 Installing Node.js...${NC}"
if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
    apt install -y nodejs
    echo -e "${GREEN}✅ Node.js installed successfully${NC}"
else
    echo -e "${GREEN}✅ Node.js already installed${NC}"
fi

# Step 5: Security Setup
echo ""
echo -e "${BLUE}🔒 Step 5: Security Setup${NC}"
echo "========================================"

# Setup SSH key
echo -e "${YELLOW}🔑 Setting up SSH key...${NC}"
mkdir -p /root/.ssh
echo "$SSH_PUBLIC_KEY" >> /root/.ssh/authorized_keys
chmod 700 /root/.ssh
chmod 600 /root/.ssh/authorized_keys
echo -e "${GREEN}✅ SSH key setup completed${NC}"

# Setup firewall
echo -e "${YELLOW}🔥 Setting up firewall...${NC}"
ufw allow 22
ufw allow 80
ufw allow 443
ufw --force enable
echo -e "${GREEN}✅ Firewall setup completed${NC}"

# Setup fail2ban
echo -e "${YELLOW}🛡️ Setting up fail2ban...${NC}"
systemctl enable fail2ban
systemctl start fail2ban
echo -e "${GREEN}✅ Fail2ban setup completed${NC}"

# Step 6: Database Setup
echo ""
echo -e "${BLUE}🗄️ Step 6: Database Setup${NC}"
echo "========================================"

echo -e "${YELLOW}🗄️ Setting up MySQL...${NC}"
systemctl enable mysql
systemctl start mysql

# Secure MySQL installation
mysql_secure_installation
echo -e "${GREEN}✅ MySQL setup completed${NC}"

# Step 7: Redis Setup
echo ""
echo -e "${BLUE}🔴 Step 7: Redis Setup${NC}"
echo "========================================"

echo -e "${YELLOW}🔴 Setting up Redis...${NC}"
systemctl enable redis-server
systemctl start redis-server
echo -e "${GREEN}✅ Redis setup completed${NC}"

# Step 8: Nginx Setup
echo ""
echo -e "${BLUE}🌐 Step 8: Nginx Setup${NC}"
echo "========================================"

echo -e "${YELLOW}🌐 Setting up Nginx...${NC}"
systemctl enable nginx
systemctl start nginx
echo -e "${GREEN}✅ Nginx setup completed${NC}"

# Step 9: Create Application Directories
echo ""
echo -e "${BLUE}📁 Step 9: Create Application Directories${NC}"
echo "========================================"

echo -e "${YELLOW}📁 Creating application directories...${NC}"
mkdir -p /opt/tool-creator-backend
mkdir -p /var/www/tool-creator
mkdir -p /var/log/tool-creator
mkdir -p /backup/tool-creator
mkdir -p /app/storage
mkdir -p /app/data

# Set permissions
chown -R www-data:www-data /var/www/tool-creator
chown -R www-data:www-data /var/log/tool-creator
chown -R www-data:www-data /app/storage
chown -R www-data:www-data /app/data
chmod -R 755 /var/www/tool-creator
chmod -R 755 /var/log/tool-creator
chmod -R 755 /app/storage
chmod -R 755 /app/data

echo -e "${GREEN}✅ Application directories created${NC}"

# Step 10: SSL Certificate Setup
echo ""
echo -e "${BLUE}🔐 Step 10: SSL Certificate Setup${NC}"
echo "========================================"

echo -e "${YELLOW}🔐 Setting up SSL certificate...${NC}"
certbot certonly --standalone \
    --email $ADMIN_EMAIL \
    --agree-tos \
    --no-eff-email \
    -d $DOMAIN \
    -d www.$DOMAIN

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ SSL certificate obtained successfully${NC}"
else
    echo -e "${RED}❌ SSL certificate setup failed${NC}"
    echo -e "${YELLOW}Please check:${NC}"
    echo "1. Domain DNS is pointing to this server"
    echo "2. Port 80 is open and accessible"
    echo "3. Domain is not already using HTTPS"
fi

# Step 11: Performance Optimization
echo ""
echo -e "${BLUE}⚡ Step 11: Performance Optimization${NC}"
echo "========================================"

# Optimize MySQL
echo -e "${YELLOW}🗄️ Optimizing MySQL...${NC}"
cat > /etc/mysql/conf.d/tool-creator.cnf << 'EOF'
[mysqld]
innodb_buffer_pool_size = 2G
innodb_log_file_size = 256M
innodb_flush_log_at_trx_commit = 2
innodb_flush_method = O_DIRECT
max_connections = 200
query_cache_size = 64M
query_cache_type = 1
tmp_table_size = 64M
max_heap_table_size = 64M
EOF

systemctl restart mysql
echo -e "${GREEN}✅ MySQL optimization completed${NC}"

# Optimize Nginx
echo -e "${YELLOW}🌐 Optimizing Nginx...${NC}"
cat > /etc/nginx/conf.d/tool-creator-optimization.conf << 'EOF'
# Gzip compression
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_proxied expired no-cache no-store private must-revalidate auth;
gzip_types
    text/plain
    text/css
    text/xml
    text/javascript
    application/javascript
    application/xml+rss
    application/json;

# Client max body size for file uploads
client_max_body_size 100M;

# Proxy buffer settings
proxy_buffer_size 4k;
proxy_buffers 8 4k;
proxy_busy_buffers_size 8k;
EOF

systemctl reload nginx
echo -e "${GREEN}✅ Nginx optimization completed${NC}"

# Step 12: Monitoring Setup
echo ""
echo -e "${BLUE}📊 Step 12: Monitoring Setup${NC}"
echo "========================================"

# Create monitoring script
echo -e "${YELLOW}📊 Creating monitoring script...${NC}"
cat > /usr/local/bin/monitor-tool-creator.sh << 'EOF'
#!/bin/bash

# Check if services are running
services=("nginx" "mysql" "redis-server" "tool-creator-backend")

for service in "${services[@]}"; do
    if ! systemctl is-active --quiet $service; then
        echo "$(date): $service is down, restarting..." >> /var/log/tool-creator/monitor.log
        systemctl restart $service
    fi
done

# Check disk space
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 80 ]; then
    echo "$(date): Disk usage is high: ${DISK_USAGE}%" >> /var/log/tool-creator/monitor.log
fi

# Check memory usage
MEM_USAGE=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
if [ $MEM_USAGE -gt 80 ]; then
    echo "$(date): Memory usage is high: ${MEM_USAGE}%" >> /var/log/tool-creator/monitor.log
fi
EOF

chmod +x /usr/local/bin/monitor-tool-creator.sh

# Add to crontab
(crontab -l 2>/dev/null; echo "*/5 * * * * /usr/local/bin/monitor-tool-creator.sh") | crontab -

echo -e "${GREEN}✅ Monitoring setup completed${NC}"

# Step 13: Backup Setup
echo ""
echo -e "${BLUE}💾 Step 13: Backup Setup${NC}"
echo "========================================"

echo -e "${YELLOW}💾 Creating backup script...${NC}"
cat > /usr/local/bin/backup-tool-creator.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/tool-creator"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup database
mysqldump -u root -p tool_creator_prod > $BACKUP_DIR/db_backup_$DATE.sql

# Backup files
tar -czf $BACKUP_DIR/files_backup_$DATE.tar.gz /opt/tool-creator-backend/storage /var/www/tool-creator

# Clean old backups (keep 30 days)
find $BACKUP_DIR -name "*.sql" -mtime +30 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +30 -delete

echo "Backup completed: $DATE" >> /var/log/tool-creator/backup.log
EOF

chmod +x /usr/local/bin/backup-tool-creator.sh

# Add to crontab
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/backup-tool-creator.sh") | crontab -

echo -e "${GREEN}✅ Backup setup completed${NC}"

# Final Summary
echo ""
echo -e "${GREEN}🎉 HETZNER SERVER SETUP COMPLETED!${NC}"
echo ""
echo -e "${BLUE}📝 Server Information:${NC}"
echo "========================================"
echo "Domain: $DOMAIN"
echo "Admin Email: $ADMIN_EMAIL"
echo "Backend Directory: /opt/tool-creator-backend"
echo "Frontend Directory: /var/www/tool-creator"
echo "Log Directory: /var/log/tool-creator"
echo "Backup Directory: /backup/tool-creator"
echo ""
echo -e "${BLUE}🔧 Services Status:${NC}"
echo "========================================"
systemctl status nginx --no-pager
systemctl status mysql --no-pager
systemctl status redis-server --no-pager
echo ""
echo -e "${BLUE}🔗 Next Steps:${NC}"
echo "========================================"
echo "1. Upload your application code"
echo "2. Run deployment script: ./scripts/deploy-production.sh"
echo "3. Test all features"
echo "4. Setup monitoring and alerts"
echo ""
echo -e "${YELLOW}⚠️  Important:${NC}"
echo "- Keep your SSH keys secure"
echo "- Monitor server performance"
echo "- Regular backups are automated"
echo "- SSL certificate auto-renewal is enabled"
echo ""
echo -e "${GREEN}🚀 Your server is ready for deployment!${NC}" 