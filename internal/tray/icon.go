package tray

import _ "embed"

// iconPNG is the menu-bar icon, embedded at build time. It is a monochrome
// template image so macOS tints it for light/dark menu bars automatically.
//
//go:embed icon.png
var iconPNG []byte
