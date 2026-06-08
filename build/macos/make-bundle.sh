#!/usr/bin/env bash
# Wrap a built portman binary into a macOS menu-bar .app bundle.
#
# Usage: make-bundle.sh <binary-path> <version> [arch]
#   binary-path : path to the compiled portman binary
#   version     : version string written into Info.plist (e.g. 1.0.0)
#   arch        : asset arch label for the zip name (default: uname -m).
#                 Pass explicitly when cross-building (e.g. amd64 on an
#                 Apple Silicon runner), since uname -m would be wrong.
#
# Produces dist/Port Manager.app and dist/PortManager-macos-<arch>.zip
set -euo pipefail

BIN="${1:?usage: make-bundle.sh <binary-path> <version> [arch]}"
VERSION="${2:-0.0.0}"
ARCH="${3:-$(uname -m)}"
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"

APP="$ROOT/dist/Port Manager.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp "$BIN" "$APP/Contents/MacOS/portman"
chmod +x "$APP/Contents/MacOS/portman"

sed "s/VERSION_PLACEHOLDER/$VERSION/g" "$HERE/Info.plist" > "$APP/Contents/Info.plist"

# Build an .icns from the PNG icon if iconutil is available; otherwise skip
# (the app still runs, just without a Finder icon).
ICON_PNG="$ROOT/assets/icon.png"
if command -v iconutil >/dev/null && command -v sips >/dev/null && [ -f "$ICON_PNG" ]; then
	ICONSET="$(mktemp -d)/icon.iconset"
	mkdir -p "$ICONSET"
	for size in 16 32 64 128 256 512; do
		sips -z "$size" "$size" "$ICON_PNG" --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
		sips -z "$((size*2))" "$((size*2))" "$ICON_PNG" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
	done
	iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/icon.icns"
fi

mkdir -p "$ROOT/dist"
( cd "$ROOT/dist" && zip -qr "PortManager-macos-${ARCH}.zip" "Port Manager.app" )
echo "built $APP and dist/PortManager-macos-${ARCH}.zip"
