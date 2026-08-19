// Command portman is a lightweight menu-bar / system-tray app that lists
// listening TCP ports, shows the runtime of each owning process, and lets you
// kill a process with one click.
package main

import (
	"github.com/forgebay/portman/internal/autostart"
	"github.com/forgebay/portman/internal/tray"
)

func main() {
	// Drop LaunchAgents left behind by pre-rename releases before the tray
	// reads the launch-at-login state.
	_ = autostart.Migrate()

	// systray.Run must own the main goroutine on macOS, so call it directly.
	tray.New().Run()
}
