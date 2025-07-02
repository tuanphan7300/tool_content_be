#!/bin/bash

# Test script cho Queue System
# Chạy script này sau khi đã setup Redis và khởi động ứng dụng

BASE_URL="http://localhost:8888"
TOKEN="your_jwt_token_here" # Thay bằng token thật

echo "🧪 Testing Queue System..."
echo "=========================="

# Test 1: Kiểm tra queue status
echo "1. Testing Queue Status..."
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/status" | jq '.'

echo -e "\n"

# Test 2: Kiểm tra worker status
echo "2. Testing Worker Status..."
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/worker/status" | jq '.'

echo -e "\n"

# Test 3: Tạo test job (giả lập)
echo "3. Creating Test Job..."
JOB_ID=$(uuidgen)
echo "Test Job ID: $JOB_ID"

# Test 4: Kiểm tra job status
echo "4. Testing Job Status..."
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/job/$JOB_ID/status" | jq '.'

echo -e "\n"

# Test 5: Kiểm tra job result
echo "5. Testing Job Result..."
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/job/$JOB_ID/result" | jq '.'

echo -e "\n"

# Test 6: Test worker control
echo "6. Testing Worker Control..."
echo "Stopping worker..."
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/worker/stop" | jq '.'

sleep 2

echo "Starting worker..."
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/queue/worker/start" | jq '.'

echo -e "\n"

echo "✅ Queue System Test Completed!"
echo "================================"

# Hướng dẫn sử dụng
echo -e "\n📖 Usage Instructions:"
echo "1. Start Redis: docker-compose -f docker-compose.redis.yml up -d"
echo "2. Start App: go run main.go"
echo "3. Monitor Queue: http://localhost:8081 (Redis Commander)"
echo "4. API Docs: See QUEUE_SYSTEM_README.md" 