// Package tray builds the menu-bar / system-tray UI. systray cannot reliably
// add or remove menu items at runtime, so we pre-allocate a fixed pool of items
// once in onReady and then Show/Hide/SetTitle them on each refresh. Each slot
// owns a single long-lived goroutine that maps clicks to whichever port the
// slot currently displays.
package tray

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"github.com/lanvu/portman/internal/model"
	"github.com/lanvu/portman/internal/ports"
	"github.com/lanvu/portman/internal/proc"
)

const (
	// maxSlots caps how many ports the menu can display at once.
	maxSlots = 64
	// refreshInterval is the slow background rescan cadence. Kept slow because
	// on macOS each scan shells out to lsof; manual Refresh covers immediacy.
	refreshInterval = 15 * time.Second
)

// slot is one reusable menu entry plus its kill submenu.
type slot struct {
	item      *systray.MenuItem
	kill      *systray.MenuItem
	forceKill *systray.MenuItem
	details   *systray.MenuItem // disabled; shows PID · CPU · mem
}

// Tray is the running tray controller.
type Tray struct {
	mu      sync.Mutex
	slots   []slot
	current []model.ListenPort // index-aligned with slots; what each slot shows
	stop    chan struct{}
}

// New returns a Tray ready to be passed to Run.
func New() *Tray { return &Tray{stop: make(chan struct{})} }

// Run starts the tray event loop. It blocks until the user quits and must be
// called from the main goroutine (a macOS requirement).
func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	if runtime.GOOS == "darwin" {
		systray.SetTemplateIcon(iconPNG, iconPNG)
	} else {
		systray.SetIcon(iconPNG)
	}
	systray.SetTitle("")
	systray.SetTooltip("portman — listening ports")

	refresh := systray.AddMenuItem("Refresh", "Rescan listening ports")
	systray.AddSeparator()

	t.slots = make([]slot, maxSlots)
	for i := range t.slots {
		it := systray.AddMenuItem("", "")
		it.Hide()
		s := slot{
			item:      it,
			kill:      it.AddSubMenuItem("Kill (SIGTERM)", "Gracefully terminate, then force kill"),
			forceKill: it.AddSubMenuItem("Force kill (SIGKILL)", "Kill immediately"),
			details:   it.AddSubMenuItem("…", "Process details"),
		}
		s.details.Disable()
		t.slots[i] = s

		idx := i // one permanent goroutine per slot; reads the live port at click time
		go func() {
			for {
				select {
				case <-s.kill.ClickedCh:
					t.handleKill(idx, false)
				case <-s.forceKill.ClickedCh:
					t.handleKill(idx, true)
				case <-t.stop:
					return
				}
			}
		}()
	}

	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit portman", "Exit the app")

	go func() {
		for {
			select {
			case <-refresh.ClickedCh:
				t.Refresh()
			case <-quit.ClickedCh:
				systray.Quit()
				return
			case <-t.stop:
				return
			}
		}
	}()

	t.Refresh()
	go t.loop()
}

func (t *Tray) onExit() {
	close(t.stop)
}

// loop performs a slow background refresh so the menu stays reasonably current
// without polling aggressively.
func (t *Tray) loop() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.Refresh()
		case <-t.stop:
			return
		}
	}
}

// handleKill terminates the process shown in slot idx and refreshes the menu.
func (t *Tray) handleKill(idx int, force bool) {
	t.mu.Lock()
	if idx >= len(t.current) {
		t.mu.Unlock()
		return // slot is currently hidden / stale
	}
	p := t.current[idx]
	t.mu.Unlock()

	if err := proc.Kill(p.PID, force); err != nil {
		systray.SetTooltip(fmt.Sprintf("Failed to kill %d (%s): %v", p.PID, p.ProcName, err))
	}
	t.Refresh()
}

// Refresh rescans listening ports and updates the slot pool.
func (t *Tray) Refresh() {
	list, err := ports.List()
	if err != nil {
		systray.SetTooltip("portman: " + err.Error())
		return
	}
	if len(list) > maxSlots {
		list = list[:maxSlots]
	}

	t.mu.Lock()
	t.current = list
	t.mu.Unlock()

	for i, s := range t.slots {
		if i < len(list) {
			p := list[i]
			s.item.SetTitle(fmt.Sprintf("%d · %s · %s", p.Port, p.Lang, displayName(p.ProcName)))
			s.details.SetTitle(fmt.Sprintf("PID %d · CPU %.1f%% · %s", p.PID, p.CPU, humanBytes(p.RSS)))
			s.item.Show()
		} else {
			s.item.Hide()
		}
	}

	if len(list) == 0 {
		systray.SetTooltip("portman — no listening ports")
	} else {
		systray.SetTooltip(fmt.Sprintf("portman — %d listening ports", len(list)))
	}
}

func displayName(name string) string {
	if name == "" {
		return "unknown"
	}
	return name
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
