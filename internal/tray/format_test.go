package tray

import (
	"testing"
	"time"

	"github.com/forgebay/portman/internal/model"
)

func TestRowTitle(t *testing.T) {
	tests := []struct {
		name string
		in   model.ListenPort
		want string
	}{
		{
			name: "framework and project both detected",
			in:   model.ListenPort{Port: 3000, Lang: model.Node, Framework: "Next.js", Project: "acme-storefront", Alive: true},
			want: "🟢 3000 · ⬢ Next.js · acme-storefront",
		},
		{
			name: "no framework falls back to the runtime label",
			in:   model.ListenPort{Port: 8080, Lang: model.Go, Project: "inventory-api", Alive: true},
			want: "🟢 8080 · 🐹 Go · inventory-api",
		},
		{
			name: "no project falls back to the process name",
			in:   model.ListenPort{Port: 5000, Lang: model.Python, Framework: "Flask", ProcName: "python3", Alive: true},
			want: "🟢 5000 · 🐍 Flask · python3",
		},
		{
			name: "nothing to name it by — the runtime is not repeated",
			in:   model.ListenPort{Port: 9999, Lang: model.Unknown, Alive: false},
			want: "⚪ 9999 · • Unknown",
		},
		{
			// Observed: a plain node server with no manifest rendered as
			// "Node.js · node", and macOS reports the framework python3 process
			// as "Python", giving "Python · Python".
			name: "plain node server with no project",
			in:   model.ListenPort{Port: 4321, Lang: model.Node, ProcName: "node", Alive: true},
			want: "🟢 4321 · ⬢ Node.js",
		},
		{
			name: "plain python server with no project",
			in:   model.ListenPort{Port: 4322, Lang: model.Python, ProcName: "Python", Alive: true},
			want: "🟢 4322 · 🐍 Python",
		},
		{
			name: "a project name that happens to match the runtime is not doubled",
			in:   model.ListenPort{Port: 7000, Lang: model.Go, Project: "go", Alive: true},
			want: "🟢 7000 · 🐹 Go",
		},
		{
			name: "a port that stopped answering gets the hollow dot",
			in:   model.ListenPort{Port: 4000, Lang: model.Node, Framework: "NestJS", Project: "orders-gateway", Alive: false},
			want: "⚪ 4000 · ⬢ NestJS · orders-gateway",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rowTitle(tc.in); got != tc.want {
				t.Errorf("rowTitle() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRowDetails(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	p := model.ListenPort{
		Lang:      model.Node,
		PID:       55136,
		CreatedMs: now.Add(-4 * time.Minute).UnixMilli(),
		CPU:       0.04,
		RSS:       45_900_000,
	}
	want := "Node.js · PID 55136 · up 4m · CPU 0.0% · 43.8 MB"
	if got := rowDetails(p, now); got != want {
		t.Errorf("rowDetails() = %q, want %q", got, want)
	}
}

func TestTrayTitleAndTooltip(t *testing.T) {
	// An empty badge is deliberate: "0" beside the icon reads as a broken app.
	if got := trayTitle(0); got != "" {
		t.Errorf("trayTitle(0) = %q, want empty", got)
	}
	if got := trayTitle(6); got != " 6" {
		t.Errorf("trayTitle(6) = %q", got)
	}
	if got := trayTooltip(0); got != "portman — no dev servers" {
		t.Errorf("trayTooltip(0) = %q", got)
	}
	if got := trayTooltip(1); got != "portman — 1 dev servers" {
		t.Errorf("trayTooltip(1) = %q", got)
	}
}

func TestCapPorts(t *testing.T) {
	list := make([]model.ListenPort, 5)
	if got := capPorts(list, 10); len(got) != 5 {
		t.Errorf("under the cap: len = %d, want 5", len(got))
	}
	if got := capPorts(list, 5); len(got) != 5 {
		t.Errorf("at the cap: len = %d, want 5", len(got))
	}
	if got := capPorts(list, 3); len(got) != 3 {
		t.Errorf("over the cap: len = %d, want 3", len(got))
	}
	if got := capPorts(nil, 3); got != nil {
		t.Errorf("nil list should stay nil, got %v", got)
	}
}

func TestLangGlyph_CoversEveryRuntime(t *testing.T) {
	// Every runtime the detector can return needs a glyph; a missing case would
	// silently render as the "•" fallback.
	for _, l := range []model.Lang{
		model.Node, model.Bun, model.Deno, model.Python, model.Go, model.Rust,
		model.Ruby, model.PHP, model.Java, model.Elixir, model.DotNet, model.Ollama,
	} {
		if g := langGlyph(l); g == "•" {
			t.Errorf("langGlyph(%q) fell through to the default glyph", l)
		}
	}
	if g := langGlyph(model.Unknown); g != "•" {
		t.Errorf("langGlyph(Unknown) = %q, want the fallback", g)
	}
}

func TestHumanDurationSince(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"seconds", 42 * time.Second, "42s"},
		{"just under a minute", 59 * time.Second, "59s"},
		{"minutes", 4 * time.Minute, "4m"},
		{"just under an hour", 59 * time.Minute, "59m"},
		{"hours", 5 * time.Hour, "5h"},
		{"just under a day", 23 * time.Hour, "23h"},
		{"days", 50 * time.Hour, "2d"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := humanDurationSince(now.Add(-tc.ago).UnixMilli(), now)
			if got != tc.want {
				t.Errorf("humanDurationSince(-%v) = %q, want %q", tc.ago, got, tc.want)
			}
		})
	}
	// An unknown start time must not render as "0s" — that would claim the
	// process just started.
	if got := humanDurationSince(0, now); got != "?" {
		t.Errorf("humanDurationSince(0) = %q, want %q", got, "?")
	}
	if got := humanDurationSince(-1, now); got != "?" {
		t.Errorf("humanDurationSince(-1) = %q, want %q", got, "?")
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{45_900_000, "43.8 MB"},
		{2 * 1024 * 1024 * 1024, "2.0 GB"},
	}
	for _, tc := range tests {
		if got := humanBytes(tc.in); got != tc.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
