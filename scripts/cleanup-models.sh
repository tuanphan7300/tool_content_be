#!/bin/bash

# Script để cleanup pretrained models trước khi push lên production
# Loại bỏ các file macOS-specific không cần thiết

set -e

echo "🧹 Cleaning up pretrained models for production..."

# Xóa các file macOS-specific
echo "📁 Removing macOS-specific files..."
find pretrained_models/ -name "._*" -delete
find pretrained_models/ -name ".DS_Store" -delete
find pretrained_models/ -name ".probe" -delete

# Kiểm tra các file còn lại
echo "✅ Remaining files:"
find pretrained_models/ -type f | sort

# Kiểm tra các file model cần thiết
required_files=(
    "pretrained_models/2stems/model.data-00000-of-00001"
    "pretrained_models/2stems/model.index"
    "pretrained_models/2stems/model.meta"
    "pretrained_models/2stems/checkpoint"
)

echo ""
echo "🔍 Checking required model files..."
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo "✅ $file ($size)"
    else
        echo "❌ $file (MISSING)"
        exit 1
    fi
done

echo ""
echo "🎯 Models are ready for production deployment!"
echo "📝 Next steps:"
echo "1. git add pretrained_models/"
echo "2. git commit -m 'Add pretrained models for production'"
echo "3. git push"
echo "4. Deploy to production server" 