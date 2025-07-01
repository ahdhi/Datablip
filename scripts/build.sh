#!/bin/bash

set -e

# Configuration
PROJECT_NAME="datablip"
OUTPUT_DIR="bin"
MAIN_PATH="cmd/datablip"
LDFLAGS=""
TAGS=""
VERBOSE=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[BUILD]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --ldflags)
            LDFLAGS="$2"
            shift 2
            ;;
        --tags)
            TAGS="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose       Enable verbose output"
            echo "  -o, --output        Specify output directory (default: bin)"
            echo "  --ldflags          Additional ldflags for go build"
            echo "  --tags             Build tags"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Get version info
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build ldflags
BUILD_LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.buildTime=$BUILD_TIME"
if [ -n "$LDFLAGS" ]; then
    BUILD_LDFLAGS="$BUILD_LDFLAGS $LDFLAGS"
fi

print_status "Building $PROJECT_NAME..."
print_status "Version: $VERSION"
print_status "Commit: $COMMIT"
print_status "Output: $OUTPUT_DIR/$PROJECT_NAME"

# Build command
BUILD_CMD="go build"
if [ "$VERBOSE" = true ]; then
    BUILD_CMD="$BUILD_CMD -v"
fi
if [ -n "$TAGS" ]; then
    BUILD_CMD="$BUILD_CMD -tags '$TAGS'"
fi
BUILD_CMD="$BUILD_CMD -ldflags '$BUILD_LDFLAGS' -o $OUTPUT_DIR/$PROJECT_NAME ./$MAIN_PATH"

if [ "$VERBOSE" = true ]; then
    print_status "Command: $BUILD_CMD"
fi

# Execute build
if eval $BUILD_CMD; then
    print_success "Build completed successfully"
    
    # Run post-build script
    if [ -f "scripts/post-build.sh" ]; then
        print_status "Running post-build script..."
        bash scripts/post-build.sh --build-dir "$OUTPUT_DIR" --binary "$PROJECT_NAME"
    fi
    
    # Display binary info
    if [ -f "$OUTPUT_DIR/$PROJECT_NAME" ]; then
        BINARY_SIZE=$(ls -lh "$OUTPUT_DIR/$PROJECT_NAME" | awk '{print $5}')
        print_success "Binary created: $OUTPUT_DIR/$PROJECT_NAME ($BINARY_SIZE)"
        
        # Test if binary is executable
        if [ -x "$OUTPUT_DIR/$PROJECT_NAME" ]; then
            print_success "Binary is executable"
        else
            print_error "Binary is not executable"
            exit 1
        fi
    fi
else
    print_error "Build failed"
    exit 1
fi