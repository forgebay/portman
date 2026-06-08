//go:build !darwin && !linux

package autostart

// IsEnabled always reports false on unsupported platforms.
func IsEnabled() bool { return false }

// Enable is unsupported on this platform.
func Enable() error { return ErrUnsupported }

// Disable is unsupported on this platform.
func Disable() error { return ErrUnsupported }
