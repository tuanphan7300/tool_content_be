#!/bin/bash

# Script kiểm tra và fix vấn đề với Demucs
# Sử dụng: ./check_demucs.sh

echo "🔍 Kiểm tra cấu hình Demucs..."

# Kiểm tra Python
echo "🐍 Kiểm tra Python..."
if command -v python3 &> /dev/null; then
    echo "✅ Python3: $(python3 --version)"
else
    echo "❌ Python3 chưa được cài đặt"
    exit 1
fi

# Kiểm tra pip
echo "📦 Kiểm tra pip..."
if command -v pip3 &> /dev/null; then
    echo "✅ pip3: $(pip3 --version)"
else
    echo "❌ pip3 chưa được cài đặt"
    exit 1
fi

# Kiểm tra Demucs
echo "🎵 Kiểm tra Demucs..."
DEMUCS_PATHS=(
    "/Users/phantuan/Library/Frameworks/Python.framework/Versions/3.11/bin/demucs"
    "/Users/phantuan/Library/Python/3.9/bin/demucs"
    "/usr/local/bin/demucs"
    "$(which demucs)"
)

DEMUCS_FOUND=false
for path in "${DEMUCS_PATHS[@]}"; do
    if [ -f "$path" ] && [ -x "$path" ]; then
        echo "✅ Demucs found: $path"
        DEMUCS_PATH="$path"
        DEMUCS_FOUND=true
        break
    fi
done

if [ "$DEMUCS_FOUND" = false ]; then
    echo "❌ Demucs chưa được cài đặt"
    echo ""
    echo "📝 Cài đặt Demucs:"
    echo "   pip3 install -U demucs"
    echo ""
    echo "   Hoặc cài đặt từ source:"
    echo "   pip3 install -U git+https://github.com/facebookresearch/demucs#egg=demucs"
    exit 1
fi

# Kiểm tra FFmpeg
echo "🎬 Kiểm tra FFmpeg..."
if command -v ffmpeg &> /dev/null; then
    echo "✅ FFmpeg: $(ffmpeg -version | head -n1)"
else
    echo "❌ FFmpeg chưa được cài đặt"
    echo "   macOS: brew install ffmpeg"
    echo "   Ubuntu: sudo apt install ffmpeg"
    exit 1
fi

# Kiểm tra FFprobe
echo "🔍 Kiểm tra FFprobe..."
if command -v ffprobe &> /dev/null; then
    echo "✅ FFprobe: $(ffprobe -version | head -n1)"
else
    echo "❌ FFprobe chưa được cài đặt"
    exit 1
fi

# Kiểm tra thư mục pretrained models
echo "📁 Kiểm tra pretrained models..."
MODELS_DIR="pretrained_models/2stems"
if [ -d "$MODELS_DIR" ]; then
    echo "✅ Thư mục models tồn tại: $MODELS_DIR"
    
    # Kiểm tra các file cần thiết
    REQUIRED_FILES=("model.data-00000-of-00001" "model.index" "model.meta" "checkpoint")
    for file in "${REQUIRED_FILES[@]}"; do
        if [ -f "$MODELS_DIR/$file" ]; then
            echo "✅ $file tồn tại"
        else
            echo "❌ $file thiếu"
        fi
    done
else
    echo "❌ Thư mục models không tồn tại: $MODELS_DIR"
    echo "   Tạo thư mục: mkdir -p $MODELS_DIR"
fi

# Test Demucs với file audio mẫu
echo ""
echo "🧪 Test Demucs..."
TEMP_AUDIO="temp_test_audio.wav"

# Tạo file audio test 1 giây
ffmpeg -f lavfi -i "sine=frequency=440:duration=1" -y "$TEMP_AUDIO" >/dev/null 2>&1

if [ -f "$TEMP_AUDIO" ]; then
    echo "✅ Tạo file audio test thành công"
    
    # Test Demucs
    echo "🎵 Chạy test Demucs..."
    OUTPUT_DIR="temp_demucs_test"
    
    if "$DEMUCS_PATH" -n htdemucs --two-stems vocals -o "$OUTPUT_DIR" "$TEMP_AUDIO" >/dev/null 2>&1; then
        echo "✅ Demucs test thành công"
        
        # Kiểm tra output
        if [ -d "$OUTPUT_DIR/htdemucs" ]; then
            echo "✅ Output directory được tạo"
            
            # Tìm file output
            for subdir in "$OUTPUT_DIR/htdemucs"/*; do
                if [ -d "$subdir" ]; then
                    if [ -f "$subdir/vocals.wav" ] && [ -f "$subdir/no_vocals.wav" ]; then
                        echo "✅ Files output được tạo:"
                        echo "   - $(basename "$subdir")/vocals.wav"
                        echo "   - $(basename "$subdir")/no_vocals.wav"
                    fi
                    break
                fi
            done
        fi
    else
        echo "❌ Demucs test thất bại"
        echo "   Có thể cần tải pretrained models"
    fi
    
    # Cleanup
    rm -rf "$OUTPUT_DIR"
    rm -f "$TEMP_AUDIO"
else
    echo "❌ Không thể tạo file audio test"
fi

# Kiểm tra cấu hình trong code
echo ""
echo "🔧 Kiểm tra cấu hình code..."
if grep -q "demucs" service/voice_service.go; then
    echo "✅ Demucs được sử dụng trong voice_service.go"
    
    # Hiển thị đường dẫn Demucs trong code
    echo "📝 Đường dẫn Demucs trong code:"
    grep -n "demucs" service/voice_service.go
else
    echo "❌ Không tìm thấy Demucs trong voice_service.go"
fi

echo ""
echo "📋 Recommendations:"

# Đưa ra gợi ý dựa trên kết quả kiểm tra
if [ "$DEMUCS_FOUND" = true ]; then
    echo "✅ Demucs đã được cài đặt và hoạt động"
    echo "   Đường dẫn: $DEMUCS_PATH"
    
    # Kiểm tra xem đường dẫn trong code có đúng không
    if grep -q "$DEMUCS_PATH" service/voice_service.go; then
        echo "✅ Đường dẫn trong code khớp với hệ thống"
    else
        echo "⚠️  Cần cập nhật đường dẫn Demucs trong code"
        echo "   Thay đổi trong service/voice_service.go:"
        echo "   $DEMUCS_PATH"
    fi
fi

echo ""
echo "📚 Tài liệu tham khảo:"
echo "   - Demucs GitHub: https://github.com/facebookresearch/demucs"
echo "   - Installation: pip3 install -U demucs"
echo "   - Models: https://github.com/facebookresearch/demucs#pretrained-models"
echo ""
echo "✅ Kiểm tra hoàn tất!" 