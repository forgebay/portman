package runtime

import (
	"testing"

	"github.com/lanvu/portman/internal/model"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		proc string
		exe  string
		cmd  []string
		want model.Lang
	}{
		{"node", "node", "/usr/local/bin/node", []string{"node", "server.js"}, model.Node},
		{"nodejs alias", "nodejs", "", nil, model.Node},
		{"deno", "deno", "", nil, model.Deno},
		{"bun", "bun", "", nil, model.Bun},
		{"ollama", "ollama", "/usr/local/bin/ollama", nil, model.Ollama},
		{"bundle is ruby not bun", "bundle", "", nil, model.Ruby},
		{"python3", "python3.12", "/usr/bin/python3.12", nil, model.Python},
		{"uvicorn", "uvicorn", "", nil, model.Python},
		{"java", "java", "", []string{"java", "-jar", "app.jar"}, model.Java},
		{"ruby", "ruby", "", nil, model.Ruby},
		{"puma", "puma", "", nil, model.Ruby},
		{"php", "php-fpm", "", nil, model.PHP},
		{"dotnet", "dotnet", "", nil, model.DotNet},
		{"unknown compiled with no exe", "myserver", "", nil, model.Native},
		{"empty", "", "", nil, model.Unknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Detect(c.proc, c.exe, c.cmd); got != c.want {
				t.Errorf("Detect(%q,%q,%v) = %q, want %q", c.proc, c.exe, c.cmd, got, c.want)
			}
		})
	}
}

func TestIsDevServer(t *testing.T) {
	for _, l := range []model.Lang{model.Node, model.Bun, model.Deno, model.Python, model.Go, model.Ollama, model.Ruby, model.PHP, model.Java, model.DotNet} {
		if !IsDevServer(l) {
			t.Errorf("IsDevServer(%q) = false, want true", l)
		}
	}
	for _, l := range []model.Lang{model.Native, model.Unknown} {
		if IsDevServer(l) {
			t.Errorf("IsDevServer(%q) = true, want false", l)
		}
	}
}
