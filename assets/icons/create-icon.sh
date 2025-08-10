#!/bin/bash

# Script to create application icons from the Shaer logo

if command -v convert &> /dev/null; then
    echo "Creating application icons from shaer_logo.png..."
    
    # Use the actual Shaer logo as the base
    BASE_LOGO="assets/icons/shaer_logo.png"
    
    if [[ ! -f "$BASE_LOGO" ]]; then
        echo "Error: shaer_logo.png not found in assets/icons/"
        exit 1
    fi
    
    # Create different sizes for various platforms
    echo "Creating PNG icons..."
    convert "$BASE_LOGO" -resize 16x16 assets/icons/icon-16.png
    convert "$BASE_LOGO" -resize 32x32 assets/icons/icon-32.png
    convert "$BASE_LOGO" -resize 48x48 assets/icons/icon-48.png
    convert "$BASE_LOGO" -resize 128x128 assets/icons/icon-128.png
    convert "$BASE_LOGO" -resize 256x256 assets/icons/icon-256.png
    convert "$BASE_LOGO" -resize 512x512 assets/icons/icon-512.png
    
    # Create main icon.png (256x256 is standard)
    cp assets/icons/icon-256.png assets/icons/icon.png
    
    # Create Windows ICO file
    echo "Creating Windows ICO file..."
    convert assets/icons/icon-16.png assets/icons/icon-32.png assets/icons/icon-48.png assets/icons/icon-256.png assets/icons/icon.ico
    
    # Create macOS ICNS file if iconutil is available (macOS only)
    if command -v iconutil &> /dev/null; then
        echo "Creating macOS ICNS file..."
        mkdir -p assets/icons/icon.iconset
        cp assets/icons/icon-16.png assets/icons/icon.iconset/icon_16x16.png
        cp assets/icons/icon-32.png assets/icons/icon.iconset/icon_16x16@2x.png
        cp assets/icons/icon-32.png assets/icons/icon.iconset/icon_32x32.png
        cp assets/icons/icon-128.png assets/icons/icon.iconset/icon_32x32@2x.png
        cp assets/icons/icon-128.png assets/icons/icon.iconset/icon_128x128.png
        cp assets/icons/icon-256.png assets/icons/icon.iconset/icon_128x128@2x.png
        cp assets/icons/icon-256.png assets/icons/icon.iconset/icon_256x256.png
        cp assets/icons/icon-512.png assets/icons/icon.iconset/icon_256x256@2x.png
        cp assets/icons/icon-512.png assets/icons/icon.iconset/icon_512x512.png
        
        iconutil -c icns assets/icons/icon.iconset -o assets/icons/icon.icns
        rm -rf assets/icons/icon.iconset
        echo "Created icon.icns"
    else
        echo "iconutil not available (macOS only), skipping ICNS creation"
    fi
    
    echo "Icon creation complete!"
    echo "Created icons:"
    ls -la assets/icons/icon*
else
    echo "ImageMagick not available. Please install it to create icons:"
    echo "  Ubuntu/Debian: sudo apt-get install imagemagick"
    echo "  macOS: brew install imagemagick"
    echo "  Or use online tools to convert the PNG to different sizes and ICO format"
fi