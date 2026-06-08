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
	case hasAny(base, "node", "nodejs", "deno", "bun"):
		return model.Node
	case strings.HasPrefix(base, "python"), base == "py", base == "uvicorn", base == "gunicorn":
		return model.Python
	case hasAny(base, "java", "jvm"):
		return model.Java
	case hasAny(base, "ruby", "puma", "unicorn", "rails"):
		return model.Ruby
	case strings.HasPrefix(base, "php"):
		return model.PHP
	case hasAny(base, "dotnet", "mono"):
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

// hasAny reports whether base equals or contains any of the candidates. We use
// containment so wrappers like "node-wrapper" still resolve, while the exact
// matches above guard against false positives.
func hasAny(base string, candidates ...string) bool {
	for _, c := range candidates {
		if base == c || strings.Contains(base, c) {
			return true
		}
	}
	return false
}
