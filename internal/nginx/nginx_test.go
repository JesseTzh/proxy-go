package nginx

import (
	"os"
	"path/filepath"
	"testing"

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
