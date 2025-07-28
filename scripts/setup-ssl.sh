#!/bin/bash

# ========================================
# SSL CERTIFICATE SETUP SCRIPT
# ========================================
# This script sets up SSL certificate using Let's Encrypt

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔒 Setting up SSL Certificate with Let's Encrypt${NC}"
echo "========================================"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Get domain from user
echo -e "${YELLOW}Enter your domain name (e.g., yourdomain.com):${NC}"
read DOMAIN

if [ -z "$DOMAIN" ]; then
    echo -e "${RED}❌ Domain name is required${NC}"
    exit 1
fi

# Check if domain is reachable
echo -e "${YELLOW}🔍 Checking if domain is reachable...${NC}"
if ! nslookup $DOMAIN > /dev/null 2>&1; then
    echo -e "${RED}❌ Domain $DOMAIN is not reachable. Please check DNS settings.${NC}"
    exit 1
fi

# Install certbot if not installed
if ! command -v certbot &> /dev/null; then
    echo -e "${YELLOW}📦 Installing certbot...${NC}"
    apt update
    apt install -y certbot python3-certbot-nginx
fi

# Stop nginx temporarily
echo -e "${YELLOW}🛑 Stopping Nginx temporarily...${NC}"
systemctl stop nginx

# Create temporary nginx config for certbot
echo -e "${YELLOW}📝 Creating temporary Nginx configuration...${NC}"
cat > /etc/nginx/sites-available/temp-certbot << EOF
server {
    listen 80;
    server_name $DOMAIN www.$DOMAIN;
    
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}
EOF

# Enable temporary config
ln -sf /etc/nginx/sites-available/temp-certbot /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

# Create web root directory
mkdir -p /var/www/html

# Start nginx
echo -e "${YELLOW}🚀 Starting Nginx...${NC}"
systemctl start nginx

# Obtain SSL certificate
echo -e "${YELLOW}🔐 Obtaining SSL certificate for $DOMAIN...${NC}"
certbot certonly --webroot \
    --webroot-path=/var/www/html \
    --email admin@$DOMAIN \
    --agree-tos \
    --no-eff-email \
    -d $DOMAIN \
    -d www.$DOMAIN

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ SSL certificate obtained successfully!${NC}"
else
    echo -e "${RED}❌ Failed to obtain SSL certificate${NC}"
    echo -e "${YELLOW}Please check:${NC}"
    echo "1. Domain DNS is pointing to this server"
    echo "2. Port 80 is open and accessible"
    echo "3. Domain is not already using HTTPS"
    exit 1
fi

# Stop nginx again
systemctl stop nginx

# Remove temporary config
rm -f /etc/nginx/sites-enabled/temp-certbot

# Update main nginx config with domain
echo -e "${YELLOW}📝 Updating Nginx configuration...${NC}"
sed -i "s/yourdomain.com/$DOMAIN/g" /etc/nginx/sites-available/tool-creator

# Enable main config
ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/

# Test nginx configuration
echo -e "${YELLOW}🧪 Testing Nginx configuration...${NC}"
nginx -t

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Nginx configuration is valid${NC}"
else
    echo -e "${RED}❌ Nginx configuration has errors${NC}"
    exit 1
fi

# Start nginx
echo -e "${YELLOW}🚀 Starting Nginx with SSL...${NC}"
systemctl start nginx

# Test SSL certificate
echo -e "${YELLOW}🧪 Testing SSL certificate...${NC}"
if curl -s -o /dev/null -w "%{http_code}" https://$DOMAIN | grep -q "200\|301\|302"; then
    echo -e "${GREEN}✅ SSL certificate is working!${NC}"
else
    echo -e "${RED}❌ SSL certificate test failed${NC}"
fi

# Setup auto-renewal
echo -e "${YELLOW}🔄 Setting up auto-renewal...${NC}"
(crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/certbot renew --quiet") | crontab -

# Create renewal script
cat > /usr/local/bin/renew-ssl.sh << 'EOF'
#!/bin/bash
certbot renew --quiet
if [ $? -eq 0 ]; then
    systemctl reload nginx
    echo "$(date): SSL certificate renewed successfully" >> /var/log/ssl-renewal.log
else
    echo "$(date): SSL certificate renewal failed" >> /var/log/ssl-renewal.log
fi
EOF

chmod +x /usr/local/bin/renew-ssl.sh

echo ""
echo -e "${GREEN}🎉 SSL setup completed successfully!${NC}"
echo ""
echo -e "${BLUE}📝 Summary:${NC}"
echo "========================================"
echo "Domain: $DOMAIN"
echo "SSL Certificate: /etc/letsencrypt/live/$DOMAIN/"
echo "Auto-renewal: Enabled (daily at 12:00)"
echo "Nginx Config: /etc/nginx/sites-available/tool-creator"
echo ""
echo -e "${BLUE}🔗 Test your site:${NC}"
echo "https://$DOMAIN"
echo ""
echo -e "${YELLOW}⚠️  Important:${NC}"
echo "1. Update your frontend .env.production with: VITE_API_URL=https://$DOMAIN/api"
echo "2. Update your backend .env with the correct domain"
echo "3. Test all features after deployment" 