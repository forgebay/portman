// Presentation helpers for the tray menu. Everything here is a pure function of
// its arguments — no systray, no clock, no process lookups — so the strings the
// user actually reads can be tested without a display server.
package tray

import (
	"fmt"
	"time"

	"github.com/forgebay/portman/internal/model"
)

// rowTitle renders a top-level menu row, e.g.
// "🟢 3000 · ⬢ Next.js · acme-storefront".
func rowTitle(p model.ListenPort) string {
	return fmt.Sprintf("%s %d · %s %s · %s",
		healthDot(p.Alive), p.Port, langGlyph(p.Lang), frameworkOrLang(p), projectOrName(p))
}

// rowDetails renders the disabled detail line inside a row's submenu, e.g.
// "Node.js · PID 55136 · up 4m · CPU 0.0% · 45.9 MB".
func rowDetails(p model.ListenPort, now time.Time) string {
	return fmt.Sprintf("%s · PID %d · up %s · CPU %.1f%% · %s",
		p.Lang, p.PID, humanDurationSince(p.CreatedMs, now), p.CPU, humanBytes(p.RSS))
}

// trayTitle is the text beside the menu-bar icon: the live dev-server count, or
// nothing at all when there is none (an empty badge reads as "off").
func trayTitle(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf(" %d", n)
}

func trayTooltip(n int) string {
	if n == 0 {
		return "portman — no dev servers"
	}
	return fmt.Sprintf("portman — %d dev servers", n)
}

// capPorts trims the list to the number of pre-allocated menu slots. systray
// cannot grow its menu at runtime, so anything beyond the pool is dropped.
func capPorts(list []model.ListenPort, max int) []model.ListenPort {
	if len(list) > max {
		return list[:max]
	}
	return list
}

func displayName(name string) string {
	if name == "" {
		return "unknown"
	}
	return name
}

// frameworkOrLang prefers the detected framework (e.g. "Next.js") and falls
// back to the runtime label.
func frameworkOrLang(p model.ListenPort) string {
	if p.Framework != "" {
		return p.Framework
	}
	return string(p.Lang)
}

// projectOrName prefers the detected project name and falls back to the process
// name.
func projectOrName(p model.ListenPort) string {
	if p.Project != "" {
		return p.Project
	}
	return displayName(p.ProcName)
}

// langGlyph returns a small emoji for a runtime so rows scan quickly.
func langGlyph(l model.Lang) string {
	switch l {
	case model.Node:
		return "⬢"
	case model.Bun:
		return "🥟"
	case model.Deno:
		return "🦕"
	case model.Python:
		return "🐍"
	case model.Go:
		return "🐹"
	case model.Rust:
		return "🦀"
	case model.Ruby:
		return "💎"
	case model.PHP:
		return "🐘"
	case model.Java:
		return "☕"
	case model.Elixir:
		return "💧"
	case model.DotNet:
		return "🟪"
	case model.Ollama:
		return "🦙"
	default:
		return "•"
	}
}

func healthDot(alive bool) string {
	if alive {
		return "🟢"
	}
	return "⚪"
}

// humanDurationSince formats elapsed time into a compact "12m" / "3h" / "2d".
// now is a parameter so the output is deterministic under test.
func humanDurationSince(createdMs int64, now time.Time) string {
	if createdMs <= 0 {
		return "?"
	}
	d := now.Sub(time.UnixMilli(createdMs))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
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
