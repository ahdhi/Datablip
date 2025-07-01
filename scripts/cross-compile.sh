#!/bin/bash

set -e

PROJECT_NAME="datablip"
MAIN_PATH="cmd/datablip"
OUTPUT_DIR="bin"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build targets (OS/ARCH)
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "freebsd/amd64"
)

print_status() {
    echo -e "\033[0;34m[CROSS-COMPILE]\033[0m $1"
}

print_success() {
    echo -e "\033[0;32m[SUCCESS]\033[0m $1"
}

mkdir -p "$OUTPUT_DIR"

BUILD_LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.buildTime=$BUILD_TIME"

print_status "Cross-compiling $PROJECT_NAME for ${#TARGETS[@]} targets..."

for target in "${TARGETS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$target"
    
    output_name="$PROJECT_NAME-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name="$output_name.exe"
    fi
    
    output_path="$OUTPUT_DIR/$output_name"
    
    print_status "Building for $GOOS/$GOARCH..."
    
    if GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "$BUILD_LDFLAGS" -o "$output_path" "./$MAIN_PATH"; then
        # Set permissions for Unix-like systems
        if [ "$GOOS" != "windows" ]; then
            chmod 755 "$output_path"
        fi
        
        BINARY_SIZE=$(ls -lh "$output_path" | awk '{print $5}')
        print_success "$GOOS/$GOARCH -> $output_path ($BINARY_SIZE)"
    else
        echo -e "\033[0;31m[ERROR]\033[0m Failed to build for $GOOS/$GOARCH"
    fi
done

print_success "Cross-compilation completed. Binaries in $OUTPUT_DIR/"
ls -la "$OUTPUT_DIR"/