package config

import (
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
