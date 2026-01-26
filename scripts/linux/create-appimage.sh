#!/bin/bash
# Create Linux AppImage for SlipStream
# Usage: ./create-appimage.sh <version> <arch>
# Example: ./create-appimage.sh 1.0.0 amd64

set -e

VERSION="${1:-dev}"
ARCH="${2:-amd64}"
APP_NAME="SlipStream"

# Map Go arch to AppImage arch
case "$ARCH" in
    amd64) APPIMAGE_ARCH="x86_64" ;;
    arm64) APPIMAGE_ARCH="aarch64" ;;
    *) APPIMAGE_ARCH="$ARCH" ;;
esac

APPIMAGE_NAME="slipstream_${VERSION}_linux_${ARCH}.AppImage"
BINARY_PATH="dist/slipstream_linux_${ARCH}_v1/slipstream"

# Paths
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/dist/appimage-build"
APP_DIR="$BUILD_DIR/SlipStream.AppDir"

echo "Creating AppImage for SlipStream v${VERSION} (${ARCH})..."

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$APP_DIR/usr/bin"
mkdir -p "$APP_DIR/usr/share/applications"
mkdir -p "$APP_DIR/usr/share/icons/hicolor/256x256/apps"

# Copy binary
cp "$PROJECT_ROOT/$BINARY_PATH" "$APP_DIR/usr/bin/slipstream"
chmod +x "$APP_DIR/usr/bin/slipstream"

# Create desktop file
cat > "$APP_DIR/usr/share/applications/slipstream.desktop" << EOF
[Desktop Entry]
Name=SlipStream
Comment=Unified media management system
Exec=slipstream
Icon=slipstream
Type=Application
Categories=AudioVideo;Network;
Terminal=false
EOF

# Copy desktop file to AppDir root (required by AppImage)
cp "$APP_DIR/usr/share/applications/slipstream.desktop" "$APP_DIR/"

# Copy icon if exists, otherwise create placeholder
if [ -f "$PROJECT_ROOT/scripts/linux/slipstream.png" ]; then
    cp "$PROJECT_ROOT/scripts/linux/slipstream.png" "$APP_DIR/usr/share/icons/hicolor/256x256/apps/slipstream.png"
    cp "$PROJECT_ROOT/scripts/linux/slipstream.png" "$APP_DIR/slipstream.png"
else
    # Create a simple placeholder icon (1x1 transparent PNG)
    echo "Warning: No icon found, AppImage will have no icon"
    touch "$APP_DIR/slipstream.png"
fi

# Create AppRun script
cat > "$APP_DIR/AppRun" << 'EOF'
#!/bin/bash
SELF=$(readlink -f "$0")
HERE=${SELF%/*}
export PATH="${HERE}/usr/bin:${PATH}"
exec "${HERE}/usr/bin/slipstream" "$@"
EOF
chmod +x "$APP_DIR/AppRun"

# Download appimagetool if not present
APPIMAGETOOL="$BUILD_DIR/appimagetool"
if [ ! -f "$APPIMAGETOOL" ]; then
    echo "Downloading appimagetool..."
    curl -L -o "$APPIMAGETOOL" "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "$APPIMAGETOOL"
fi

# Create AppImage
echo "Creating AppImage..."
ARCH="$APPIMAGE_ARCH" "$APPIMAGETOOL" "$APP_DIR" "$PROJECT_ROOT/dist/$APPIMAGE_NAME"

# Clean up
rm -rf "$BUILD_DIR"

echo "Created: $PROJECT_ROOT/dist/$APPIMAGE_NAME"
