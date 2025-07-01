#!/bin/bash

# Post-build script for Go program file permissions
# This script sets appropriate permissions for Go binaries to execute in any environment

set -e  # Exit on any error

# Configuration
BUILD_DIR="./bin"
BINARY_NAME=""  # Leave empty to auto-detect, or specify your binary name
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    if [ "$VERBOSE" = true ]; then
        echo -e "${color}[INFO]${NC} $message"
    fi
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Function to detect Go binary
detect_binary() {
    if [ -n "$BINARY_NAME" ]; then
        echo "$BUILD_DIR/$BINARY_NAME"
        return
    fi
    
    # Look for Go binaries in build directory
    local binaries=($(find "$BUILD_DIR" -type f -executable 2>/dev/null || true))
    
    if [ ${#binaries[@]} -eq 0 ]; then
        # If no executables found, look for files without extension (common for Go binaries)
        binaries=($(find "$BUILD_DIR" -type f ! -name "*.*" 2>/dev/null || true))
    fi
    
    if [ ${#binaries[@]} -eq 1 ]; then
        echo "${binaries[0]}"
    elif [ ${#binaries[@]} -gt 1 ]; then
        print_error "Multiple binaries found. Please specify BINARY_NAME in the script:"
        printf '%s\n' "${binaries[@]}"
        exit 1
    else
        print_error "No binary found in $BUILD_DIR"
        exit 1
    fi
}

# Function to set file permissions
set_permissions() {
    local binary_path=$1
    
    if [ ! -f "$binary_path" ]; then
        print_error "Binary not found: $binary_path"
        exit 1
    fi
    
    print_status "$BLUE" "Setting permissions for: $binary_path"
    
    # Set permissions: owner (rwx), group (rx), others (rx)
    # This is 755 in octal notation
    chmod 755 "$binary_path"
    
    # Verify permissions were set correctly
    local perms=$(stat -c "%a" "$binary_path" 2>/dev/null || stat -f "%A" "$binary_path" 2>/dev/null)
    
    if [ "$perms" = "755" ]; then
        print_success "Permissions set successfully: $perms (rwxr-xr-x)"
    else
        print_error "Failed to set permissions correctly. Current permissions: $perms"
        exit 1
    fi
    
    # Additional checks for different environments
    print_status "$YELLOW" "Performing environment compatibility checks..."
    
    # Check if binary is actually executable
    if [ -x "$binary_path" ]; then
        print_status "$GREEN" "✓ Binary is executable"
    else
        print_error "✗ Binary is not executable after permission change"
        exit 1
    fi
    
    # Check file type (should be executable)
    local file_type=$(file "$binary_path" 2>/dev/null || echo "unknown")
    if echo "$file_type" | grep -q "executable"; then
        print_status "$GREEN" "✓ File type confirmed as executable"
    else
        print_status "$YELLOW" "⚠ File type check: $file_type"
    fi
}

# Function to create build directory if it doesn't exist
ensure_build_dir() {
    if [ ! -d "$BUILD_DIR" ]; then
        print_status "$YELLOW" "Creating build directory: $BUILD_DIR"
        mkdir -p "$BUILD_DIR"
    fi
}

# Function to set permissions for all binaries in build directory
set_all_permissions() {
    print_status "$BLUE" "Scanning for binaries in $BUILD_DIR..."
    
    local count=0
    while IFS= read -r -d '' binary; do
        set_permissions "$binary"
        ((count++))
    done < <(find "$BUILD_DIR" -type f -executable -print0 2>/dev/null || true)
    
    if [ $count -eq 0 ]; then
        # Fallback: look for files without extensions
        while IFS= read -r -d '' binary; do
            if file "$binary" | grep -q "executable"; then
                set_permissions "$binary"
                ((count++))
            fi
        done < <(find "$BUILD_DIR" -type f ! -name "*.*" -print0 2>/dev/null || true)
    fi
    
    if [ $count -eq 0 ]; then
        print_error "No binaries found to set permissions for"
        exit 1
    fi
    
    print_success "Set permissions for $count binary/binaries"
}

# Main execution
main() {
    print_status "$BLUE" "Starting post-build permission setup..."
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -d|--build-dir)
                BUILD_DIR="$2"
                shift 2
                ;;
            -b|--binary)
                BINARY_NAME="$2"
                shift 2
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  -v, --verbose       Enable verbose output"
                echo "  -d, --build-dir     Specify build directory (default: ./bin)"
                echo "  -b, --binary        Specify binary name (auto-detect if not provided)"
                echo "  -h, --help          Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    ensure_build_dir
    
    if [ -n "$BINARY_NAME" ]; then
        # Set permissions for specific binary
        set_permissions "$BUILD_DIR/$BINARY_NAME"
    else
        # Set permissions for all binaries
        set_all_permissions
    fi
    
    print_success "Post-build permission setup completed successfully!"
}

# Run main function with all arguments
main "$@"