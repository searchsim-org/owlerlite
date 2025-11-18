#!/bin/bash

# Build script for OwlerLite browser extension

set -e

echo "Building OwlerLite extension..."

# Create dist directory
mkdir -p dist/icons

# Copy source files to dist
echo "Copying files..."
cp src/manifest.json dist/
cp src/popup.html dist/
cp src/popup.js dist/
cp src/popup.css dist/
cp src/sidebar.html dist/
cp src/sidebar.js dist/
cp src/sidebar.css dist/
cp src/background.js dist/

# Copy icons if they exist
if [ -d "src/icons" ]; then
  cp src/icons/*.png dist/icons/ 2>/dev/null || echo "Warning: No icon files found. Please generate icons using generate-icons.html"
fi

# Create Firefox-compatible manifest if needed
# (Manifest V3 is compatible with both Chrome and Firefox now)

echo "Build complete!"
echo ""
echo "To load the extension:"
echo "  Chrome/Edge: chrome://extensions/ → Load unpacked → select 'dist' folder"
echo "  Firefox: about:debugging#/runtime/this-firefox → Load Temporary Add-on → select 'dist/manifest.json'"
echo ""
echo "Note: If icons are missing, open generate-icons.html in a browser and download them to src/icons/"
echo ""
echo "New features:"
echo "  - Sidebar interface for querying (Ctrl+Shift+O)"
echo "  - Configuration popup for API keys and settings"
echo "  - Clean, minimalist design"

