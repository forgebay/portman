//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubLaunchctl replaces the real launchctl call for the duration of a test and
// returns a pointer to the recorded invocations. Without this a test would
// register a LaunchAgent on the machine running it.
func stubLaunchctl(t *testing.T) *[][]string {
	t.Helper()
	var calls [][]string
	orig := runLaunchctl
	runLaunchctl = func(args ...string) error {
		calls = append(calls, args)
		return nil
	}
	t.Cleanup(func() { runLaunchctl = orig })
	return &calls
}

// agentDir points HOME at a temp dir so plists land there, and returns it.
func agentDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir LaunchAgents: %v", err)
	}
	return dir
}

func writePlist(t *testing.T, dir, agentLabel string) string {
	t.Helper()
	p := filepath.Join(dir, agentLabel+".plist")
	if err := os.WriteFile(p, []byte("<plist/>"), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestMigrate_NoLegacy_LeavesEverythingAlone(t *testing.T) {
	dir := agentDir(t)
	stubLaunchctl(t)

	if err := Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// A user who never enabled launch-at-login must not get it switched on.
	if IsEnabled() {
		t.Error("Migrate enabled launch-at-login when no legacy agent existed")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected an empty LaunchAgents dir, got %d entries", len(entries))
	}
}

func TestMigrate_LegacyPresent_MovesToCurrentLabel(t *testing.T) {
	dir := agentDir(t)
	calls := stubLaunchctl(t)
	legacy := writePlist(t, dir, legacyLabels[0])

	if err := Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy plist still present after Migrate")
	}
	if !IsEnabled() {
		t.Error("launch-at-login should carry over to the new label")
	}
	// The old agent has to be unloaded, not just unlinked, or launchd keeps
	// running it until the next login.
	var unloaded bool
	for _, c := range *calls {
		if len(c) == 2 && c[0] == "unload" && strings.Contains(c[1], legacyLabels[0]) {
			unloaded = true
		}
	}
	if !unloaded {
		t.Errorf("expected launchctl unload of the legacy agent, got %v", *calls)
	}
}

func TestMigrate_BothPresent_KeepsCurrentUntouched(t *testing.T) {
	dir := agentDir(t)
	stubLaunchctl(t)
	legacy := writePlist(t, dir, legacyLabels[0])
	current := writePlist(t, dir, label)

	if err := Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy plist should be removed")
	}
	// Already-migrated installs must not have their plist rewritten.
	body, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current plist: %v", err)
	}
	if string(body) != "<plist/>" {
		t.Errorf("current plist was rewritten: %q", body)
	}
}

func TestEnableDisable_RoundTrip(t *testing.T) {
	agentDir(t)
	stubLaunchctl(t)

	if IsEnabled() {
		t.Fatal("should start disabled")
	}
	if err := Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !IsEnabled() {
		t.Error("Enable did not register the agent")
	}
	if err := Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if IsEnabled() {
		t.Error("Disable did not remove the agent")
	}
	// Disabling twice is what happens when the plist was removed by hand.
	if err := Disable(); err != nil {
		t.Errorf("Disable on an absent agent should be a no-op, got %v", err)
	}
}

func TestEnable_WritesLabelAndExecutablePath(t *testing.T) {
	dir := agentDir(t)
	stubLaunchctl(t)

	if err := Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, label+".plist"))
	if err != nil {
		t.Fatalf("read plist: %v", err)
	}
	if !strings.Contains(string(body), label) {
		t.Errorf("plist is missing the label %q: %s", label, body)
	}
	exe, _ := os.Executable()
	if !strings.Contains(string(body), filepath.Base(exe)) {
		t.Errorf("plist does not point at the running binary: %s", body)
	}
}
