package app

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/config"
)

func TestSetupLoggerWritesStartupLogsToConsoleAndFile(t *testing.T) {
	t.Cleanup(func() {
		slog.SetDefault(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	})

	var console bytes.Buffer
	cfg := &config.Config{}
	cfg.Paths.LogDir = t.TempDir()

	setupLogger(cfg, &console)
	slog.Info("startup probe", "phase", "test")

	if !strings.Contains(console.String(), "startup probe") {
		t.Fatalf("expected startup log on console, got %q", console.String())
	}

	logPath := filepath.Join(cfg.Paths.LogDir, "proxy-go.log")
	data := readEventually(t, logPath)
	if !strings.Contains(string(data), "startup probe") {
		t.Fatalf("expected startup log in %s, got %q", logPath, string(data))
	}
}

func readEventually(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	return data
}
