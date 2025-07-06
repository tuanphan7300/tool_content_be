#!/bin/bash

echo "=== Creating Test TTS File ==="

# Use the specific directory
LATEST_DIR="storage/1751703675222606000_1-1"

# Create TTS directory if it doesn't exist
TTS_DIR="$LATEST_DIR/tts"
mkdir -p "$TTS_DIR"

# Create a simple test TTS file using FFmpeg (sine wave)
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
TTS_FILE="$TTS_DIR/tts_${TIMESTAMP}.mp3"

echo "Creating test TTS file: $TTS_FILE"

# Generate a 10-second sine wave as test TTS
ffmpeg -f lavfi -i "sine=frequency=440:duration=10" \
    -c:a libmp3lame -b:a 128k \
    -y "$TTS_FILE"

if [ $? -eq 0 ]; then
    echo "✅ Test TTS file created successfully!"
    ls -lh "$TTS_FILE"
else
    echo "❌ Failed to create test TTS file"
    exit 1
fi

echo "=== Test TTS Creation Complete ===" 