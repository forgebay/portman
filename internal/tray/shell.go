// Shelling out to the OS: opening URLs and folders, copying to the clipboard,
// and finding an editor. The command *selection* is split out as pure functions
// so the per-platform choices can be asserted in tests; only the actual
// exec.Start/Run calls are untested.
package tray

import (
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// openCommand returns the command that asks the OS to open target with its
// default handler.
func openCommand(goos, target string) (string, []string) {
	if goos == "darwin" {
		return "open", []string{target}
	}
	return "xdg-open", []string{target}
}

// clipboardCommand returns the command that reads stdin onto the clipboard.
// hasWaylandCopy selects wl-copy over xclip on Linux; callers resolve that by
// looking for the binary on PATH.
func clipboardCommand(goos string, hasWaylandCopy bool) (string, []string) {
	if goos == "darwin" {
		return "pbcopy", nil
	}
	if hasWaylandCopy {
		return "wl-copy", nil
	}
	return "xclip", []string{"-selection", "clipboard"}
}

// editorCandidates lists the CLI editors we try, in preference order, with the
// menu label to show when one is found.
var editorCandidates = []struct{ bin, label string }{
	{"code", "Open in VS Code"},
	{"cursor", "Open in Cursor"},
	{"subl", "Open in Sublime Text"},
}

// pickEditor returns the first candidate that available reports as installed.
// available is injected so the choice can be tested without touching PATH.
func pickEditor(available func(string) bool) (bin, label string) {
	for _, c := range editorCandidates {
		if available(c.bin) {
			return c.bin, c.label
		}
	}
	return "", ""
}

var (
	editorOnce  sync.Once
	editorBin   string
	editorLabel string
)

// editor resolves the editor once per process and caches the result.
func editor() (string, string) {
	editorOnce.Do(func() {
		editorBin, editorLabel = pickEditor(onPath)
	})
	return editorBin, editorLabel
}

func onPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// openInEditor opens dir in the detected editor.
func openInEditor(dir string) {
	bin, _ := editor()
	if bin == "" {
		return
	}
	_ = exec.Command(bin, dir).Start()
}

// openURL opens a URL or path with the OS default handler.
func openURL(target string) {
	name, args := openCommand(runtime.GOOS, target)
	_ = exec.Command(name, args...).Start()
}

// copyText puts s on the system clipboard.
func copyText(s string) error {
	name, args := clipboardCommand(runtime.GOOS, onPath("wl-copy"))
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}
