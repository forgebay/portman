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
		{"deno", "deno", "", nil, model.Node},
		{"bun", "bun", "", nil, model.Node},
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
