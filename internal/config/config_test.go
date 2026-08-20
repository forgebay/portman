package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "portman", "config.json")
	want := Config{RefreshSeconds: 5, ShowAll: true}
	if err := saveTo(p, want); err != nil {
		t.Fatalf("saveTo: %v", err)
	}
	got := loadFrom(p, Default())
	if got != want {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope.json")
	if got := loadFrom(p, Default()); got != Default() {
		t.Errorf("missing file = %+v, want default %+v", got, Default())
	}
}

func TestLoadInvalidRefreshFallsBack(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	if err := saveTo(p, Config{RefreshSeconds: 0, ShowAll: true}); err != nil {
		t.Fatal(err)
	}
	got := loadFrom(p, Default())
	if got.RefreshSeconds != Default().RefreshSeconds {
		t.Errorf("RefreshSeconds = %d, want default %d", got.RefreshSeconds, Default().RefreshSeconds)
	}
	if !got.ShowAll {
		t.Error("ShowAll should be preserved")
	}
}

func TestPath_LivesUnderTheUserConfigDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("no user config dir on this platform: %v", err)
	}
	want := filepath.Join(base, "portman", "config.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestSaveLoad_ThroughPublicAPI(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// A fresh machine has no file yet, so Load must hand back the defaults.
	if got := Load(); got != Default() {
		t.Errorf("Load on a clean machine = %+v, want defaults %+v", got, Default())
	}

	want := Config{RefreshSeconds: 30, ShowAll: true}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := Load(); got != want {
		t.Errorf("Load after Save = %+v, want %+v", got, want)
	}
}

func TestSaveTo_ReportsAnUnusableDirectory(t *testing.T) {
	// Put a regular file where the config directory would have to go, so
	// MkdirAll cannot succeed.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if err := saveTo(filepath.Join(blocker, "portman", "config.json"), Default()); err == nil {
		t.Error("saveTo should fail when the parent path is a file")
	}
}

func TestLoadFrom_MalformedJSONFallsBackToDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := loadFrom(p, Default()); got != Default() {
		t.Errorf("loadFrom on malformed JSON = %+v, want defaults", got)
	}
}
