# DataBlip - High-Performance Multi-Threaded Downloader

A fast, reliable, multi-threaded file downloader written in Go that splits large files into chunks for concurrent downloading, providing detailed progress tracking and robust error handling.

## Features

- **Multi-threaded downloading** - Splits files into chunks for parallel download
- **Real-time progress tracking** - Shows overall and per-chunk progress with speed indicators
- **Robust error handling** - Automatic retries and verification
- **Resume capability** - Handles interrupted downloads gracefully
- **Cross-platform support** - Works on Linux, macOS, Windows, and FreeBSD
- **Docker support** - Containerized deployment option
- **Configurable timeouts** - Customizable connection and read timeouts
- **File verification** - Ensures download integrity

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/govind1331/Datablip.git
cd datablip

# Build using Make
make build

# Or build using the build script
bash scripts/build.sh

# Install system-wide
make install
```

### Cross-Platform Binaries

```bash
# Build for all supported platforms
make cross-compile

# Create release packages
make release
```

### Docker

```bash
# Build Docker image
make docker

# Or manually
docker build -t datablip:latest -f build/docker/Dockerfile .
```

## Usage

### Basic Usage

```bash
# Download a file
./bin/datablip -url "https://example.com/largefile.zip" -output "largefile.zip"

# Specify number of chunks
./bin/datablip -url "https://example.com/file.iso" -output "file.iso" -chunks 8

# With custom timeouts
./bin/datablip -url "https://example.com/file.bin" -output "file.bin" \
               -connect-timeout 60s -read-timeout 30m
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-url` | URL of the file to download | Required |
| `-output` | Path to save the downloaded file | Required |
| `-chunks` | Number of concurrent download chunks | 4 |
| `-connect-timeout` | Connection timeout (e.g., '30s', '1m') | 30s |
| `-read-timeout` | Read timeout per chunk (e.g., '10m', '1h') | 10m |

### Docker Usage

```bash
# Run with Docker
docker run --rm -v $(pwd):/downloads datablip:latest \
  -url "https://example.com/file.zip" \
  -output "file.zip" \
  -chunks 6

# Interactive mode
docker run --rm -it -v $(pwd):/downloads datablip:latest
```

## Building

### Prerequisites

- Go 1.21 or later
- Git (for version information)
- Make (optional, for convenience)

### Build Options

#### Using Make (Recommended)

```bash
# Standard build
make build

# Verbose build
make build-verbose

# Cross-compile for all platforms
make cross-compile

# Run tests
make test

# Clean build artifacts
make clean

# Create release packages
make release

# Show all available targets
make help
```

#### Using Build Scripts

```bash
# Basic build
bash scripts/build.sh

# Verbose build
bash scripts/build.sh --verbose

# Custom output directory
bash scripts/build.sh --output ./dist

# Cross-compile
bash scripts/cross-compile.sh
```

#### Manual Build

```bash
# Create output directory
mkdir -p bin

# Build
go build -o bin/datablip ./cmd/datablip

# Set permissions
chmod +x bin/datablip
```

### Build System Features

- **Automatic version embedding** - Git version, commit, and build time
- **Cross-platform compilation** - Supports multiple OS/architecture combinations
- **Integrated permission handling** - Post-build script ensures proper file permissions
- **Release packaging** - Creates distribution-ready packages
- **Docker integration** - Containerized builds and deployment

### Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
- FreeBSD (amd64)

## Project Structure

```
datablip/
├── cmd/datablip/        # Main application
├── bin/                 # Build output
├── scripts/             # Build and utility scripts
│   ├── build.sh         # Main build script
│   ├── post-build.sh    # Permissions script
│   ├── cross-compile.sh # Cross-compilation
│   └── release.sh       # Release packaging
├── build/
│   └── docker/          # Docker configuration
├── docs/                # Documentation
├── Makefile            # Build automation
└── README.md
```

## Configuration

### Environment Variables

```bash
# Build configuration
export DATABLIP_VERSION="1.0.0"
export DATABLIP_BUILD_DIR="./dist"

# Runtime configuration
export DATABLIP_DEFAULT_CHUNKS=8
export DATABLIP_DEFAULT_TIMEOUT="60s"
```

### Build Flags

The build system supports various flags for customization:

```bash
# Custom ldflags
make build LDFLAGS="-s -w"

# Build tags
bash scripts/build.sh --tags "netgo,osusergo"

# Custom version
VERSION=v1.0.0 make build
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific tests
go test -v ./cmd/datablip -run TestSpecificFunction
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Tidy dependencies
make tidy
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make fmt test build`
6. Submit a pull request

## Performance

DataBlip is optimized for high-performance downloads:

- **Concurrent chunks** - Splits downloads into parallel streams
- **Efficient buffering** - Optimized read/write buffer sizes
- **Connection reuse** - HTTP keep-alive for better performance
- **Memory management** - Minimal memory footprint
- **Progress optimization** - Efficient real-time progress updates

### Benchmarks

Typical performance improvements over single-threaded downloads:

- **Large files (>100MB)** - 2-4x faster
- **High-latency connections** - 3-6x faster
- **High-bandwidth connections** - 4-8x faster

## Troubleshooting

### Common Issues

1. **Permission Errors**
   ```bash
   # Fix with post-build script
   bash scripts/post-build.sh --binary datablip
   ```

2. **Build Failures**
   ```bash
   # Clean and rebuild
   make clean build
   ```

3. **Cross-Compilation Issues**
   ```bash
   # Install required tools
   go install golang.org/x/tools/cmd/goimports@latest
   ```

### Debug Mode

```bash
# Enable verbose output
./bin/datablip -url "..." -output "..." --verbose

# Check version info
./bin/datablip --version
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Community

We welcome contributions and feedback! Here’s how you can engage with the DataBlip community:

* **Report a Bug**: If you find a bug, please [open an issue](https://github.com/govind1331/Datablip/issues) and provide as much detail as possible.
* **Request a Feature**: Have an idea for a new feature? We'd love to hear it. Please [open an issue](https://github.com/govind1331/Datablip/issues) to start the conversation.
* **Ask a Question**: For general questions and discussions, please use our [GitHub Discussions](https://github.com/govind1331/Datablip/discussions).
* **Contribute Code**: We are open to contributions! Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

## Acknowledgments

- Built with Go's excellent HTTP and concurrency libraries
- Inspired by modern download managers and CLI tools