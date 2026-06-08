package proc

import (
	"os/exec"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// TestKillGraceful spawns a long-running child process and confirms Kill
// terminates it.
func TestKillGraceful(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := int32(cmd.Process.Pid)
	defer cmd.Process.Kill() // safety net

	if err := Kill(pid, false); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	// Reap the child so it doesn't linger as a zombie and confirm exit.
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("process still running after Kill")
	}

	if running, _ := (&process.Process{Pid: pid}).IsRunning(); running {
		t.Fatal("process reported running after Kill")
	}
}

func TestKillForce(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := int32(cmd.Process.Pid)
	defer cmd.Process.Kill()

	if err := Kill(pid, true); err != nil {
		t.Fatalf("Kill force: %v", err)
	}
	_ = cmd.Wait()
}

func TestKillUnknownPID(t *testing.T) {
	// PID that is extremely unlikely to exist.
	if err := Kill(2147483600, false); err == nil {
		t.Fatal("expected error killing nonexistent pid")
	}
}
