#!/bin/bash
# Create macOS DMG installer for SlipStream
# Usage: ./create-dmg.sh <version> <arch>
# Example: ./create-dmg.sh 1.0.0 arm64

set -e

VERSION="${1:-dev}"
ARCH="${2:-arm64}"
APP_NAME="SlipStream"
DMG_NAME="slipstream_${VERSION}_darwin_${ARCH}.dmg"
BINARY_PATH="dist/slipstream_darwin_${ARCH}/slipstream"

# Paths
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/dist/dmg-build"
APP_DIR="$BUILD_DIR/$APP_NAME.app"

echo "Creating macOS app bundle for SlipStream v${VERSION} (${ARCH})..."

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

# Copy binary
cp "$PROJECT_ROOT/$BINARY_PATH" "$APP_DIR/Contents/MacOS/slipstream"
chmod +x "$APP_DIR/Contents/MacOS/slipstream"

# Create Info.plist
cat > "$APP_DIR/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>SlipStream</string>
    <key>CFBundleDisplayName</key>
    <string>SlipStream</string>
    <key>CFBundleIdentifier</key>
    <string>com.slipstream.app</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleExecutable</key>
    <string>slipstream</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <false/>
</dict>
</plist>
EOF

# Copy icon if exists
if [ -f "$PROJECT_ROOT/scripts/macos/slipstream.icns" ]; then
    cp "$PROJECT_ROOT/scripts/macos/slipstream.icns" "$APP_DIR/Contents/Resources/slipstream.icns"
    # Add icon reference to Info.plist
    /usr/libexec/PlistBuddy -c "Add :CFBundleIconFile string slipstream.icns" "$APP_DIR/Contents/Info.plist" 2>/dev/null || true
fi

# Create DMG
echo "Creating DMG..."
DMG_PATH="$PROJECT_ROOT/dist/$DMG_NAME"

# Create temporary DMG
hdiutil create -volname "$APP_NAME" -srcfolder "$BUILD_DIR" -ov -format UDRW "$BUILD_DIR/temp.dmg"

# Convert to compressed DMG
hdiutil convert "$BUILD_DIR/temp.dmg" -format UDZO -o "$DMG_PATH"

# Clean up
rm -rf "$BUILD_DIR"

echo "Created: $DMG_PATH"
