package app

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestStartDoesNotStartChildrenWhenHTTPListenFails(t *testing.T) {
	cfg := testutil.NewConfig(t)
	cfg.Runtime.StartChildren = true
	cfg.Server.StartInitialPort = false

	logPath := filepath.Join(t.TempDir(), "children.log")
	cfg.Runtime.NginxBinary = fakeRuntimeBinary(t, logPath)
	cfg.Runtime.SingBoxBinary = cfg.Runtime.NginxBinary

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	cfg.Server.InternalAddr = listener.Addr().String()

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("new application: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = application.Shutdown(ctx)
	}()

	err = application.Start(context.Background())
	if err == nil {
		t.Fatalf("expected start to fail when internal listener address is already in use")
	}

	data, readErr := os.ReadFile(logPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read child log: %v", readErr)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected child processes not to be started, got log:\n%s", data)
	}
}

func fakeRuntimeBinary(t *testing.T, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-runtime")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		"case \"$1\" in\n" +
		"  -t|check) exit 0 ;;\n" +
		"  run) while :; do sleep 1; done ;;\n" +
		"  *) while :; do sleep 1; done ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake runtime binary: %v", err)
	}
	return path
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
