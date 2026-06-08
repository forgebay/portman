// Package runtime detects the programming language / runtime of a process from
// its name, executable path and command line. The detection is a pure function
// so it is fully unit-testable without a running process.
package runtime

import (
	"debug/buildinfo"
	"path/filepath"
	"strings"

	"github.com/lanvu/portman/internal/model"
)

// Detect maps a process to a runtime. Order matters: interpreters and launchers
// are matched first (they tell us the real runtime even when the entrypoint is
// a script), then we fall back to inspecting a compiled binary.
//
//   - name: process name, e.g. "node", "python3.12", "java"
//   - exe:  absolute path to the executable (may be empty if unreadable)
//   - cmd:  full argv (may be empty)
func Detect(name, exe string, cmd []string) model.Lang {
	if name == "" {
		return model.Unknown
	}
	base := strings.ToLower(filepath.Base(name))

	switch {
	// JS runtimes — exact matches so "bundle" (Ruby) isn't mistaken for "bun".
	case base == "node" || base == "nodejs" || strings.HasPrefix(base, "node-"):
		return model.Node
	case base == "bun":
		return model.Bun
	case base == "deno":
		return model.Deno
	// Ollama is a Go binary; match by name before the Go buildinfo check below.
	case base == "ollama":
		return model.Ollama
	case strings.HasPrefix(base, "python"), base == "py", base == "uvicorn", base == "gunicorn":
		return model.Python
	case base == "java" || base == "jvm":
		return model.Java
	case base == "ruby" || strings.HasPrefix(base, "ruby"), base == "puma", base == "unicorn", base == "rails", base == "bundle":
		return model.Ruby
	case strings.HasPrefix(base, "php"):
		return model.PHP
	case base == "dotnet" || base == "mono":
		return model.DotNet
	}

	// No interpreter matched: most likely a compiled binary. Try to positively
	// identify a Go binary via its embedded build info; this also covers Go
	// dev servers whose name is the module binary.
	if exe != "" {
		if _, err := buildinfo.ReadFile(exe); err == nil {
			return model.Go
		}
	}

	// Unknown compiled binary (Rust, C/C++, statically-linked, etc.).
	if base != "" {
		return model.Native
	}
	return model.Unknown
}

// devLangs is the set of runtimes we consider "dev servers" and show in the UI.
// Everything else (Native/Unknown — i.e. OS daemons and other compiled apps) is
// hidden unless a project was detected for it.
var devLangs = map[model.Lang]bool{
	model.Node:   true,
	model.Bun:    true,
	model.Deno:   true,
	model.Python: true,
	model.Java:   true,
	model.Ruby:   true,
	model.PHP:    true,
	model.DotNet: true,
	model.Go:     true,
	model.Ollama: true,
}

// IsDevServer reports whether a runtime belongs to the dev-server allowlist.
func IsDevServer(l model.Lang) bool { return devLangs[l] }
