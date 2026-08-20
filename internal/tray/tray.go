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

	"github.com/forgebay/portman/internal/autostart"
	"github.com/forgebay/portman/internal/config"
	"github.com/forgebay/portman/internal/model"
	"github.com/forgebay/portman/internal/ports"
	"github.com/forgebay/portman/internal/proc"
)

// maxSlots caps how many ports the menu can display at once.
const maxSlots = 64

// slot is one reusable menu entry plus its action submenu.
type slot struct {
	item      *systray.MenuItem
	url       *systray.MenuItem // disabled; shows http://localhost:<port>
	open      *systray.MenuItem
	copyURL   *systray.MenuItem
	editor    *systray.MenuItem
	reveal    *systray.MenuItem
	kill      *systray.MenuItem
	forceKill *systray.MenuItem
	details   *systray.MenuItem // disabled; shows runtime · PID · uptime · CPU · mem
}

// Tray is the running tray controller.
type Tray struct {
	mu        sync.Mutex
	slots     []slot
	current   []model.ListenPort // index-aligned with slots; what each slot shows
	stop      chan struct{}
	reconfig  chan time.Duration // signals the loop to reset its ticker
	cfg       config.Config
	hasEditor bool

	showAllItem   *systray.MenuItem
	intervalItems map[int]*systray.MenuItem // refresh seconds -> checkbox item
}

// New returns a Tray ready to be passed to Run.
func New() *Tray {
	return &Tray{
		stop:     make(chan struct{}),
		reconfig: make(chan time.Duration, 1),
		cfg:      config.Load(),
	}
}

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
	systray.SetTooltip("portman — dev servers")

	refresh := systray.AddMenuItem("Refresh", "Rescan dev servers")
	systray.AddSeparator()

	revealLabel := "Reveal in Finder"
	if runtime.GOOS != "darwin" {
		revealLabel = "Open project folder"
	}
	editorBin, editorLabel := editor()
	t.hasEditor = editorBin != ""
	if editorLabel == "" {
		editorLabel = "Open in editor"
	}

	t.slots = make([]slot, maxSlots)
	for i := range t.slots {
		it := systray.AddMenuItem("", "")
		it.Hide()
		s := slot{
			item:      it,
			url:       it.AddSubMenuItem("", ""),
			open:      it.AddSubMenuItem("Open in browser", "Open http://localhost:<port>"),
			copyURL:   it.AddSubMenuItem("Copy URL", "Copy http://localhost:<port> to the clipboard"),
			editor:    it.AddSubMenuItem(editorLabel, "Open the project in your editor"),
			reveal:    it.AddSubMenuItem(revealLabel, "Open the project directory"),
			kill:      it.AddSubMenuItem("Kill (SIGTERM)", "Gracefully terminate, then force kill"),
			forceKill: it.AddSubMenuItem("Force kill (SIGKILL)", "Kill immediately"),
			details:   it.AddSubMenuItem("…", "Process details"),
		}
		s.url.Disable()
		s.details.Disable()
		if !t.hasEditor {
			s.editor.Hide()
		}
		t.slots[i] = s

		idx := i // one permanent goroutine per slot; reads the live port at click time
		go func() {
			for {
				select {
				case <-s.open.ClickedCh:
					t.handleOpen(idx)
				case <-s.copyURL.ClickedCh:
					t.handleCopy(idx)
				case <-s.editor.ClickedCh:
					t.handleEditor(idx)
				case <-s.reveal.ClickedCh:
					t.handleReveal(idx)
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

	// Settings submenu: refresh cadence + show-all toggle.
	settings := systray.AddMenuItem("Settings", "Preferences")
	i5 := settings.AddSubMenuItemCheckbox("Refresh every 5s", "", t.cfg.RefreshSeconds == 5)
	i15 := settings.AddSubMenuItemCheckbox("Refresh every 15s", "", t.cfg.RefreshSeconds == 15)
	i30 := settings.AddSubMenuItemCheckbox("Refresh every 30s", "", t.cfg.RefreshSeconds == 30)
	showAll := settings.AddSubMenuItemCheckbox("Show all ports", "Include non-dev / system processes", t.cfg.ShowAll)
	t.intervalItems = map[int]*systray.MenuItem{5: i5, 15: i15, 30: i30}
	t.showAllItem = showAll

	startup := systray.AddMenuItemCheckbox("Start at login", "Launch portman automatically when you log in", autostart.IsEnabled())
	quit := systray.AddMenuItem("Quit portman", "Exit the app")

	go func() {
		for {
			select {
			case <-refresh.ClickedCh:
				t.Refresh()
			case <-i5.ClickedCh:
				t.setInterval(5)
			case <-i15.ClickedCh:
				t.setInterval(15)
			case <-i30.ClickedCh:
				t.setInterval(30)
			case <-showAll.ClickedCh:
				t.toggleShowAll()
			case <-startup.ClickedCh:
				t.toggleStartup(startup)
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

// loop performs a background refresh on a ticker whose interval can change at
// runtime (via the Settings submenu) without restarting the loop.
func (t *Tray) loop() {
	t.mu.Lock()
	d := time.Duration(t.cfg.RefreshSeconds) * time.Second
	t.mu.Unlock()
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.Refresh()
		case nd := <-t.reconfig:
			ticker.Reset(nd)
		case <-t.stop:
			return
		}
	}
}

// setInterval changes the refresh cadence, persists it, and resets the ticker.
func (t *Tray) setInterval(sec int) {
	t.mu.Lock()
	t.cfg.RefreshSeconds = sec
	cfg := t.cfg
	t.mu.Unlock()

	for s, it := range t.intervalItems {
		if s == sec {
			it.Check()
		} else {
			it.Uncheck()
		}
	}
	_ = config.Save(cfg)
	select {
	case t.reconfig <- time.Duration(sec) * time.Second:
	default:
	}
	t.Refresh()
}

// toggleShowAll flips the dev-only filter and persists the choice.
func (t *Tray) toggleShowAll() {
	t.mu.Lock()
	t.cfg.ShowAll = !t.cfg.ShowAll
	cfg := t.cfg
	t.mu.Unlock()

	if cfg.ShowAll {
		t.showAllItem.Check()
	} else {
		t.showAllItem.Uncheck()
	}
	_ = config.Save(cfg)
	t.Refresh()
}

// toggleStartup flips launch-at-login and keeps the checkbox in sync with the
// actual on-disk state (so a failure doesn't leave the UI lying).
func (t *Tray) toggleStartup(item *systray.MenuItem) {
	var err error
	if item.Checked() {
		err = autostart.Disable()
	} else {
		err = autostart.Enable()
	}
	if err != nil {
		systray.SetTooltip("portman: start-at-login failed: " + err.Error())
	}
	if autostart.IsEnabled() {
		item.Check()
	} else {
		item.Uncheck()
	}
}

// portAt returns the port currently shown in slot idx, or false if the slot is
// hidden/stale.
func (t *Tray) portAt(idx int) (model.ListenPort, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if idx >= len(t.current) {
		return model.ListenPort{}, false
	}
	return t.current[idx], true
}

func (t *Tray) handleOpen(idx int) {
	if p, ok := t.portAt(idx); ok {
		openURL(fmt.Sprintf("http://localhost:%d", p.Port))
	}
}

func (t *Tray) handleCopy(idx int) {
	if p, ok := t.portAt(idx); ok {
		url := fmt.Sprintf("http://localhost:%d", p.Port)
		if err := copyText(url); err == nil {
			systray.SetTooltip("Copied " + url)
		}
	}
}

func (t *Tray) handleEditor(idx int) {
	if p, ok := t.portAt(idx); ok && p.Cwd != "" {
		openInEditor(p.Cwd)
	}
}

func (t *Tray) handleReveal(idx int) {
	if p, ok := t.portAt(idx); ok && p.Cwd != "" {
		openURL(p.Cwd)
	}
}

// handleKill terminates the process shown in slot idx and refreshes the menu.
func (t *Tray) handleKill(idx int, force bool) {
	p, ok := t.portAt(idx)
	if !ok {
		return
	}
	if err := proc.Kill(p.PID, force); err != nil {
		systray.SetTooltip(fmt.Sprintf("Failed to kill %d (%s): %v", p.PID, p.ProcName, err))
	}
	t.Refresh()
}

// Refresh rescans dev servers and updates the slot pool.
func (t *Tray) Refresh() {
	t.mu.Lock()
	showAll := t.cfg.ShowAll
	t.mu.Unlock()

	list, err := ports.List(showAll)
	if err != nil {
		systray.SetTooltip("portman: " + err.Error())
		return
	}
	list = capPorts(list, maxSlots)

	t.mu.Lock()
	t.current = list
	t.mu.Unlock()

	for i, s := range t.slots {
		if i < len(list) {
			p := list[i]
			s.item.SetTitle(rowTitle(p))
			s.url.SetTitle(fmt.Sprintf("http://localhost:%d", p.Port))
			s.details.SetTitle(rowDetails(p, time.Now()))
			setEnabled(s.reveal, p.Cwd != "")
			setEnabled(s.editor, t.hasEditor && p.Cwd != "")
			s.item.Show()
		} else {
			s.item.Hide()
		}
	}

	systray.SetTitle(trayTitle(len(list)))
	systray.SetTooltip(trayTooltip(len(list)))
}

func setEnabled(item *systray.MenuItem, enabled bool) {
	if enabled {
		item.Enable()
	} else {
		item.Disable()
	}
}
