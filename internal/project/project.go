// Package project detects the project name and web framework behind a running
// dev server. It walks up from the process working directory to the nearest
// manifest (package.json, go.mod, pyproject.toml, ...) for the name, and uses
// the command line plus package.json dependencies to name the framework.
//
// Detection is a pure function of (cwd, cmdline) so it is unit-testable against
// fixture directories.
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// maxWalkUp bounds how many parent directories we search for a manifest.
const maxWalkUp = 6

// Detect returns the project name and framework for a process whose working
// directory is cwd and whose argv is cmd. Either result may be "" when unknown.
func Detect(cwd string, cmd []string) (project, framework string) {
	// Framework from the command line first — it's cheap and usually decisive
	// (e.g. "next dev", "vite", "uvicorn app:app").
	framework = frameworkFromCmd(cmd)

	dir, manifest := findManifest(cwd)
	if manifest == "" {
		return project, framework
	}

	project = nameFromManifest(manifest)
	if project == "" {
		project = filepath.Base(dir)
	}

	// If the command line didn't reveal a framework, try manifest dependencies.
	if framework == "" {
		switch filepath.Base(manifest) {
		case "package.json":
			framework = frameworkFromPackageJSON(manifest)
		case "Cargo.toml":
			framework = frameworkFromCargo(manifest)
		}
	}
	return project, framework
}

// manifestFiles are searched in priority order within each directory.
var manifestFiles = []string{
	"package.json", "go.mod", "pyproject.toml", "Cargo.toml",
	"composer.json", "deno.json", "deno.jsonc", "Gemfile",
	"pom.xml", "build.gradle",
}

// findManifest walks up from cwd looking for the first known manifest.
func findManifest(cwd string) (dir, manifest string) {
	if cwd == "" {
		return "", ""
	}
	d := cwd
	for i := 0; i < maxWalkUp; i++ {
		for _, name := range manifestFiles {
			p := filepath.Join(d, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return d, p
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			break // reached filesystem root
		}
		d = parent
	}
	return "", ""
}

// nameFromManifest extracts the project name from a manifest file.
func nameFromManifest(manifest string) string {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return ""
	}
	switch filepath.Base(manifest) {
	case "package.json", "composer.json", "deno.json", "deno.jsonc":
		var m struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(data, &m)
		// composer/npm scoped names like "@scope/app" → keep the last segment.
		return lastSegment(m.Name)
	case "go.mod":
		// "module github.com/user/app" → "app"
		if mod := lineValue(data, "module"); mod != "" {
			return lastSegment(mod)
		}
	case "pyproject.toml", "Cargo.toml":
		// name = "app"  (under [project]/[tool.poetry]/[package])
		return tomlName(data)
	}
	return ""
}

// frameworkFromCmd inspects argv for a recognizable framework launcher.
func frameworkFromCmd(cmd []string) string {
	joined := strings.ToLower(strings.Join(cmd, " "))
	switch {
	case containsTok(joined, "next"):
		return "Next.js"
	case containsTok(joined, "nuxt"):
		return "Nuxt"
	case containsTok(joined, "vite"):
		return "Vite"
	case strings.Contains(joined, "remix"):
		return "Remix"
	case containsTok(joined, "astro"):
		return "Astro"
	case strings.Contains(joined, "webpack"):
		return "Webpack"
	case strings.Contains(joined, "ng ") || strings.HasSuffix(joined, "/ng") || containsTok(joined, "ng"):
		return "Angular"
	case strings.Contains(joined, "uvicorn"):
		return "FastAPI"
	case strings.Contains(joined, "gunicorn"):
		return "Gunicorn"
	case strings.Contains(joined, "manage.py") || containsTok(joined, "django"):
		return "Django"
	case containsTok(joined, "flask"):
		return "Flask"
	case strings.Contains(joined, "rails"):
		return "Rails"
	case strings.Contains(joined, "artisan"):
		return "Laravel"
	case strings.Contains(joined, "spring-boot") || strings.Contains(joined, "org.springframework") || containsTok(joined, "bootrun"):
		return "Spring Boot"
	case strings.Contains(joined, "phx.server") || containsTok(joined, "phoenix"):
		return "Phoenix"
	}
	return ""
}

// pkgFrameworks maps a dependency name (prefix) to a framework label, in order.
var pkgFrameworks = []struct{ dep, name string }{
	{"next", "Next.js"},
	{"nuxt", "Nuxt"},
	{"@remix-run/", "Remix"},
	{"@sveltejs/kit", "SvelteKit"},
	{"@redwoodjs/", "RedwoodJS"},
	{"@builder.io/qwik", "Qwik"},
	{"solid-start", "SolidStart"},
	{"gatsby", "Gatsby"},
	{"astro", "Astro"},
	{"vite", "Vite"},
	{"@angular/core", "Angular"},
	{"react-scripts", "CRA"},
	{"@nestjs/core", "NestJS"},
	{"webpack", "Webpack"},
	{"fastify", "Fastify"},
	{"express", "Express"},
}

// cargoFrameworks maps a Rust crate dependency to a framework label.
var cargoFrameworks = []struct{ dep, name string }{
	{"actix-web", "Actix"},
	{"axum", "Axum"},
	{"rocket", "Rocket"},
	{"warp", "Warp"},
}

// frameworkFromCargo line-scans Cargo.toml for a known web-framework crate.
func frameworkFromCargo(manifest string) string {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return ""
	}
	text := string(data)
	for _, f := range cargoFrameworks {
		// Match a dependency line like `axum = "0.7"` or `axum = { version = ... }`.
		if strings.Contains(text, f.dep+" =") || strings.Contains(text, f.dep+"=") {
			return f.name
		}
	}
	return ""
}

// frameworkFromPackageJSON reads dependencies + devDependencies and maps them.
func frameworkFromPackageJSON(manifest string) string {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return ""
	}
	var m struct {
		Deps    map[string]string `json:"dependencies"`
		DevDeps map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	has := func(prefix string) bool {
		for k := range m.Deps {
			if k == prefix || strings.HasPrefix(k, prefix) {
				return true
			}
		}
		for k := range m.DevDeps {
			if k == prefix || strings.HasPrefix(k, prefix) {
				return true
			}
		}
		return false
	}
	for _, f := range pkgFrameworks {
		if has(f.dep) {
			return f.name
		}
	}
	return ""
}

// --- small parsing helpers ---

func lastSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// lineValue returns the token after the given key on a line like "key value".
func lineValue(data []byte, key string) string {
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, key+" ") || strings.HasPrefix(ln, key+"\t") {
			fields := strings.Fields(ln)
			if len(fields) >= 2 {
				return strings.Trim(fields[1], `"`)
			}
		}
	}
	return ""
}

// tomlName finds the first `name = "..."` line (works for [project],
// [tool.poetry] and [package] tables — the first name is the project's).
func tomlName(data []byte) string {
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "name") && strings.Contains(ln, "=") {
			v := ln[strings.Index(ln, "=")+1:]
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

// containsTok reports whether s contains tok as a whole whitespace/slash token,
// avoiding matches inside unrelated words.
func containsTok(s, tok string) bool {
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '/' || r == '\\' }) {
		if f == tok {
			return true
		}
	}
	return false
}
