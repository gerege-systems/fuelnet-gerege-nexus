#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
BUILD_DIR="$SCRIPT_DIR/build"
APP_NAME="Gerege Nexus"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"

echo "🔨 Building $APP_NAME for macOS..."

# Clean previous build
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

# Copy Info.plist
cp "$SCRIPT_DIR/Info.plist" "$APP_BUNDLE/Contents/Info.plist"

# Get SDK Path
SDK_PATH=$(xcrun --show-sdk-path)

# Compile Swift app binary
echo "📦 Compiling Swift native desktop engine..."
swiftc -O \
    -sdk "$SDK_PATH" \
    -framework Cocoa \
    -framework WebKit \
    -framework LocalAuthentication \
    -framework UserNotifications \
    "$SCRIPT_DIR/src/BiometricAuth.swift" \
    "$SCRIPT_DIR/src/ServerManager.swift" \
    "$SCRIPT_DIR/src/PreferencesWindowController.swift" \
    "$SCRIPT_DIR/src/WebViewController.swift" \
    "$SCRIPT_DIR/src/MainWindowController.swift" \
    "$SCRIPT_DIR/src/MenuBuilder.swift" \
    "$SCRIPT_DIR/src/AppDelegate.swift" \
    "$SCRIPT_DIR/src/main.swift" \
    -o "$APP_BUNDLE/Contents/MacOS/$APP_NAME"

# Copy icon if available
if [ -f "$ROOT_DIR/frontend/public/brand.png" ]; then
    cp "$ROOT_DIR/frontend/public/brand.png" "$APP_BUNDLE/Contents/Resources/AppIcon.png"
fi

# Code sign app locally
echo "🔏 Signing application bundle..."
codesign --force --deep --sign - "$APP_BUNDLE"

echo "✅ Success! macOS Native Desktop App built at:"
echo "   $APP_BUNDLE"
