// Package autostart registers (or removes) portman as a "launch at login"
// item. It is split per-OS via build tags: macOS uses a LaunchAgent plist,
// Linux uses an XDG autostart .desktop file. The public API is:
//
//	IsEnabled() bool   // is launch-at-login currently configured?
//	Enable() error     // configure launch-at-login for the current binary
//	Disable() error    // remove launch-at-login
package autostart

import "errors"

// ErrUnsupported is returned by Enable/Disable on platforms without an
// autostart implementation.
var ErrUnsupported = errors.New("launch at login is not supported on this platform")
