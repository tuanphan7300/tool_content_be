#!/bin/bash

# Script để setup TikTok Optimizer Service Config
# Chạy script này để thêm service config và pricing cho TikTok Optimizer

set -e

echo "🔧 Setting up TikTok Optimizer Service Config..."

# Kiểm tra xem có file SQL không
if [ ! -f "./scripts/setup-tiktok-service-config.sql" ]; then
    echo "❌ Error: setup-tiktok-service-config.sql not found!"
    exit 1
fi

# Kiểm tra xem có docker-compose không
if [ -f "./docker-compose.yml" ]; then
    echo "🐳 Using Docker Compose..."
    
    # Chạy SQL script trong MySQL container
    echo "📝 Executing SQL script..."
    docker-compose exec -T db mysql -u root -pRoot@123 tool < ./scripts/setup-tiktok-service-config.sql
    
    echo "✅ TikTok Optimizer Service Config setup completed!"
    echo ""
    echo "📊 Verification:"
    docker-compose exec -T db mysql -u root -pRoot@123 tool -e "
    SELECT 'Service Config' as type, service_type, service_name, is_active 
    FROM service_config 
    WHERE service_type = 'tiktok-optimizer';
    
    SELECT 'Service Pricing' as type, service_name, pricing_type, price_per_unit, currency 
    FROM service_pricing 
    WHERE service_name = 'tiktok-optimizer';
    "
else
    echo "💻 Using local MySQL..."
    
    # Kiểm tra xem có MySQL command line không
    if ! command -v mysql &> /dev/null; then
        echo "❌ Error: MySQL command line not found!"
        echo "Please install MySQL client or use Docker Compose"
        exit 1
    fi
    
    # Chạy SQL script với local MySQL
    echo "📝 Executing SQL script..."
    mysql -u root -p tool < ./scripts/setup-tiktok-service-config.sql
    
    echo "✅ TikTok Optimizer Service Config setup completed!"
    echo ""
    echo "📊 Verification:"
    mysql -u root -p tool -e "
    SELECT 'Service Config' as type, service_type, service_name, is_active 
    FROM service_config 
    WHERE service_type = 'tiktok-optimizer';
    
    SELECT 'Service Pricing' as type, service_name, pricing_type, price_per_unit, currency 
    FROM service_pricing 
    WHERE service_name = 'tiktok-optimizer';
    "
fi

echo ""
echo "🎉 TikTok Optimizer Service Config setup completed successfully!"
echo "🚀 You can now use the TikTok Optimizer API with full service config integration!" 