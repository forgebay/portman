// Command portman is a lightweight menu-bar / system-tray app that lists
// listening TCP ports, shows the runtime of each owning process, and lets you
// kill a process with one click.
package main

import "github.com/lanvu/portman/internal/tray"

func main() {
	// systray.Run must own the main goroutine on macOS, so call it directly.
	tray.New().Run()
}
