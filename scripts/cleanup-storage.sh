#!/bin/bash

# Storage cleanup script để tối ưu 160GB NVMe SSD
# Chạy hàng ngày để dọn dẹp files tạm và cũ

echo "Starting storage cleanup..."

STORAGE_DIR="/app/storage"
LOG_FILE="/var/log/cleanup-storage.log"

# Tạo log entry
echo "$(date): Starting cleanup" >> "$LOG_FILE"

# Dọn dẹp files tạm cũ hơn 24h
find "$STORAGE_DIR" -type f -name "*.tmp" -mtime +1 -delete 2>/dev/null
echo "$(date): Cleaned temp files" >> "$LOG_FILE"

# Dọn dẹp thư mục processing cũ hơn 2 ngày  
find "$STORAGE_DIR" -type d -name "*_processing_*" -mtime +2 -exec rm -rf {} + 2>/dev/null
echo "$(date): Cleaned processing directories" >> "$LOG_FILE"

# Dọn dẹp videos đã xử lý cũ hơn 7 ngày (trừ kết quả cuối)
find "$STORAGE_DIR" -type f \( -name "*.mp4" -o -name "*.avi" -o -name "*.mov" \) -mtime +7 ! -name "*_final_*" -delete 2>/dev/null
echo "$(date): Cleaned old videos" >> "$LOG_FILE"

# Dọn dẹp audio files tạm cũ hơn 3 ngày
find "$STORAGE_DIR" -type f -name "*.wav" -mtime +3 -delete 2>/dev/null
find "$STORAGE_DIR" -type f -name "*.mp3" -path "*/audio/*" -mtime +3 -delete 2>/dev/null
echo "$(date): Cleaned audio files" >> "$LOG_FILE"

# Kiểm tra dung lượng còn lại
DISK_USAGE=$(df -h "$STORAGE_DIR" | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 80 ]; then
    echo "$(date): WARNING - Disk usage is ${DISK_USAGE}%" >> "$LOG_FILE"
    # Dọn dẹp thêm nếu disk > 80%
    find "$STORAGE_DIR" -type f -mtime +1 ! -name "*_final_*" -delete 2>/dev/null
    echo "$(date): Emergency cleanup performed" >> "$LOG_FILE"
fi

echo "$(date): Cleanup completed. Disk usage: ${DISK_USAGE}%" >> "$LOG_FILE"
echo "Storage cleanup completed. Disk usage: ${DISK_USAGE}%"
