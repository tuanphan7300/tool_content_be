#!/bin/bash

# System monitoring script cho server 4 CPU, 16GB RAM
# Monitor CPU, RAM, disk usage và active processes

LOG_FILE="/var/log/system-monitor.log"
ALERT_THRESHOLD_CPU=80
ALERT_THRESHOLD_RAM=85
ALERT_THRESHOLD_DISK=85

echo "=== System Monitor - $(date) ===" >> "$LOG_FILE"

# CPU Usage
CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | awk -F'%' '{print $1}')
echo "CPU Usage: ${CPU_USAGE}%" >> "$LOG_FILE"

# RAM Usage 
RAM_USAGE=$(free | grep Mem | awk '{printf("%.1f"), $3/$2 * 100.0}')
echo "RAM Usage: ${RAM_USAGE}%" >> "$LOG_FILE"

# Disk Usage
DISK_USAGE=$(df -h / | tail -1 | awk '{print $5}' | sed 's/%//')
echo "Disk Usage: ${DISK_USAGE}%" >> "$LOG_FILE"

# Active Go processes (API + Worker)
GO_PROCESSES=$(pgrep -f "creator-tool-backend" | wc -l)
echo "Active Go processes: $GO_PROCESSES" >> "$LOG_FILE"

# Queue status (nếu có Redis)
if command -v redis-cli &> /dev/null; then
    QUEUE_SIZE=$(redis-cli llen audio_processing_queue 2>/dev/null || echo "N/A")
    echo "Queue size: $QUEUE_SIZE" >> "$LOG_FILE"
fi

# Memory usage của processes chính
echo "Top memory consumers:" >> "$LOG_FILE"
ps aux --sort=-%mem | head -6 >> "$LOG_FILE"

# Alerts
if (( $(echo "$CPU_USAGE > $ALERT_THRESHOLD_CPU" | bc -l) )); then
    echo "ALERT: High CPU usage: ${CPU_USAGE}%" >> "$LOG_FILE"
fi

if (( $(echo "$RAM_USAGE > $ALERT_THRESHOLD_RAM" | bc -l) )); then
    echo "ALERT: High RAM usage: ${RAM_USAGE}%" >> "$LOG_FILE"
fi

if [ "$DISK_USAGE" -gt $ALERT_THRESHOLD_DISK ]; then
    echo "ALERT: High Disk usage: ${DISK_USAGE}%" >> "$LOG_FILE"
fi

echo "=================================" >> "$LOG_FILE"
