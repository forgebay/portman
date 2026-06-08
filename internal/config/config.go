// Package config persists the small set of user settings (refresh cadence and
// the show-all-ports toggle) to a JSON file under the user config dir.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the persisted user settings.
type Config struct {
	RefreshSeconds int  `json:"refreshSeconds"`
	ShowAll        bool `json:"showAll"`
}

// Default returns the built-in defaults: 15s refresh, dev-servers only.
func Default() Config { return Config{RefreshSeconds: 15, ShowAll: false} }

// Path returns the config file location, e.g.
// ~/Library/Application Support/portman/config.json (macOS) or
// ~/.config/portman/config.json (Linux).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "portman", "config.json"), nil
}

// Load reads the config file, returning defaults (merged) when it is missing or
// invalid so the app always has a usable configuration.
func Load() Config {
	c := Default()
	p, err := Path()
	if err != nil {
		return c
	}
	return loadFrom(p, c)
}

// loadFrom is the testable core of Load.
func loadFrom(path string, def Config) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	c := def
	if json.Unmarshal(data, &c) != nil {
		return def
	}
	if c.RefreshSeconds <= 0 {
		c.RefreshSeconds = def.RefreshSeconds
	}
	return c
}

// Save writes the config to disk, creating the directory if needed.
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	return saveTo(p, c)
}

// saveTo is the testable core of Save.
func saveTo(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
