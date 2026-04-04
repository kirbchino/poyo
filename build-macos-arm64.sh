#!/bin/bash
# Poyo Build Script for macOS ARM64

set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="./build"
BINARY_NAME="poyo"

# Go proxy for faster downloads in China
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.org

echo "🔨 Building Poyo for macOS ARM64..."
echo "Version: $VERSION"
echo "GOPROXY: $GOPROXY"

# Clean build directory
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR

# Build with optimizations
GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-s -w -X main.Version=$VERSION" \
    -o $OUTPUT_DIR/$BINARY_NAME \
    ./cmd/poyo

# Show build info
echo ""
echo "✅ Build complete!"
echo "Binary: $OUTPUT_DIR/$BINARY_NAME"
file $OUTPUT_DIR/$BINARY_NAME
ls -lh $OUTPUT_DIR/$BINARY_NAME

echo ""
echo "To install:"
echo "  cp $OUTPUT_DIR/$BINARY_NAME /usr/local/bin/"
