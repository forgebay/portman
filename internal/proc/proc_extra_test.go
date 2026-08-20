package proc

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// spawnSleeper starts a real child process that will sit idle until signalled,
// and returns its pid. Killing a real process is the only honest way to test
// the SIGTERM/SIGKILL escalation.
func spawnSleeper(t *testing.T) (*exec.Cmd, int32) {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return cmd, int32(cmd.Process.Pid)
}

// deadPID returns a pid that has already exited and been reaped.
func deadPID(t *testing.T) int32 {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run true: %v", err)
	}
	return int32(cmd.Process.Pid)
}

func TestRead_ReportsMemoryForALiveProcess(t *testing.T) {
	s, err := Read(int32(os.Getpid()))
	if err != nil {
		t.Fatalf("Read on self: %v", err)
	}
	// The test binary is definitely resident, so RSS must be non-zero. CPU is
	// a since-start average and can legitimately round to 0, so it is not
	// asserted on.
	if s.RSS == 0 {
		t.Error("RSS = 0 for the running test process")
	}
	if s.CPU < 0 {
		t.Errorf("CPU = %v, should never be negative", s.CPU)
	}
}

func TestRead_FailsOnAnExitedProcess(t *testing.T) {
	if _, err := Read(deadPID(t)); err == nil {
		t.Error("Read on an exited pid should return an error")
	}
}

func TestKill_ForceStopsTheProcessImmediately(t *testing.T) {
	cmd, pid := spawnSleeper(t)

	if err := Kill(pid, true); err != nil {
		t.Fatalf("Kill(force): %v", err)
	}
	if err := waitGone(cmd, 3*time.Second); err != nil {
		t.Error(err)
	}
}

func TestKill_GracefulStopsAProcessThatHonoursSIGTERM(t *testing.T) {
	cmd, pid := spawnSleeper(t)

	// sleep exits on SIGTERM, so this must return well inside terminateGrace
	// rather than escalating to SIGKILL.
	start := time.Now()
	if err := Kill(pid, false); err != nil {
		t.Fatalf("Kill(graceful): %v", err)
	}
	if elapsed := time.Since(start); elapsed >= terminateGrace {
		t.Errorf("graceful kill took %v, so it escalated instead of exiting on SIGTERM", elapsed)
	}
	if err := waitGone(cmd, 3*time.Second); err != nil {
		t.Error(err)
	}
}

func TestKill_FailsOnAnExitedProcess(t *testing.T) {
	if err := Kill(deadPID(t), false); err == nil {
		t.Error("Kill on an exited pid should return an error")
	}
}

// waitGone reaps the child and reports whether it actually terminated.
func waitGone(cmd *exec.Cmd, within time.Duration) error {
	done := make(chan error, 1)
	go func() { _, err := cmd.Process.Wait(); done <- err }()
	select {
	case <-done:
		return nil
	case <-time.After(within):
		return errStillRunning
	}
}

var errStillRunning = &stillRunning{}

type stillRunning struct{}

func (*stillRunning) Error() string { return "process was still running after the kill" }
