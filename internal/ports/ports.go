// Package ports lists the TCP ports currently in the LISTEN state along with
// the process that owns each one. It uses gopsutil so the same code works on
// macOS (which shells out to lsof) and Linux (which reads /proc).
package ports

import (
	"fmt"
	"sort"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/lanvu/portman/internal/model"
	"github.com/lanvu/portman/internal/runtime"
)

// listener is a deduplicated (pid, port) pair in the LISTEN state.
type listener struct {
	PID  int32
	Port int
}

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

// List returns the listening TCP ports owned by processes visible to the
// current user, sorted by port number. Ports owned by other users or by the
// system (which report PID 0 without elevation) are skipped by design — portman
// manages your own processes.
func List() ([]model.ListenPort, error) {
	conns, err := psnet.Connections("tcp") // tcp covers both IPv4 and IPv6
	if err != nil {
		return nil, fmt.Errorf("listing tcp connections: %w", err)
	}

	listeners := filterListening(conns)
	out := make([]model.ListenPort, 0, len(listeners))
	for _, l := range listeners {
		out = append(out, enrich(l))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out, nil
}

// enrich attaches process metadata (name, runtime, CPU, memory) to a listener.
// Any per-field failure is tolerated so a partially-described port is still
// shown rather than dropped.
func enrich(l listener) model.ListenPort {
	lp := model.ListenPort{Port: l.Port, PID: l.PID, Lang: model.Unknown}

	p, err := process.NewProcess(l.PID)
	if err != nil {
		return lp
	}
	name, _ := p.Name()
	exe, _ := p.Exe()
	cmd, _ := p.CmdlineSlice()
	lp.ProcName = name
	lp.Lang = runtime.Detect(name, exe, cmd)
	if cpu, err := p.CPUPercent(); err == nil {
		lp.CPU = cpu
	}
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		lp.RSS = mi.RSS
	}
	return lp
}
