//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

func desktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "portman.desktop"), nil
}

// IsEnabled reports whether the XDG autostart entry exists.
func IsEnabled() bool {
	p, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Enable writes an XDG autostart .desktop entry pointing at the current binary.
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)

	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(fmt.Sprintf(desktopTemplate, exe)), 0o644)
}

// Disable removes the XDG autostart entry.
func Disable() error {
	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=Port Manager
Comment=List listening ports and kill processes from the system tray
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`
