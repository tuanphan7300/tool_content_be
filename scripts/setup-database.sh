#!/bin/bash

# ========================================
# DATABASE SETUP SCRIPT FOR PRODUCTION
# ========================================
# This script sets up MySQL database for production deployment

# Configuration
DB_NAME="tool_creator_prod"
DB_USER="tool_creator_prod"
DB_HOST="localhost"
DB_PORT="3306"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🗄️  Setting up production database..."

# Check if MySQL is running
if ! systemctl is-active --quiet mysql; then
    echo -e "${RED}❌ MySQL is not running. Please start MySQL first.${NC}"
    exit 1
fi

# Get database password from user
echo -e "${YELLOW}Enter the database password for user '$DB_USER':${NC}"
read -s DB_PASSWORD
echo ""

# Create database
echo "📦 Creating database '$DB_NAME'..."
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database '$DB_NAME' created successfully${NC}"
else
    echo -e "${RED}❌ Failed to create database${NC}"
    exit 1
fi

# Create user
echo "👤 Creating user '$DB_USER'..."
mysql -u root -p -e "CREATE USER IF NOT EXISTS '$DB_USER'@'$DB_HOST' IDENTIFIED BY '$DB_PASSWORD';"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ User '$DB_USER' created successfully${NC}"
else
    echo -e "${RED}❌ Failed to create user${NC}"
    exit 1
fi

# Grant permissions
echo "🔐 Granting permissions to user '$DB_USER'..."
mysql -u root -p -e "GRANT ALL PRIVILEGES ON $DB_NAME.* TO '$DB_USER'@'$DB_HOST';"
mysql -u root -p -e "FLUSH PRIVILEGES;"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Permissions granted successfully${NC}"
else
    echo -e "${RED}❌ Failed to grant permissions${NC}"
    exit 1
fi

# Run migrations
echo "🔄 Running database migrations..."
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_init_database.sql
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Initial migration completed${NC}"
else
    echo -e "${RED}❌ Failed to run initial migration${NC}"
    exit 1
fi

# Run soft delete migration
echo "🔄 Running soft delete migration..."
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_soft_delete.sql
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Soft delete migration completed${NC}"
else
    echo -e "${RED}❌ Failed to run soft delete migration${NC}"
    exit 1
fi

# Run other migrations
echo "🔄 Running additional migrations..."
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_google_oauth.sql
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_service_config.sql
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_add_tiktok_optimizer_fields.sql
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME < migration_fix_units_used.sql

echo -e "${GREEN}✅ All migrations completed successfully${NC}"

# Test connection
echo "🧪 Testing database connection..."
mysql -u $DB_USER -p$DB_PASSWORD $DB_NAME -e "SELECT 'Connection successful' as status;"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database connection test passed${NC}"
else
    echo -e "${RED}❌ Database connection test failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 Database setup completed successfully!${NC}"
echo ""
echo "📝 Database configuration for .env file:"
echo "========================================"
echo "DB_HOST=$DB_HOST"
echo "DB_PORT=$DB_PORT"
echo "DB_USER=$DB_USER"
echo "DB_PASSWORD=$DB_PASSWORD"
echo "DB_NAME=$DB_NAME"
echo "========================================"
echo ""
echo "⚠️  IMPORTANT:"
echo "- Keep the database password secure"
echo "- Backup the database regularly"
echo "- Monitor database performance" 