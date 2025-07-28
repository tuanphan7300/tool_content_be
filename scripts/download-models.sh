#!/bin/bash

# ========================================
# AI MODELS DOWNLOAD SCRIPT
# ========================================
# This script downloads required AI models for production deployment

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🤖 Downloading AI Models for Production${NC}"
echo "========================================"

# Create directories
echo -e "${YELLOW}📁 Creating model directories...${NC}"
mkdir -p pretrained_models/2stems
mkdir -p data

# Check if models already exist
if [ -f "pretrained_models/2stems/model.data-00000-of-00001" ]; then
    echo -e "${GREEN}✅ Models already exist. Skipping download.${NC}"
    echo "If you want to re-download, delete the pretrained_models directory first."
    exit 0
fi

echo -e "${YELLOW}📥 Downloading Demucs HTDemucs 2-stems model...${NC}"
echo "This may take 30-60 minutes depending on your internet connection."

# Download Demucs model files
cd pretrained_models/2stems

# Model files URLs (these are example URLs - you need to get the actual URLs)
echo -e "${YELLOW}Downloading model files...${NC}"

# Method 1: Try using demucs command if available
if command -v demucs &> /dev/null; then
    echo -e "${BLUE}Using demucs command to download models...${NC}"
    demucs --download
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Models downloaded successfully using demucs command${NC}"
    else
        echo -e "${RED}❌ Failed to download using demucs command${NC}"
        echo -e "${YELLOW}Trying manual download...${NC}"
    fi
fi

# Method 2: Manual download (if demucs command fails)
if [ ! -f "model.data-00000-of-00001" ]; then
    echo -e "${YELLOW}Manual download required.${NC}"
    echo -e "${RED}⚠️  IMPORTANT: You need to manually download the HTDemucs 2-stems model.${NC}"
    echo ""
    echo "Please follow these steps:"
    echo "1. Visit: https://github.com/facebookresearch/demucs"
    echo "2. Download the HTDemucs 2-stems model"
    echo "3. Extract the files to: pretrained_models/2stems/"
    echo ""
    echo "Required files:"
    echo "- model.data-00000-of-00001 (~75MB)"
    echo "- model.index (~5KB)"
    echo "- model.meta (~787KB)"
    echo "- checkpoint (~67B)"
    echo ""
    echo "Or run this command to download:"
    echo "python3 -m demucs --download"
    echo ""
    exit 1
fi

cd ../..

# Verify models
echo -e "${YELLOW}🔍 Verifying downloaded models...${NC}"
required_files=(
    "pretrained_models/2stems/model.data-00000-of-00001"
    "pretrained_models/2stems/model.index"
    "pretrained_models/2stems/model.meta"
    "pretrained_models/2stems/checkpoint"
)

all_files_exist=true
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ $file${NC}"
    else
        echo -e "${RED}❌ $file (missing)${NC}"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = true ]; then
    echo ""
    echo -e "${GREEN}🎉 All AI models downloaded successfully!${NC}"
    echo ""
    echo "📊 Model sizes:"
    ls -lh pretrained_models/2stems/
    echo ""
    echo "✅ Your system is ready for AI processing!"
else
    echo ""
    echo -e "${RED}❌ Some model files are missing. Please download them manually.${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}📝 Next steps:${NC}"
echo "1. Test the models with a sample video"
echo "2. Monitor system performance"
echo "3. Set up monitoring and alerts" 