package tray

import (
	"reflect"
	"testing"
)

func TestOpenCommand(t *testing.T) {
	if name, args := openCommand("darwin", "http://localhost:3000"); name != "open" ||
		!reflect.DeepEqual(args, []string{"http://localhost:3000"}) {
		t.Errorf("darwin: got %q %v", name, args)
	}
	if name, args := openCommand("linux", "/home/x/proj"); name != "xdg-open" ||
		!reflect.DeepEqual(args, []string{"/home/x/proj"}) {
		t.Errorf("linux: got %q %v", name, args)
	}
}

func TestClipboardCommand(t *testing.T) {
	if name, args := clipboardCommand("darwin", false); name != "pbcopy" || args != nil {
		t.Errorf("darwin: got %q %v", name, args)
	}
	// Wayland sessions have no working xclip, so wl-copy must win when present.
	if name, _ := clipboardCommand("linux", true); name != "wl-copy" {
		t.Errorf("linux wayland: got %q, want wl-copy", name)
	}
	name, args := clipboardCommand("linux", false)
	if name != "xclip" || !reflect.DeepEqual(args, []string{"-selection", "clipboard"}) {
		t.Errorf("linux x11: got %q %v", name, args)
	}
}

func TestPickEditor(t *testing.T) {
	installed := func(set ...string) func(string) bool {
		return func(bin string) bool {
			for _, s := range set {
				if s == bin {
					return true
				}
			}
			return false
		}
	}

	// Preference order matters: with both present, VS Code wins.
	if bin, label := pickEditor(installed("code", "subl")); bin != "code" || label != "Open in VS Code" {
		t.Errorf("got %q / %q, want code", bin, label)
	}
	if bin, label := pickEditor(installed("subl")); bin != "subl" || label != "Open in Sublime Text" {
		t.Errorf("got %q / %q, want subl", bin, label)
	}
	if bin, label := pickEditor(installed()); bin != "" || label != "" {
		t.Errorf("nothing installed should yield empty, got %q / %q", bin, label)
	}
}
