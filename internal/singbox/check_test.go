package singbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/process"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestCheckUsesSingBoxCheckCommand(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "args.log")
	binary := fakeSingBoxBinary(t, logPath, 0)
	service := &Service{Binary: binary, Proc: &process.ManagedProcess{Name: "sing-box", Path: binary}}

	if err := service.Check(context.Background(), "/tmp/config.json"); err != nil {
		t.Fatalf("check sing-box config: %v", err)
	}
	args, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	want := "check -c /tmp/config.json"
	if strings.TrimSpace(string(args)) != want {
		t.Fatalf("sing-box args = %q, want %q", strings.TrimSpace(string(args)), want)
	}
}

func TestCheckFailureIsVisibleInSingBoxLogs(t *testing.T) {
	binary := fakeSingBoxBinary(t, filepath.Join(t.TempDir(), "args.log"), 2)
	service := &Service{
		Binary: binary,
		Proc:   &process.ManagedProcess{Name: "sing-box", Path: binary},
	}

	err := service.Check(context.Background(), "/tmp/config.json")
	if err == nil {
		t.Fatalf("expected check failure")
	}
	logs := strings.Join(service.Logs(), "\n")
	if !strings.Contains(logs, "sing-box config check failed") {
		t.Fatalf("expected check failure in logs, got %q", logs)
	}
}

func TestStartRequiresProxyInboundBeforeBinaryRuns(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	logPath := filepath.Join(t.TempDir(), "args.log")
	service := New(cfg, db, fakeSingBoxBinary(t, logPath, 0))

	err := service.Start(context.Background())
	if err == nil {
		t.Fatalf("expected missing inbound error")
	}
	if !strings.Contains(err.Error(), "sing-box proxy inbound is required before start") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(logPath); !os.IsNotExist(statErr) {
		t.Fatalf("sing-box binary should not be executed when proxy inbounds are missing")
	}
}

func TestNewServiceUsesSingBoxProcessName(t *testing.T) {
	cfg := testutil.NewConfig(t)
	svc := New(cfg, nil, cfg.Runtime.SingBoxBinary)
	if svc.Proc.Name != "sing-box" {
		t.Fatalf("unexpected process name: %q", svc.Proc.Name)
	}
	if svc.Proc.Path != cfg.Runtime.SingBoxBinary {
		t.Fatalf("unexpected process path: %q", svc.Proc.Path)
	}
}

func fakeSingBoxBinary(t *testing.T, logPath string, exitCode int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sing-box")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"exit " + string(rune('0'+exitCode)) + "\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake sing-box: %v", err)
	}
	return path
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
