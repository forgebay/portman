#!/usr/bin/env bash
# Install portman for the current user on Linux: copies the binary to
# ~/.local/bin, the icon to the hicolor theme, and a .desktop launcher.
#
# Usage: run from the extracted release directory (must contain ./portman).
#
# Runtime dependencies (install via your package manager if the tray icon does
# not appear): libgtk-3-0 libayatana-appindicator3-1
# On GNOME, also enable the "AppIndicator and KStatusNotifierItem" extension.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$HOME/.local/bin"
APP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"

mkdir -p "$BIN_DIR" "$APP_DIR" "$ICON_DIR"

install -m 0755 "$HERE/portman" "$BIN_DIR/portman"
[ -f "$HERE/icon.png" ] && install -m 0644 "$HERE/icon.png" "$ICON_DIR/portman.png"
install -m 0644 "$HERE/portman.desktop" "$APP_DIR/portman.desktop"

# Point the launcher at the installed binary by absolute path.
sed -i "s|^Exec=portman|Exec=$BIN_DIR/portman|" "$APP_DIR/portman.desktop" || true

echo "Installed portman to $BIN_DIR/portman"
echo "Make sure $BIN_DIR is on your PATH, then launch 'Port Manager' or run: portman"
