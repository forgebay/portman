// Package proc handles killing processes and reading per-process resource
// stats. Kill uses a graceful SIGTERM-then-SIGKILL escalation so well-behaved
// servers get a chance to shut down cleanly.
package proc

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// terminateGrace is how long we wait for a process to exit after SIGTERM before
// escalating to SIGKILL.
const terminateGrace = 3 * time.Second

// Stats holds the lightweight per-process metrics shown in the UI.
type Stats struct {
	CPU float64 // percent (since-start average; cheap, non-blocking)
	RSS uint64  // resident memory in bytes
}

// Kill terminates the process with the given pid. When force is false it sends
// SIGTERM and, if the process is still alive after terminateGrace, escalates to
// SIGKILL. When force is true it sends SIGKILL immediately.
//
// Killing a process owned by another user fails with a permission error, which
// is returned to the caller so the UI can surface it.
func Kill(pid int32, force bool) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d: %w", pid, err)
	}
	if force {
		return p.Kill()
	}
	if err := p.Terminate(); err != nil {
		return fmt.Errorf("SIGTERM %d: %w", pid, err)
	}

	deadline := time.Now().Add(terminateGrace)
	for time.Now().Before(deadline) {
		if running, _ := p.IsRunning(); !running {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return p.Kill() // escalate
}

// Read returns the current CPU and memory usage for a pid. It uses the cheap,
// non-blocking CPUPercent (a since-start average) so it is safe to call from a
// refresh loop.
func Read(pid int32) (Stats, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return Stats{}, err
	}
	var s Stats
	if cpu, err := p.CPUPercent(); err == nil {
		s.CPU = cpu
	}
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		s.RSS = mi.RSS
	}
	return s, nil
}
