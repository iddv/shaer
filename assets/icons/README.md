# Application Icons

This directory contains application icons for different platforms.

## Icon Requirements

### Windows
- `icon.ico` - Windows icon file (16x16, 32x32, 48x48, 256x256)

### macOS
- `icon.icns` - macOS icon file (multiple resolutions)
- Individual PNG files for different sizes

### Linux
- `icon.png` - Standard PNG icon (typically 256x256)
- `icon.svg` - Scalable vector icon (preferred)

## Icon Generation

To generate icons from a source image:

1. Create a high-resolution source image (1024x1024 PNG)
2. Use online tools or command-line utilities to generate platform-specific formats:
   - For Windows: Use `convert` (ImageMagick) to create .ico files
   - For macOS: Use `iconutil` to create .icns files
   - For Linux: Use standard PNG files

## Current Status

Currently using placeholder icons. Replace with actual application icons before distribution.