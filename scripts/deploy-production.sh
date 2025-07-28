#!/bin/bash

# ========================================
# MASTER PRODUCTION DEPLOYMENT SCRIPT
# ========================================
# This script deploys the entire Tool Creator system to production

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 TOOL CREATOR PRODUCTION DEPLOYMENT${NC}"
echo "========================================"
echo "This script will deploy the entire system to production"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Configuration
BACKEND_DIR="/opt/tool-creator-backend"
FRONTEND_DIR="/var/www/tool-creator"
DOMAIN=""

# Get domain
echo -e "${YELLOW}Enter your domain name (e.g., yourdomain.com):${NC}"
read DOMAIN

if [ -z "$DOMAIN" ]; then
    echo -e "${RED}❌ Domain name is required${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}📋 Deployment Configuration:${NC}"
echo "Domain: $DOMAIN"
echo "Backend Directory: $BACKEND_DIR"
echo "Frontend Directory: $FRONTEND_DIR"
echo ""

echo -e "${YELLOW}Do you want to continue? (y/N):${NC}"
read -r response
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled."
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

# Step 1: System Preparation
echo ""
echo -e "${BLUE}🔧 Step 1: System Preparation${NC}"
echo "========================================"

# Update system
echo -e "${YELLOW}📦 Updating system packages...${NC}"
apt update && apt upgrade -y
check_status "System update"

# Install required packages
echo -e "${YELLOW}📦 Installing required packages...${NC}"
apt install -y curl wget git nginx mysql-server redis-server supervisor certbot python3-certbot-nginx python3 python3-pip ffmpeg build-essential
check_status "Package installation"

# Install Go
echo -e "${YELLOW}🐹 Installing Go...${NC}"
if ! command -v go &> /dev/null; then
    wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    source /etc/profile
    rm go1.23.linux-amd64.tar.gz
fi
check_status "Go installation"

# Step 2: Backend Setup
echo ""
echo -e "${BLUE}🔧 Step 2: Backend Setup${NC}"
echo "========================================"

# Create backend directory
echo -e "${YELLOW}📁 Creating backend directory...${NC}"
mkdir -p $BACKEND_DIR
cd $BACKEND_DIR

# Clone or copy backend code
if [ -d ".git" ]; then
    echo -e "${YELLOW}🔄 Updating existing backend...${NC}"
    git pull
else
    echo -e "${YELLOW}📥 Cloning backend repository...${NC}"
    # Replace with your actual repository URL
    git clone https://github.com/your-username/tool-creator.git .
fi
check_status "Backend code setup"

# Generate secrets
echo -e "${YELLOW}🔐 Generating secrets...${NC}"
chmod +x scripts/generate-secrets.sh
./scripts/generate-secrets.sh
check_status "Secret generation"

# Setup database
echo -e "${YELLOW}🗄️  Setting up database...${NC}"
chmod +x scripts/setup-database.sh
./scripts/setup-database.sh
check_status "Database setup"

# Download AI models
echo -e "${YELLOW}🤖 Downloading AI models...${NC}"
chmod +x scripts/download-models.sh
./scripts/download-models.sh
check_status "AI models download"

# Build backend
echo -e "${YELLOW}🔨 Building backend...${NC}"
go mod download
go build -o main .
check_status "Backend build"

# Setup backend service
echo -e "${YELLOW}⚙️  Setting up backend service...${NC}"
chmod +x scripts/setup-backend-service.sh
echo "$BACKEND_DIR" | ./scripts/setup-backend-service.sh
check_status "Backend service setup"

# Step 3: Frontend Setup
echo ""
echo -e "${BLUE}🎨 Step 3: Frontend Setup${NC}"
echo "========================================"

# Create frontend directory
echo -e "${YELLOW}📁 Creating frontend directory...${NC}"
mkdir -p $FRONTEND_DIR

# Clone or copy frontend code
FRONTEND_SOURCE="/opt/tool-creator-frontend"
if [ -d "$FRONTEND_SOURCE" ]; then
    echo -e "${YELLOW}🔄 Updating existing frontend...${NC}"
    cd $FRONTEND_SOURCE
    git pull
else
    echo -e "${YELLOW}📥 Cloning frontend repository...${NC}"
    mkdir -p $FRONTEND_SOURCE
    cd $FRONTEND_SOURCE
    # Replace with your actual repository URL
    git clone https://github.com/your-username/tool-creator-frontend.git .
fi
check_status "Frontend code setup"

# Setup frontend environment
echo -e "${YELLOW}⚙️  Setting up frontend environment...${NC}"
cp env.production.example .env.production
sed -i "s/yourdomain.com/$DOMAIN/g" .env.production
check_status "Frontend environment setup"

# Build frontend
echo -e "${YELLOW}🔨 Building frontend...${NC}"
chmod +x scripts/build-production.sh
./scripts/build-production.sh
check_status "Frontend build"

# Deploy frontend
echo -e "${YELLOW}📤 Deploying frontend...${NC}"
cp -r dist/* $FRONTEND_DIR/
chown -R www-data:www-data $FRONTEND_DIR
check_status "Frontend deployment"

# Step 4: Nginx Setup
echo ""
echo -e "${BLUE}🌐 Step 4: Nginx Setup${NC}"
echo "========================================"

# Copy nginx configuration
echo -e "${YELLOW}📝 Setting up Nginx configuration...${NC}"
cp nginx/tool-creator.conf /etc/nginx/sites-available/tool-creator
sed -i "s/yourdomain.com/$DOMAIN/g" /etc/nginx/sites-available/tool-creator
ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
check_status "Nginx configuration"

# Test nginx configuration
echo -e "${YELLOW}🧪 Testing Nginx configuration...${NC}"
nginx -t
check_status "Nginx configuration test"

# Step 5: SSL Setup
echo ""
echo -e "${BLUE}🔒 Step 5: SSL Setup${NC}"
echo "========================================"

# Setup SSL certificate
echo -e "${YELLOW}🔐 Setting up SSL certificate...${NC}"
chmod +x scripts/setup-ssl.sh
echo "$DOMAIN" | ./scripts/setup-ssl.sh
check_status "SSL setup"

# Step 6: Final Configuration
echo ""
echo -e "${BLUE}⚙️  Step 6: Final Configuration${NC}"
echo "========================================"

# Start services
echo -e "${YELLOW}🚀 Starting services...${NC}"
systemctl start nginx
systemctl start tool-creator-backend
systemctl enable nginx
systemctl enable tool-creator-backend
check_status "Service startup"

# Setup firewall
echo -e "${YELLOW}🔥 Setting up firewall...${NC}"
ufw allow 22
ufw allow 80
ufw allow 443
ufw --force enable
check_status "Firewall setup"

# Create backup script
echo -e "${YELLOW}💾 Creating backup script...${NC}"
cat > /usr/local/bin/backup-tool-creator.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/tool-creator"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup database
mysqldump -u tool_creator_prod -p tool_creator_prod > $BACKUP_DIR/db_backup_$DATE.sql

# Backup files
tar -czf $BACKUP_DIR/files_backup_$DATE.tar.gz /opt/tool-creator-backend/storage /var/www/tool-creator

# Clean old backups (keep 30 days)
find $BACKUP_DIR -name "*.sql" -mtime +30 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +30 -delete

echo "Backup completed: $DATE"
EOF

chmod +x /usr/local/bin/backup-tool-creator.sh

# Add backup to crontab
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/backup-tool-creator.sh") | crontab -

check_status "Backup setup"

# Step 7: Verification
echo ""
echo -e "${BLUE}✅ Step 7: Verification${NC}"
echo "========================================"

# Test health endpoint
echo -e "${YELLOW}🧪 Testing health endpoint...${NC}"
sleep 5
if curl -s -f https://$DOMAIN/health > /dev/null; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${RED}❌ Health check failed${NC}"
fi

# Test frontend
echo -e "${YELLOW}🧪 Testing frontend...${NC}"
if curl -s -f https://$DOMAIN > /dev/null; then
    echo -e "${GREEN}✅ Frontend is accessible${NC}"
else
    echo -e "${RED}❌ Frontend is not accessible${NC}"
fi

# Show service status
echo -e "${YELLOW}📊 Service status:${NC}"
systemctl status nginx --no-pager
systemctl status tool-creator-backend --no-pager

echo ""
echo -e "${GREEN}🎉 PRODUCTION DEPLOYMENT COMPLETED!${NC}"
echo ""
echo -e "${BLUE}📝 Deployment Summary:${NC}"
echo "========================================"
echo "Domain: https://$DOMAIN"
echo "Backend: $BACKEND_DIR"
echo "Frontend: $FRONTEND_DIR"
echo "Database: tool_creator_prod"
echo "SSL: Let's Encrypt (auto-renewal enabled)"
echo ""
echo -e "${BLUE}🔧 Useful Commands:${NC}"
echo "Backend logs: journalctl -u tool-creator-backend -f"
echo "Nginx logs: tail -f /var/log/nginx/tool-creator.error.log"
echo "Backup: /usr/local/bin/backup-tool-creator.sh"
echo "Monitor: /usr/local/bin/monitor-tool-creator.sh"
echo ""
echo -e "${YELLOW}⚠️  Next Steps:${NC}"
echo "1. Test all features on https://$DOMAIN"
echo "2. Monitor system performance"
echo "3. Setup monitoring and alerts"
echo "4. Document any issues found"
echo ""
echo -e "${GREEN}🚀 Your Tool Creator is now live!${NC}" 