//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const label = "vn.forgebay.portman"

// legacyLabels are LaunchAgent labels shipped by earlier releases. They are
// cleaned up on startup so a label change does not leave the user with two
// login items, both launching portman.
var legacyLabels = []string{"vn.redsun.portman"}

func agentPath(agentLabel string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist"), nil
}

func plistPath() (string, error) {
	return agentPath(label)
}

// Migrate unloads and deletes LaunchAgents written under a legacy label. If one
// was present the user had launch-at-login switched on, so it is re-registered
// under the current label to preserve that setting across the rename.
func Migrate() error {
	var hadLegacy bool
	for _, legacy := range legacyLabels {
		p, err := agentPath(legacy)
		if err != nil {
			return err
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		hadLegacy = true
		// Best-effort unload; the agent may not be loaded in this session.
		_ = exec.Command("launchctl", "unload", p).Run()
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if !hadLegacy || IsEnabled() {
		return nil
	}
	return Enable()
}

// IsEnabled reports whether the LaunchAgent plist exists.
func IsEnabled() bool {
	p, err := plistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Enable writes a LaunchAgent that runs the current binary at login and loads
// it immediately.
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)

	p, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(fmt.Sprintf(plistTemplate, label, exe)), 0o644); err != nil {
		return err
	}
	// Best-effort load; ignore errors (e.g. already loaded).
	_ = exec.Command("launchctl", "unload", p).Run()
	_ = exec.Command("launchctl", "load", p).Run()
	return nil
}

// Disable unloads and removes the LaunchAgent plist.
func Disable() error {
	p, err := plistPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", p).Run()
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`
