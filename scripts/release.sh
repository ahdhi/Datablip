#!/bin/bash

set -e

PROJECT_NAME="datablip"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
RELEASE_DIR="release"

print_status() {
    echo -e "\033[0;34m[RELEASE]\033[0m $1"
}

print_success() {
    echo -e "\033[0;32m[SUCCESS]\033[0m $1"
}

# Clean and create release directory
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

print_status "Creating release packages for version $VERSION..."

# Cross-compile first
bash scripts/cross-compile.sh

# Package each binary
for binary in bin/${PROJECT_NAME}-*; do
    if [ -f "$binary" ]; then
        basename=$(basename "$binary")
        
        # Determine platform from filename
        if [[ $basename =~ ${PROJECT_NAME}-(.*) ]]; then
            platform="${BASH_REMATCH[1]}"
            
            # Create package directory
            package_dir="$RELEASE_DIR/${PROJECT_NAME}-${VERSION}-${platform}"
            mkdir -p "$package_dir"
            
            # Copy binary
            cp "$binary" "$package_dir/"
            
            # Copy documentation
            cp README.md "$package_dir/" 2>/dev/null || true
            cp docs/* "$package_dir/" 2>/dev/null || true
            
            # Create archive
            if [[ $platform == *"windows"* ]]; then
                # Create ZIP for Windows
                (cd "$RELEASE_DIR" && zip -r "${PROJECT_NAME}-${VERSION}-${platform}.zip" "${PROJECT_NAME}-${VERSION}-${platform}")
            else
                # Create tar.gz for Unix-like systems
                (cd "$RELEASE_DIR" && tar -czf "${PROJECT_NAME}-${VERSION}-${platform}.tar.gz" "${PROJECT_NAME}-${VERSION}-${platform}")
            fi
            
            # Remove temporary directory
            rm -rf "$package_dir"
            
            print_success "Created package for $platform"
        fi
    fi
done

print_success "Release packages created in $RELEASE_DIR/"
ls -la "$RELEASE_DIR"/