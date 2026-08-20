package ports

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// The unit tests for the entry filter classify executable paths written by hand,
// which is how 0.4.1 shipped a filter that hid every Python dev server: the
// invented paths never looked like the real interpreter. These tests start real
// processes and read the paths off them, so the filter is exercised against
// whatever this machine actually reports.

// freePort asks the kernel for an unused port and returns it.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// startServer runs cmd from dir, waits until the port accepts a connection, and
// registers cleanup.
func startServer(t *testing.T, dir string, port int, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start %s on this machine: %v", name, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s never started listening on %d", name, port)
}

// listed reports whether List surfaced the given port.
func listed(t *testing.T, port int) bool {
	t.Helper()
	// The metadata cache is keyed by pid and outlives a single List call, so
	// clear it to make each case independent.
	cacheMu.Lock()
	cache = map[int32]meta{}
	cacheMu.Unlock()

	list, err := List(false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, p := range list {
		if p.Port == port {
			return true
		}
	}
	return false
}

func TestList_ShowsNodeServerWithoutAnyManifest(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	// A bare directory: no package.json here or in any parent inside it.
	dir := t.TempDir()
	port := freePort(t)
	script := filepath.Join(dir, "server.js")
	body := fmt.Sprintf("require('http').createServer((_,r)=>r.end('ok')).listen(%d,'127.0.0.1');", port)
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	startServer(t, dir, port, "node", script)

	if !listed(t, port) {
		t.Errorf("a node server on %d with no manifest was not listed — this is the 0.4.0 bug", port)
	}
}

func TestList_ShowsPythonServerWithoutAnyManifest(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
	// On macOS the framework Python's binary sits inside Python.app/Contents,
	// which 0.4.1 misread as an installed application.
	dir := t.TempDir()
	port := freePort(t)
	script := filepath.Join(dir, "server.py")
	body := "import sys\n" +
		"from http.server import HTTPServer, BaseHTTPRequestHandler\n" +
		"class H(BaseHTTPRequestHandler):\n" +
		"    def do_GET(self):\n" +
		"        self.send_response(200); self.end_headers(); self.wfile.write(b'ok')\n" +
		"    def log_message(self, *a): pass\n" +
		fmt.Sprintf("HTTPServer(('127.0.0.1', %d), H).serve_forever()\n", port)
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	startServer(t, dir, port, "python3", script)

	if !listed(t, port) {
		t.Errorf("a python3 server on %d was not listed — this is the 0.4.1 regression", port)
	}
}
