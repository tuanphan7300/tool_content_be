#!/bin/bash

# Script để chạy migration cho bảng user_process_status
echo "Running migration for user_process_status table..."

# Đọc thông tin database từ env.example hoặc config
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"3306"}
DB_USER=${DB_USER:-"root"}
DB_PASSWORD=${DB_PASSWORD:-"Root@123"}
DB_NAME=${DB_NAME:-"tool"}

echo "Connecting to database: $DB_HOST:$DB_PORT/$DB_NAME"

# Chạy migration SQL
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < migration_add_process_status.sql

if [ $? -eq 0 ]; then
    echo "✅ Migration completed successfully!"
    echo "✅ Table user_process_status has been created"
    echo "✅ Anti-spam feature is now active"
else
    echo "❌ Migration failed!"
    exit 1
fi 