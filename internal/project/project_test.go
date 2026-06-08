package project

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetect_PackageJSON_Next(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"name":"my-shop","dependencies":{"next":"14.0.0","react":"18"}}`)
	proj, fw := Detect(dir, []string{"node", "/x/node_modules/.bin/next", "dev"})
	if proj != "my-shop" {
		t.Errorf("project = %q, want my-shop", proj)
	}
	if fw != "Next.js" {
		t.Errorf("framework = %q, want Next.js", fw)
	}
}

func TestDetect_Vite_FromDeps(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"name":"landing","devDependencies":{"vite":"5"}}`)
	proj, fw := Detect(dir, []string{"node", "server.js"}) // cmd gives no hint
	if proj != "landing" || fw != "Vite" {
		t.Errorf("got (%q,%q), want (landing,Vite)", proj, fw)
	}
}

func TestDetect_GoMod(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module github.com/acme/api\n\ngo 1.22\n")
	proj, _ := Detect(dir, []string{"./api"})
	if proj != "api" {
		t.Errorf("project = %q, want api", proj)
	}
}

func TestDetect_Pyproject_Uvicorn(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "pyproject.toml", "[project]\nname = \"billing\"\nversion = \"0.1.0\"\n")
	proj, fw := Detect(dir, []string{"python", "-m", "uvicorn", "app:app"})
	if proj != "billing" || fw != "FastAPI" {
		t.Errorf("got (%q,%q), want (billing,FastAPI)", proj, fw)
	}
}

func TestDetect_WalkUp(t *testing.T) {
	root := t.TempDir()
	write(t, root, "package.json", `{"name":"monorepo"}`)
	sub := filepath.Join(root, "apps", "web")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	proj, _ := Detect(sub, nil)
	if proj != "monorepo" {
		t.Errorf("project = %q, want monorepo (walk up)", proj)
	}
}

func TestDetect_NoManifest_FrameworkStillFromCmd(t *testing.T) {
	dir := t.TempDir()
	proj, fw := Detect(dir, []string{"node", "/x/.bin/astro", "dev"})
	if proj != "" {
		t.Errorf("project = %q, want empty", proj)
	}
	if fw != "Astro" {
		t.Errorf("framework = %q, want Astro", fw)
	}
}

func TestDetect_Cargo_Axum(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Cargo.toml", "[package]\nname = \"edge\"\n\n[dependencies]\naxum = \"0.7\"\n")
	proj, fw := Detect(dir, []string{"./target/release/edge"})
	if proj != "edge" || fw != "Axum" {
		t.Errorf("got (%q,%q), want (edge,Axum)", proj, fw)
	}
}

func TestDetect_Phoenix_FromCmd(t *testing.T) {
	dir := t.TempDir()
	_, fw := Detect(dir, []string{"/usr/bin/beam.smp", "-mode", "embedded", "phx.server"})
	if fw != "Phoenix" {
		t.Errorf("framework = %q, want Phoenix", fw)
	}
}

func TestDetect_SpringBoot_FromCmd(t *testing.T) {
	dir := t.TempDir()
	_, fw := Detect(dir, []string{"java", "-jar", "app.jar", "--spring-boot.run"})
	if fw != "Spring Boot" {
		t.Errorf("framework = %q, want Spring Boot", fw)
	}
}

func TestDetect_Gatsby_FromDeps(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"name":"blog","dependencies":{"gatsby":"5"}}`)
	proj, fw := Detect(dir, []string{"node", "server.js"})
	if proj != "blog" || fw != "Gatsby" {
		t.Errorf("got (%q,%q), want (blog,Gatsby)", proj, fw)
	}
}

func TestDetect_EmptyCwd(t *testing.T) {
	proj, fw := Detect("", nil)
	if proj != "" || fw != "" {
		t.Errorf("got (%q,%q), want empty", proj, fw)
	}
}
