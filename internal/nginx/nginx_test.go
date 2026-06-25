package nginx

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestEnsureRuntimeDirsCreatesNginxDefaults(t *testing.T) {
	cfg := testutil.NewConfig(t)
	cfg.Paths.LogDir = filepath.Join(cfg.Paths.DataDir, "logs")
	tempRoot := filepath.Join(cfg.Paths.DataDir, "tmp", "nginx")
	oldTempRoot := dockerNginxTempRoot
	dockerNginxTempRoot = tempRoot
	t.Cleanup(func() { dockerNginxTempRoot = oldTempRoot })

	svc := New(cfg, nil, cfg.Runtime.NginxBinary)
	if err := svc.ensureRuntimeDirs(); err != nil {
		t.Fatalf("ensure runtime dirs: %v", err)
	}

	for _, dir := range []string{
		cfg.Paths.NginxConfDir,
		cfg.Paths.LogDir,
		filepath.Join(cfg.Paths.NginxConfDir, "logs"),
		filepath.Join(tempRoot, "client_body"),
		filepath.Join(tempRoot, "proxy"),
		filepath.Join(tempRoot, "fastcgi"),
		filepath.Join(tempRoot, "uwsgi"),
		filepath.Join(tempRoot, "scgi"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func TestReloadStartsManagedProcessWhenPidFileIsEmpty(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	if err := os.MkdirAll(cfg.Paths.NginxConfDir, 0755); err != nil {
		t.Fatalf("create nginx conf dir: %v", err)
	}
	confPath := filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf")
	if err := os.WriteFile(confPath, []byte("events {}\n"), 0644); err != nil {
		t.Fatalf("write nginx conf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Paths.NginxConfDir, "nginx.pid"), nil, 0644); err != nil {
		t.Fatalf("write empty pid: %v", err)
	}
	binary, logPath := fakeReloadNginxBinary(t)
	svc := New(cfg, db, binary)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("reload should start managed process after empty pid failure: %v", err)
	}
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	log := waitForLogContains(t, logPath, "daemon off;")
	if !strings.Contains(log, "-s reload") {
		t.Fatalf("expected reload attempt, log:\n%s", log)
	}
}

func fakeReloadNginxBinary(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "nginx")
	logPath := filepath.Join(dir, "nginx-args.log")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(logPath) + "\n" +
		"case \"$*\" in\n" +
		"  *\"-s reload\"*) printf 'nginx: [error] invalid PID number \"\" in \"/var/lib/proxy-go/nginx/nginx.pid\"\\n' >&2; exit 1 ;;\n" +
		"  *\"-t\"*) exit 0 ;;\n" +
		"esac\n" +
		"trap 'exit 0' INT TERM\n" +
		"while :; do sleep 1; done\n"
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write fake nginx: %v", err)
	}
	return binary, logPath
}

func waitForLogContains(t *testing.T, logPath, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var log string
	for time.Now().Before(deadline) {
		logBytes, err := os.ReadFile(logPath)
		if err == nil {
			log = string(logBytes)
			if strings.Contains(log, want) {
				return log
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected log to contain %q, log:\n%s", want, log)
	return log
}
