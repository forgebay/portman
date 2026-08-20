package ports

import (
	"testing"

	"github.com/forgebay/portman/internal/model"
)

func TestIsDevServerEntry(t *testing.T) {
	tests := []struct {
		name string
		in   model.ListenPort
		want bool
	}{
		{
			// The bug this filter used to have: a real server started from a
			// directory with no package.json disappeared from the menu, so
			// portman came up empty while the server was running.
			name: "dev runtime with no project detected is still a dev server",
			in:   model.ListenPort{Lang: model.Node, Exe: "/Users/x/.nvm/versions/node/v24.0.2/bin/node"},
			want: true,
		},
		{
			name: "dev runtime with a project detected",
			in:   model.ListenPort{Lang: model.Node, Project: "acme-storefront", Framework: "Next.js", Exe: "/Users/x/.nvm/versions/node/v24.0.2/bin/node"},
			want: true,
		},
		{
			name: "unreadable working directory does not hide the server",
			in:   model.ListenPort{Lang: model.Python, Cwd: "", Exe: "/opt/homebrew/bin/python3.13"},
			want: true,
		},
		{
			name: "compiled binary from a project directory",
			in:   model.ListenPort{Lang: model.Go, Exe: "/Users/x/code/inventory-api/bin/inventory-api"},
			want: true,
		},
		{
			name: "Ollama shows even though it is not a project",
			in:   model.ListenPort{Lang: model.Ollama, Exe: "/usr/local/bin/ollama"},
			want: true,
		},
		{
			name: "runtime bundled inside a macOS app is not the user's server",
			in:   model.ListenPort{Lang: model.Node, Exe: "/Applications/Postman.app/Contents/MacOS/PostmanEngine"},
			want: false,
		},
		{
			// Electron vendored into a project is the app the user is building.
			name: "Electron under node_modules stays visible",
			in:   model.ListenPort{Lang: model.Node, Exe: "/Users/x/code/my-app/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron"},
			want: true,
		},
		{
			name: "macOS system daemon",
			in:   model.ListenPort{Lang: model.Python, Exe: "/usr/libexec/some-apple-daemon"},
			want: false,
		},
		{
			name: "snap-packaged app",
			in:   model.ListenPort{Lang: model.Node, Exe: "/snap/some-app/current/bin/node"},
			want: false,
		},
		{
			name: "non-dev runtime is excluded by the allowlist",
			in:   model.ListenPort{Lang: model.Native, Exe: "/usr/bin/some-c-daemon"},
			want: false,
		},
		{
			name: "unknown runtime",
			in:   model.ListenPort{Lang: model.Unknown},
			want: false,
		},
		{
			name: "empty executable path does not exclude a dev runtime",
			in:   model.ListenPort{Lang: model.Node, Exe: ""},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDevServerEntry(tc.in); got != tc.want {
				t.Errorf("isDevServerEntry(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
