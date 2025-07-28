# 🚀 HETZNER DEPLOYMENT GUIDE

## 📋 **BƯỚC 1: THUÊ SERVER HETZNER**

### **1.1 Đăng ký tài khoản Hetzner**
- Truy cập: https://www.hetzner.com/
- Đăng ký tài khoản mới
- Verify email và thông tin

### **1.2 Thuê Server (Cloud CX21)**
- **Location**: Frankfurt (FSN1) hoặc Nuremberg (NBG1)
- **CPU**: 4 vCPU
- **RAM**: 8GB
- **Storage**: 160GB NVMe SSD
- **Network**: 1Gbps
- **OS**: Ubuntu 22.04 LTS
- **Price**: ~€15-20/tháng

### **1.3 Cấu hình ban đầu**
- Tạo SSH key pair
- Ghi nhớ IP address
- Ghi nhớ root password

## 🔧 **BƯỚC 2: SETUP SERVER**

### **2.1 Kết nối SSH**
```bash
ssh root@YOUR_SERVER_IP
```

### **2.2 Chạy setup script**
```bash
# Download setup script
wget https://raw.githubusercontent.com/your-repo/tool-creator/main/scripts/hetzner-setup.sh
chmod +x hetzner-setup.sh
./hetzner-setup.sh
```

### **2.3 Thông tin cần chuẩn bị**
- **Domain**: toolcreator.com
- **Admin Email**: admin@toolcreator.com
- **SSH Public Key**: ssh-rsa AAAAB3NzaC1yc2E...

## 🌐 **BƯỚC 3: DOMAIN & DNS**

### **3.1 Mua domain**
- Mua domain từ Namecheap, GoDaddy, hoặc Hetzner
- Domain: toolcreator.com

### **3.2 Cấu hình DNS**
```
Type: A
Name: @
Value: YOUR_SERVER_IP
TTL: 300

Type: A
Name: www
Value: YOUR_SERVER_IP
TTL: 300
```

### **3.3 Chờ DNS propagate**
- Thời gian: 5-30 phút
- Test: `nslookup toolcreator.com`

## 📤 **BƯỚC 4: UPLOAD CODE**

### **4.1 Upload backend**
```bash
# Từ máy local
cd tool_content_be
scp -r . root@YOUR_SERVER_IP:/opt/tool-creator-backend/
```

### **4.2 Upload frontend**
```bash
# Build frontend
cd tool_content_fe
npm run build:seo

# Upload
scp -r dist/* root@YOUR_SERVER_IP:/var/www/tool-creator/
```

### **4.3 Set permissions**
```bash
# Trên server
chown -R www-data:www-data /opt/tool-creator-backend
chown -R www-data:www-data /var/www/tool-creator
chmod -R 755 /opt/tool-creator-backend
chmod -R 755 /var/www/tool-creator
```

## ⚙️ **BƯỚC 5: CONFIGURATION**

### **5.1 Backend Environment**
```bash
cd /opt/tool-creator-backend
cp env.production.example .env
nano .env
```

**Cập nhật các giá trị:**
```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=tool_creator_prod
DB_PASSWORD=YOUR_DB_PASSWORD
DB_NAME=tool_creator_prod

# Domain
DOMAIN=toolcreator.com

# JWT
JWTACCESSKEY=YOUR_JWT_SECRET

# AI Keys
OPENAI_API_KEY=YOUR_OPENAI_KEY
GEMINI_API_KEY=YOUR_GEMINI_KEY

# Google OAuth
GOOGLE_CLIENT_ID=YOUR_GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET=YOUR_GOOGLE_CLIENT_SECRET
```

### **5.2 Frontend Environment**
```bash
cd /var/www/tool-creator
nano .env.production
```

**Cập nhật:**
```env
VITE_API_URL=https://toolcreator.com/api
VITE_APP_NAME=Tool Creator
VITE_GOOGLE_CLIENT_ID=YOUR_GOOGLE_CLIENT_ID
```

### **5.3 Nginx Configuration**
```bash
cd /opt/tool-creator-backend
cp nginx/tool-creator.conf /etc/nginx/sites-available/tool-creator
sed -i "s/yourdomain.com/toolcreator.com/g" /etc/nginx/sites-available/tool-creator
ln -sf /etc/nginx/sites-available/tool-creator /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx
```

## 🚀 **BƯỚC 6: DEPLOYMENT**

### **6.1 Setup Database**
```bash
cd /opt/tool-creator-backend
chmod +x scripts/setup-database.sh
./scripts/setup-database.sh
```

### **6.2 Download AI Models**
```bash
chmod +x scripts/download-models.sh
./scripts/download-models.sh
```

### **6.3 Build Backend**
```bash
go mod download
go build -o main .
```

### **6.4 Setup Backend Service**
```bash
chmod +x scripts/setup-backend-service.sh
echo "/opt/tool-creator-backend" | ./scripts/setup-backend-service.sh
```

### **6.5 Start Services**
```bash
systemctl start tool-creator-backend
systemctl enable tool-creator-backend
systemctl status tool-creator-backend
```

## ✅ **BƯỚC 7: TESTING**

### **7.1 Test Backend**
```bash
curl -I https://toolcreator.com/health
curl -I https://toolcreator.com/ping
```

### **7.2 Test Frontend**
```bash
curl -I https://toolcreator.com
```

### **7.3 Test SSL**
```bash
openssl s_client -connect toolcreator.com:443 -servername toolcreator.com
```

### **7.4 Test Database**
```bash
mysql -u tool_creator_prod -p tool_creator_prod -e "SELECT 'Connection successful' as status;"
```

## 🔍 **BƯỚC 8: SEO & ANALYTICS**

### **8.1 SEO Setup**
```bash
cd /var/www/tool-creator
chmod +x scripts/build-seo.sh
./scripts/build-seo.sh
```

### **8.2 Google Search Console**
- Truy cập: https://search.google.com/search-console
- Add property: https://toolcreator.com
- Verify ownership
- Submit sitemap: https://toolcreator.com/sitemap.xml

### **8.3 Google Analytics**
- Tạo GA4 property
- Add tracking code vào frontend
- Setup goals và conversions

## 📊 **BƯỚC 9: MONITORING**

### **9.1 Check Services**
```bash
systemctl status nginx
systemctl status mysql
systemctl status redis-server
systemctl status tool-creator-backend
```

### **9.2 Check Logs**
```bash
journalctl -u tool-creator-backend -f
tail -f /var/log/nginx/tool-creator.error.log
tail -f /var/log/tool-creator/monitor.log
```

### **9.3 Performance Monitoring**
```bash
htop
df -h
free -h
```

## 🔧 **BƯỚC 10: MAINTENANCE**

### **10.1 Regular Updates**
```bash
apt update && apt upgrade -y
systemctl restart nginx
systemctl restart tool-creator-backend
```

### **10.2 Backup Verification**
```bash
ls -la /backup/tool-creator/
tail -f /var/log/tool-creator/backup.log
```

### **10.3 SSL Renewal**
```bash
certbot renew --dry-run
```

## 🚨 **TROUBLESHOOTING**

### **Issue: Website not accessible**
```bash
# Check nginx
systemctl status nginx
nginx -t

# Check firewall
ufw status

# Check ports
netstat -tlnp | grep :80
netstat -tlnp | grep :443
```

### **Issue: Backend not working**
```bash
# Check service
systemctl status tool-creator-backend

# Check logs
journalctl -u tool-creator-backend -n 50

# Check environment
cat /opt/tool-creator-backend/.env
```

### **Issue: Database connection failed**
```bash
# Check MySQL
systemctl status mysql

# Test connection
mysql -u tool_creator_prod -p

# Check database
mysql -u tool_creator_prod -p -e "SHOW DATABASES;"
```

### **Issue: SSL certificate expired**
```bash
# Renew certificate
certbot renew

# Reload nginx
systemctl reload nginx
```

## 📞 **SUPPORT & RESOURCES**

### **Hetzner Support**
- **Documentation**: https://docs.hetzner.com/
- **Support**: https://www.hetzner.com/support
- **Status**: https://status.hetzner.com/

### **Useful Commands**
```bash
# Server info
uname -a
cat /etc/os-release
free -h
df -h

# Service management
systemctl list-units --type=service --state=running
systemctl list-units --type=service --state=failed

# Network
ip addr show
netstat -tlnp
ss -tlnp

# Security
ufw status
fail2ban-client status
```

## 🎉 **DEPLOYMENT COMPLETE!**

Sau khi hoàn thành tất cả các bước trên, website của bạn sẽ:
- ✅ Chạy trên https://toolcreator.com
- ✅ Có SSL certificate tự động
- ✅ Có monitoring và backup
- ✅ Tối ưu SEO
- ✅ Sẵn sàng cho production

**Website URL**: https://toolcreator.com
**Admin Panel**: https://toolcreator.com/account
**API Health**: https://toolcreator.com/health 