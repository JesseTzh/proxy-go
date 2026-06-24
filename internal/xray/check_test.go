package xray

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/proxy-go/proxy-go/internal/process"
)

func TestCheckUsesCurrentXrayConfigTestCommand(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "args.log")
	binary := fakeXrayBinary(t, logPath, 0)
	conf := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(conf, []byte(`{"inbounds":[],"outbounds":[]}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	service := &Service{Binary: binary}
	if err := service.Check(context.Background(), conf); err != nil {
		t.Fatalf("check xray config: %v", err)
	}

	args, err := readFileUntil(t, logPath, "run -config "+conf)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	want := "run -test -config " + conf
	if strings.TrimSpace(string(args)) != want {
		t.Fatalf("xray args = %q, want %q", strings.TrimSpace(string(args)), want)
	}
}

func TestCheckFailureIsVisibleInXrayLogs(t *testing.T) {
	binary := fakeXrayBinary(t, filepath.Join(t.TempDir(), "args.log"), 2)
	conf := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(conf, []byte(`{"inbounds":[],"outbounds":[]}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	service := &Service{
		Binary: binary,
		Proc:   &process.ManagedProcess{Name: "xray", Path: binary},
	}
	err := service.Check(context.Background(), conf)
	if err == nil {
		t.Fatalf("expected check failure")
	}

	logs := strings.Join(service.Logs(), "\n")
	if !strings.Contains(logs, "xray config test failed") {
		t.Fatalf("expected config test error in logs, got:\n%s", logs)
	}
	if !strings.Contains(logs, "simulated config error") {
		t.Fatalf("expected command output in logs, got:\n%s", logs)
	}
}

func TestStartRequiresExistingConfigFile(t *testing.T) {
	cfg := testConfig(t)
	cfg.Paths.XrayConfDir = t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	service := New(cfg, nil, fakeXrayBinary(t, logPath, 0))

	err := service.Start(context.Background())
	if err == nil {
		t.Fatalf("expected missing config error")
	}
	if !strings.Contains(err.Error(), "xray config file is required before start") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, readErr := os.ReadFile(logPath); readErr == nil {
		t.Fatalf("xray binary should not be executed when config is missing")
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read args log: %v", readErr)
	}
}

func TestStartUsesExistingConfigWithoutRendering(t *testing.T) {
	cfg := testConfig(t)
	cfg.Paths.XrayConfDir = t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	service := New(cfg, nil, fakeLongRunningXrayBinary(t, logPath))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})
	conf := filepath.Join(cfg.Paths.XrayConfDir, "config.json")
	if err := os.WriteFile(conf, []byte(`{"inbounds":[{}],"outbounds":[{}]}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start xray: %v", err)
	}

	args, err := readFileUntil(t, logPath, "run -config "+conf)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	text := string(args)
	if !strings.Contains(text, "run -test -config "+conf) {
		t.Fatalf("expected config check command in args, got:\n%s", text)
	}
	if !strings.Contains(text, "run -config "+conf) {
		t.Fatalf("expected start command in args, got:\n%s", text)
	}
}

func fakeXrayBinary(t *testing.T, logPath string, exitCode int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xray")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"if [ " + fmt.Sprint(exitCode) + " -ne 0 ]; then\n" +
		"  echo simulated config error >&2\n" +
		"  exit " + fmt.Sprint(exitCode) + "\n" +
		"fi\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake xray: %v", err)
	}
	return path
}

func fakeLongRunningXrayBinary(t *testing.T, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xray")
	script := "#!/bin/sh\n" +
		"trap 'exit 0' INT TERM\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"if [ \"$2\" = \"-test\" ]; then exit 0; fi\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake xray: %v", err)
	}
	return path
}

func readFileUntil(t *testing.T, path string, needle string) ([]byte, error) {
	t.Helper()
	var last []byte
	var lastErr error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		last, lastErr = os.ReadFile(path)
		if lastErr == nil && strings.Contains(string(last), needle) {
			return last, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return last, lastErr
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
