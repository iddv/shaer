# Build and Packaging Guide

This document provides comprehensive instructions for building and packaging the File Sharing App for all supported platforms.

## Prerequisites

### Development Environment
- Go 1.23.4 or later
- Git
- Make (optional, for using Makefile)

### Platform-Specific Requirements

#### Windows
- GCC compiler (via MinGW-w64 for cross-compilation from Linux)
- NSIS (Nullsoft Scriptable Install System) for creating installers

#### macOS
- Xcode Command Line Tools
- `iconutil` (included with macOS) for icon conversion
- `hdiutil` (included with macOS) for DMG creation

#### Linux
- GCC compiler
- GTK 3.0 development libraries
- ImageMagick (for icon conversion)
- `dpkg-deb` (for .deb packages)
- `appimagetool` (downloaded automatically for AppImage creation)

## Quick Start

### Build for Current Platform
```bash
make build-local
```

### Build for All Platforms
```bash
make build
```

### Create Packages for All Platforms
```bash
make package
```

## Detailed Build Instructions

### Using Make (Recommended)

The project includes a comprehensive Makefile with the following targets:

```bash
# Show all available targets
make help

# Clean build artifacts
make clean

# Build for all platforms
make build

# Build for specific platforms
make build-windows
make build-darwin
make build-linux

# Build for current platform only
make build-local

# Generate checksums
make checksums

# Create packages for distribution
make package

# Test the local build
make test-build

# Install locally (Linux/macOS)
make install
```

### Using Build Scripts

#### Cross-Platform Build Script
```bash
./build/build.sh
```

#### Windows-Specific Build
```bash
# On Windows
build\build-windows.bat

# On Linux (cross-compilation)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o dist/file-sharing-app-windows-amd64.exe ./cmd/main.go
```

#### Complete Packaging
```bash
./build/package-all.sh
```

### Manual Build Commands

#### Basic Build
```bash
go build -o dist/file-sharing-app ./cmd/main.go
```

#### Build with Version Information
```bash
VERSION="1.0.0"
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

go build -ldflags="${LDFLAGS}" -o dist/file-sharing-app ./cmd/main.go
```

#### Cross-Compilation Examples
```bash
# Windows (from Linux)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags="${LDFLAGS}" -o dist/file-sharing-app-windows-amd64.exe ./cmd/main.go

# macOS (from Linux)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="${LDFLAGS}" -o dist/file-sharing-app-darwin-amd64 ./cmd/main.go

# Linux ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="${LDFLAGS}" -o dist/file-sharing-app-linux-arm64 ./cmd/main.go
```

## Packaging

### Windows Packaging

#### ZIP Package (Basic)
```bash
mkdir -p dist/windows-package
cp dist/file-sharing-app-windows-amd64.exe dist/windows-package/
cp README.md LICENSE dist/windows-package/
cd dist && zip -r file-sharing-app-windows.zip windows-package/
```

#### NSIS Installer (Advanced)
```bash
# Requires NSIS to be installed
makensis build/package-windows.nsi
```

### macOS Packaging

#### App Bundle
```bash
./build/package-macos.sh
```

This creates:
- `File Sharing App.app` - macOS application bundle
- `File Sharing App-1.0.0.dmg` - Disk image for distribution

#### Manual App Bundle Creation
```bash
APP_BUNDLE="dist/File Sharing App.app"
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${APP_BUNDLE}/Contents/Resources"

# Copy binary (create universal binary if both architectures exist)
lipo -create \
    dist/file-sharing-app-darwin-amd64 \
    dist/file-sharing-app-darwin-arm64 \
    -output "${APP_BUNDLE}/Contents/MacOS/file-sharing-app"

# Create Info.plist (see build/package-macos.sh for full example)
```

### Linux Packaging

#### Tar.gz Package (Universal)
```bash
./build/package-linux.sh
```

This creates:
- `file-sharing-app-1.0.0-linux.tar.gz` - Portable package with install scripts
- `file-sharing-app_1.0.0_amd64.deb` - Debian/Ubuntu package
- `File Sharing App-1.0.0-x86_64.AppImage` - Universal Linux AppImage

#### Manual .deb Package Creation
```bash
# Create package structure
DEB_DIR="dist/file-sharing-app_1.0.0_amd64"
mkdir -p "${DEB_DIR}/DEBIAN"
mkdir -p "${DEB_DIR}/usr/bin"
mkdir -p "${DEB_DIR}/usr/share/applications"

# Copy files
cp dist/file-sharing-app-linux-amd64 "${DEB_DIR}/usr/bin/file-sharing-app"
# Create control file and desktop file (see build/package-linux.sh)

# Build package
dpkg-deb --build "${DEB_DIR}" "dist/file-sharing-app_1.0.0_amd64.deb"
```

## Testing Builds

### Automated Testing
```bash
make test-build
```

### Manual Testing
```bash
# Test version information
./dist/file-sharing-app -version

# Test help
./dist/file-sharing-app -help

# Test basic functionality (requires display)
./dist/file-sharing-app
```

### Cross-Platform Testing

#### Windows (using Wine on Linux)
```bash
wine dist/file-sharing-app-windows-amd64.exe -version
```

#### macOS (on macOS)
```bash
./dist/file-sharing-app-darwin-amd64 -version
```

## Continuous Integration

The project includes GitHub Actions workflows for automated building and packaging:

- `.github/workflows/build.yml` - Builds for all platforms and creates releases

### Triggering Releases

1. Create and push a version tag:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create packages
   - Generate checksums
   - Create a GitHub release with all artifacts

## Troubleshooting

### Common Build Issues

#### CGO Compilation Errors
- Ensure proper C compiler is installed for target platform
- For Windows cross-compilation: `sudo apt-get install gcc-mingw-w64-x86-64`
- For Linux: `sudo apt-get install build-essential`

#### Missing Dependencies
```bash
# Linux
sudo apt-get install libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config

# macOS
xcode-select --install
```

#### Fyne-Specific Issues
- Ensure CGO is enabled: `export CGO_ENABLED=1`
- Install platform-specific GUI libraries
- For headless builds, some GUI operations may fail

### Platform-Specific Issues

#### Windows
- Use proper file extensions (.exe)
- Ensure MinGW-w64 is properly configured for cross-compilation
- NSIS installer requires Windows or Wine

#### macOS
- Code signing may be required for distribution
- Notarization needed for macOS 10.15+
- Universal binaries require both amd64 and arm64 builds

#### Linux
- Different distributions may have different library requirements
- AppImage provides best compatibility across distributions
- .deb packages work on Debian-based systems only

## Distribution

### Release Checklist

1. **Pre-release**
   - [ ] Update version in relevant files
   - [ ] Update CHANGELOG.md
   - [ ] Test builds on all platforms
   - [ ] Verify all features work correctly

2. **Build and Package**
   - [ ] Run full build and packaging process
   - [ ] Verify all packages are created
   - [ ] Test installation on target platforms
   - [ ] Generate and verify checksums

3. **Release**
   - [ ] Create GitHub release with all packages
   - [ ] Update documentation
   - [ ] Announce release

### Package Verification

Always verify packages before distribution:

```bash
# Verify checksums
sha256sum -c dist/packages-checksums.txt

# Test installation
# Windows: Run installer or extract ZIP
# macOS: Mount DMG and test app
# Linux: Install .deb or run AppImage
```

## Advanced Topics

### Custom Build Configurations

#### Debug Builds
```bash
go build -gcflags="all=-N -l" -o dist/file-sharing-app-debug ./cmd/main.go
```

#### Static Linking
```bash
CGO_ENABLED=0 go build -a -ldflags='-extldflags "-static"' -o dist/file-sharing-app-static ./cmd/main.go
```

#### Optimized Builds
```bash
go build -ldflags="-s -w" -o dist/file-sharing-app-optimized ./cmd/main.go
```

### Custom Packaging

#### Docker-based Building
```dockerfile
FROM golang:1.23.4-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY . .
RUN go build -o file-sharing-app ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/file-sharing-app .
CMD ["./file-sharing-app"]
```

#### Custom Installers
- Windows: Modify `build/package-windows.nsi`
- macOS: Customize `build/package-macos.sh`
- Linux: Modify `build/package-linux.sh`

This comprehensive build system ensures consistent, reliable builds across all supported platforms while providing flexibility for different deployment scenarios.