package xray

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/process"
	"github.com/proxy-go/proxy-go/internal/testutil"
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

	args, err := os.ReadFile(logPath)
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

func TestStartRequiresEnabledProxyInbound(t *testing.T) {
	cfg := testConfig(t)
	cfg.Paths.XrayConfDir = t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	db := testutil.NewDB(t)
	service := New(cfg, db, fakeXrayBinary(t, logPath, 0))

	err := service.Start(context.Background())
	if err == nil {
		t.Fatalf("expected missing proxy inbound error")
	}
	if !strings.Contains(err.Error(), "xray proxy inbound is required before start") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, readErr := os.ReadFile(logPath); readErr == nil {
		t.Fatalf("xray binary should not be executed when proxy inbounds are missing")
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read args log: %v", readErr)
	}
}

func TestStartRendersAndStartsWhenEnabledProxyInboundExists(t *testing.T) {
	cfg := testConfig(t)
	cfg.Paths.XrayConfDir = t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	db := testutil.NewDB(t)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	if err := db.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		DomainID:               domain.ID,
		Name:                   "main",
		Template:               "vless-xhttp",
		Protocol:               "vless",
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "0.0.0.0",
		ListenPort:             443,
		Network:                "xhttp",
		Security:               "reality",
		XHTTPPath:              "/xhttp",
		XHTTPMode:              "auto",
		RealityPrivateKey:      "private-key",
		RealityPublicKey:       "public-key",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60,
		Enabled:                true,
	}).Error; err != nil {
		t.Fatalf("create proxy inbound: %v", err)
	}
	service := New(cfg, db, fakeLongRunningXrayBinary(t, logPath))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})
	conf := filepath.Join(cfg.Paths.XrayConfDir, "config.json")

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start xray: %v", err)
	}

	args, err := readFileUntil(t, logPath, "run -config "+conf)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	text := string(args)
	testConf, ok := findConfigArg(text, "run -test -config ")
	if !ok {
		t.Fatalf("expected config check command in args, got:\n%s", text)
	}
	if testConf == conf {
		t.Fatalf("expected config check to use a temporary config path, got final path %q", testConf)
	}
	if !strings.HasSuffix(testConf, ".json") {
		t.Fatalf("expected temporary config checked by xray to keep .json extension, got %q", testConf)
	}
	if !strings.Contains(text, "run -config "+conf) {
		t.Fatalf("expected start command in args, got:\n%s", text)
	}
}

func findConfigArg(text, prefix string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", false
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
