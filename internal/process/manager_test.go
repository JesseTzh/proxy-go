package process

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestManagedProcessCapturesBoundedOutputAndExitError(t *testing.T) {
	proc := &ManagedProcess{
		Name:          "test-process",
		Path:          "/bin/sh",
		Args:          []string{"-c", "printf 'old-output-that-should-be-truncated'; printf 'stderr-detail\\n' >&2; exit 7"},
		LogLimitBytes: 64,
	}

	if err := proc.Start(context.Background()); err != nil {
		t.Fatalf("start process: %v", err)
	}
	waitForStopped(t, proc)

	logs := strings.Join(proc.Logs(), "\n")
	if strings.Contains(logs, "old-output-that-should-be-truncated") {
		t.Fatalf("expected old output to be truncated, logs:\n%s", logs)
	}
	if !strings.Contains(logs, "stderr-detail") {
		t.Fatalf("expected stderr output in logs:\n%s", logs)
	}
	if !strings.Contains(logs, "process exited: exit status 7") {
		t.Fatalf("expected exit error in logs:\n%s", logs)
	}
}

func TestManagedProcessOutlivesStartContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	proc := &ManagedProcess{
		Name: "long-running-process",
		Path: "/bin/sh",
		Args: []string{"-c", "trap 'exit 0' INT; while :; do :; done"},
	}

	if err := proc.Start(ctx); err != nil {
		t.Fatalf("start process: %v", err)
	}
	cancel()

	time.Sleep(50 * time.Millisecond)
	status := proc.Status()
	if running, _ := status["running"].(bool); !running {
		t.Fatalf("expected process to outlive start context, status: %#v", status)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := proc.Stop(stopCtx); err != nil {
		t.Fatalf("stop process: %v", err)
	}
}

func waitForStopped(t *testing.T, proc *ManagedProcess) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := proc.Status()
		if running, _ := status["running"].(bool); !running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process did not stop before deadline")
}
