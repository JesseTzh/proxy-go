package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDockerRuntimeBinaryPaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Runtime.NginxBinary != "/usr/local/bin/nginx" {
		t.Fatalf("unexpected nginx binary path: %q", cfg.Runtime.NginxBinary)
	}
	if cfg.Runtime.SingBoxBinary != "/usr/local/bin/sing-box" {
		t.Fatalf("unexpected sing-box binary path: %q", cfg.Runtime.SingBoxBinary)
	}
	if cfg.Paths.WebRoot != "/usr/share/proxy-go/web" {
		t.Fatalf("unexpected web root: %q", cfg.Paths.WebRoot)
	}
	if !cfg.Runtime.StartChildren {
		t.Fatalf("expected child processes to be enabled by default")
	}
}

func TestLoadKeepsRuntimeBinaryPathsHardcoded(t *testing.T) {
	t.Setenv("PROXY_GO_RUNTIME_NGINX_BINARY", "/tmp/env-nginx")
	t.Setenv("PROXY_GO_RUNTIME_SING_BOX_BINARY", "/tmp/env-sing-box")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(configPath, []byte(`
runtime:
  nginx_binary: "/tmp/config-nginx"
  sing_box_binary: "/tmp/config-sing-box"
`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Runtime.NginxBinary != DockerNginxBinary {
		t.Fatalf("unexpected nginx binary path: %q", cfg.Runtime.NginxBinary)
	}
	if cfg.Runtime.SingBoxBinary != DockerSingBoxBinary {
		t.Fatalf("unexpected sing-box binary path: %q", cfg.Runtime.SingBoxBinary)
	}
}
