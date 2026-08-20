package project

import "testing"

// frameworkFromCmd is what names the framework for runtimes with no manifest to
// read (Python, Ruby, PHP, JVM), so every launcher it claims to recognise needs
// to actually match. A silent miss shows the user a bare runtime label.
func TestFrameworkFromCmd_RecognisesEveryLauncher(t *testing.T) {
	tests := []struct {
		name string
		cmd  []string
		want string
	}{
		{"next dev", []string{"node", "/app/node_modules/next/dist/bin/next", "dev"}, "Next.js"},
		{"nuxt", []string{"node", "/app/node_modules/nuxt/bin/nuxt.js"}, "Nuxt"},
		{"vite", []string{"node", "/app/node_modules/.bin/vite"}, "Vite"},
		{"remix", []string{"node", "/app/node_modules/@remix-run/dev/dist/cli.js", "remix", "dev"}, "Remix"},
		{"astro", []string{"node", "/app/node_modules/astro/astro.js", "dev"}, "Astro"},
		{"webpack", []string{"node", "/app/node_modules/webpack-dev-server/bin/webpack-dev-server.js"}, "Webpack"},
		{"angular cli", []string{"node", "/app/node_modules/.bin/ng", "serve"}, "Angular"},
		{"uvicorn", []string{"python3", "/app/.venv/bin/uvicorn", "main:app"}, "FastAPI"},
		{"gunicorn", []string{"python3", "/app/.venv/bin/gunicorn", "wsgi:app"}, "Gunicorn"},
		{"django manage.py", []string{"python3", "manage.py", "runserver"}, "Django"},
		{"flask", []string{"python3", "-m", "flask", "run"}, "Flask"},
		{"rails", []string{"ruby", "bin/rails", "server"}, "Rails"},
		{"laravel artisan", []string{"php", "artisan", "serve"}, "Laravel"},
		{"spring boot", []string{"java", "-jar", "-Dspring-boot.run.profiles=dev", "app.jar"}, "Spring Boot"},
		{"phoenix", []string{"mix", "phx.server"}, "Phoenix"},
		{"nothing recognisable", []string{"./my-server", "--port", "8080"}, ""},
		{"empty argv", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := frameworkFromCmd(tc.cmd); got != tc.want {
				t.Errorf("frameworkFromCmd(%v) = %q, want %q", tc.cmd, got, tc.want)
			}
		})
	}
}

// containsTok exists to stop a framework name matching inside an unrelated
// word — "ng" must not fire on every path containing those two letters.
func TestFrameworkFromCmd_DoesNotMatchInsideWords(t *testing.T) {
	tests := [][]string{
		{"/usr/local/bin/nginx", "-g", "daemon off;"},
		{"node", "/app/nextcloud-sync/index.js"},
		{"python3", "/app/managements/report.py"},
	}
	for _, cmd := range tests {
		if got := frameworkFromCmd(cmd); got == "Angular" || got == "Next.js" {
			t.Errorf("frameworkFromCmd(%v) = %q — matched inside an unrelated word", cmd, got)
		}
	}
}
