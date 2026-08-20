// Package ports lists the TCP dev-server ports currently in the LISTEN state
// along with the process, runtime, project and framework behind each one. It
// uses gopsutil so the same code works on macOS (libproc / lsof) and Linux
// (/proc).
package ports

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/forgebay/portman/internal/model"
	"github.com/forgebay/portman/internal/project"
	"github.com/forgebay/portman/internal/runtime"
)

// listener is a deduplicated (pid, port) pair in the LISTEN state.
type listener struct {
	PID  int32
	Port int
}

// meta holds the per-process fields that don't change over a process's life.
// They are cached by PID so we don't re-read manifests/cwd on every refresh.
type meta struct {
	name      string
	lang      model.Lang
	project   string
	framework string
	cwd       string
	createdMs int64
	exe       string // absolute path to the executable
}

var (
	cacheMu sync.Mutex
	cache   = map[int32]meta{}
)

// filterListening keeps only LISTEN sockets owned by a visible process
// (PID > 0) and collapses IPv4+IPv6 duplicates that share a (pid, port). It is
// pure so it can be unit-tested without touching the OS.
func filterListening(conns []psnet.ConnectionStat) []listener {
	seen := make(map[string]bool)
	out := make([]listener, 0, len(conns))
	for _, c := range conns {
		if c.Status != "LISTEN" || c.Pid <= 0 {
			continue
		}
		key := fmt.Sprintf("%d-%d", c.Pid, c.Laddr.Port)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, listener{PID: c.Pid, Port: int(c.Laddr.Port)})
	}
	return out
}

// List returns listening TCP ports sorted by port. By default it shows only dev
// servers (Node, Bun, Deno, Python, Go, Rust, Elixir, Ruby, PHP, Java, .NET,
// Ollama, ...) and hides OS/system processes; when showAll is true the dev-only
// filter is skipped. Each result is health-probed (does the port still accept
// connections?). Ports owned by other users (PID 0) are skipped upstream.
func List(showAll bool) ([]model.ListenPort, error) {
	conns, err := psnet.Connections("tcp") // tcp covers both IPv4 and IPv6
	if err != nil {
		return nil, fmt.Errorf("listing tcp connections: %w", err)
	}

	listeners := filterListening(conns)
	pruneCache(listeners)

	out := make([]model.ListenPort, 0, len(listeners))
	for _, l := range listeners {
		lp := enrich(l)
		if showAll || isDevServerEntry(lp) {
			out = append(out, lp)
		}
	}

	probeHealth(out)
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out, nil
}

// probeHealth concurrently checks whether each port still accepts TCP
// connections, setting Alive. Bounded by a short timeout so a refresh never
// stalls. No payload is sent — a successful connect is enough.
func probeHealth(ports []model.ListenPort) {
	var wg sync.WaitGroup
	for i := range ports {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(ports[i].Port))
			conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
			if err == nil {
				ports[i].Alive = true
				_ = conn.Close()
			}
		}(i)
	}
	wg.Wait()
}

// isDevServerEntry decides whether a listening process is a dev server worth
// showing.
//
// This used to also require that a project or framework had been detected. That
// hid real dev servers: a server started from a directory with no manifest, or
// one whose working directory could not be read, vanished from the list — so
// the menu came up empty while a server was plainly running, which reads as a
// broken app rather than a filter doing its job.
//
// The runtime allowlist already excludes OS daemons written in C and friends.
// What is left to exclude is runtimes bundled inside a packaged application,
// and that is a question about where the executable lives, not about whether a
// manifest happened to be nearby.
func isDevServerEntry(lp model.ListenPort) bool {
	if lp.Lang == model.Ollama {
		return true
	}
	if !runtime.IsDevServer(lp.Lang) {
		return false
	}
	return !belongsToPackagedApp(lp.Exe)
}

// packagedAppMarkers are path fragments that mean the executable ships inside
// an installed application or the OS, not somewhere a developer starts a server
// from. Electron apps and tools like Postman carry their own Node or Go runtime
// and open local ports; those are not the user's dev servers.
var packagedAppMarkers = []string{
	".app/contents/",    // macOS application bundle
	"/snap/",            // Linux snap
	"/var/lib/flatpak/", // Linux flatpak
	"/system/library/",  // macOS system frameworks and daemons
	"/usr/libexec/",
	"/usr/sbin/",
}

func belongsToPackagedApp(exe string) bool {
	if exe == "" {
		return false
	}
	e := strings.ToLower(exe)
	// A runtime vendored into a project — node_modules/electron ships a whole
	// Electron.app — belongs to the app the user is building, so it stays.
	if strings.Contains(e, "/node_modules/") {
		return false
	}
	for _, marker := range packagedAppMarkers {
		if strings.Contains(e, marker) {
			return true
		}
	}
	return false
}

// enrich attaches process metadata to a listener. The static parts (name,
// runtime, project, framework, cwd) come from a per-PID cache; CPU and memory
// are read fresh every call. Any per-field failure is tolerated.
func enrich(l listener) model.ListenPort {
	lp := model.ListenPort{Port: l.Port, PID: l.PID, Lang: model.Unknown}

	p, err := process.NewProcess(l.PID)
	if err != nil {
		return lp
	}

	m := lookupMeta(l.PID, p)
	lp.ProcName = m.name
	lp.Exe = m.exe
	lp.Lang = m.lang
	lp.Project = m.project
	lp.Framework = m.framework
	lp.Cwd = m.cwd
	lp.CreatedMs = m.createdMs

	if cpu, err := p.CPUPercent(); err == nil {
		lp.CPU = cpu
	}
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		lp.RSS = mi.RSS
	}
	return lp
}

// lookupMeta returns cached static metadata for pid, computing and caching it
// on a miss.
func lookupMeta(pid int32, p *process.Process) meta {
	cacheMu.Lock()
	if m, ok := cache[pid]; ok {
		cacheMu.Unlock()
		return m
	}
	cacheMu.Unlock()

	name, _ := p.Name()
	exe, _ := p.Exe()
	cmd, _ := p.CmdlineSlice()
	cwd := processCwd(p)
	proj, fw := project.Detect(cwd, cmd)
	created, _ := p.CreateTime() // ms since epoch; 0 on error

	m := meta{
		name:      name,
		exe:       exe,
		lang:      runtime.Detect(name, exe, cmd),
		project:   proj,
		framework: fw,
		cwd:       cwd,
		createdMs: created,
	}
	cacheMu.Lock()
	cache[pid] = m
	cacheMu.Unlock()
	return m
}

// pruneCache drops cache entries for PIDs no longer listening, so the map
// doesn't grow without bound across refreshes.
func pruneCache(listeners []listener) {
	live := make(map[int32]bool, len(listeners))
	for _, l := range listeners {
		live[l.PID] = true
	}
	cacheMu.Lock()
	for pid := range cache {
		if !live[pid] {
			delete(cache, pid)
		}
	}
	cacheMu.Unlock()
}

// processCwd returns the process working directory, trying gopsutil first
// (libproc on macOS, /proc on Linux) and falling back to lsof on macOS where a
// non-CGO build or permission quirk leaves Cwd unimplemented.
func processCwd(p *process.Process) string {
	if cwd, err := p.Cwd(); err == nil && cwd != "" {
		return cwd
	}
	return lsofCwd(p.Pid)
}

func lsofCwd(pid int32) string {
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(int(pid)), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "n") {
			return ln[1:]
		}
	}
	return ""
}
