#!/bin/bash

# Script để setup GPT translation service
echo "Setting up GPT Translation Service..."

# Đọc thông tin database từ env
source .env

# Chạy SQL script
echo "Adding GPT translation service to database..."
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < add_gpt_translation_service.sql

if [ $? -eq 0 ]; then
    echo "✅ GPT Translation Service setup completed successfully!"
    echo ""
    echo "📋 Summary:"
    echo "- Service name: gpt_translation"
    echo "- Model: gpt-4o-mini"
    echo "- Price: $0.00015 per token"
    echo "- Status: Active"
    echo ""
    echo "🔄 To switch between Gemini and GPT:"
    echo "1. Gemini (default): UPDATE service_config SET is_active = 1 WHERE service_name = 'gemini_translation';"
    echo "2. GPT: UPDATE service_config SET is_active = 1 WHERE service_name = 'gpt_translation';"
else
    echo "❌ Failed to setup GPT Translation Service"
    exit 1
fi 