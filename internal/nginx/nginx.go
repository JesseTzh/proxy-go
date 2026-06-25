package nginx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/process"
	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
	"gorm.io/gorm"
)

type Service struct {
	cfg    *config.Config
	db     *gorm.DB
	Binary string
	Proc   *process.ManagedProcess
}

var dockerNginxTempRoot = "/tmp/nginx"

func New(cfg *config.Config, db *gorm.DB, binary string) *Service {
	conf := filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf")
	return &Service{cfg: cfg, db: db, Binary: binary, Proc: &process.ManagedProcess{Name: "nginx", Path: binary, Args: []string{"-c", conf, "-g", "daemon off;"}, Dir: cfg.Paths.NginxConfDir}}
}

func (s *Service) GenerateConfig() (string, error) {
	snapshot, err := runtimeconfig.LoadWithConfig(s.db, s.cfg)
	if err != nil {
		return "", err
	}
	return Render(RenderInput{
		Snapshot:         snapshot,
		PidFile:          filepath.Join(s.cfg.Paths.NginxConfDir, "nginx.pid"),
		AccessLog:        filepath.Join(s.cfg.Paths.LogDir, "nginx-access.log"),
		ErrorLog:         filepath.Join(s.cfg.Paths.LogDir, "nginx-error.log"),
		HTTPPort:         s.cfg.Server.PublicHTTPPort,
		HTTPSPort:        s.cfg.Server.PublicHTTPSPort,
		GoInternalAddr:   s.cfg.Server.InternalAddr,
		ManagedHTTPSAddr: s.cfg.Server.ManagedHTTPSAddr,
		CertDir:          s.cfg.Paths.CertDir,
	})
}

func (s *Service) Apply(ctx context.Context) error {
	started := time.Now()
	slog.Info("nginx apply starting", "binary", s.Binary, "confDir", s.cfg.Paths.NginxConfDir)
	final, err := s.writeConfig(ctx)
	if err != nil {
		return err
	}
	if err := s.Reload(ctx); err != nil {
		return err
	}
	slog.Info("nginx apply completed", "config", final, "elapsed", time.Since(started).String())
	return nil
}

func (s *Service) Start(ctx context.Context) error {
	started := time.Now()
	slog.Info("nginx start starting", "binary", s.Binary, "confDir", s.cfg.Paths.NginxConfDir)
	final, err := s.writeConfig(ctx)
	if err != nil {
		return err
	}
	if err := s.Proc.Start(ctx); err != nil {
		return err
	}
	slog.Info("nginx start completed", "config", final, "elapsed", time.Since(started).String())
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	return s.Proc.Stop(ctx)
}

func (s *Service) Restart(ctx context.Context) error {
	started := time.Now()
	slog.Info("nginx restart starting", "binary", s.Binary, "confDir", s.cfg.Paths.NginxConfDir)
	final, err := s.writeConfig(ctx)
	if err != nil {
		return err
	}
	if err := s.Proc.Restart(ctx); err != nil {
		return err
	}
	slog.Info("nginx restart completed", "config", final, "elapsed", time.Since(started).String())
	return nil
}

func (s *Service) Status() any {
	return s.Proc.Status()
}

func (s *Service) writeConfig(ctx context.Context) (string, error) {
	conf, err := s.GenerateConfig()
	if err != nil {
		return "", err
	}
	if err := s.ensureRuntimeDirs(); err != nil {
		return "", err
	}
	tmp := filepath.Join(s.cfg.Paths.NginxConfDir, "nginx.conf.tmp")
	final := filepath.Join(s.cfg.Paths.NginxConfDir, "nginx.conf")
	if err := os.WriteFile(tmp, []byte(conf), 0644); err != nil {
		return "", err
	}
	slog.Info("nginx config rendered", "path", tmp, "bytes", len(conf))
	if err := s.Check(ctx, tmp); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, final); err != nil {
		return "", err
	}
	return final, nil
}

func (s *Service) ensureRuntimeDirs() error {
	for _, dir := range []string{
		s.cfg.Paths.NginxConfDir,
		s.cfg.Paths.LogDir,
		filepath.Join(s.cfg.Paths.NginxConfDir, "logs"),
		filepath.Join(dockerNginxTempRoot, "client_body"),
		filepath.Join(dockerNginxTempRoot, "proxy"),
		filepath.Join(dockerNginxTempRoot, "fastcgi"),
		filepath.Join(dockerNginxTempRoot, "uwsgi"),
		filepath.Join(dockerNginxTempRoot, "scgi"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Check(ctx context.Context, conf string) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	slog.Info("nginx config check starting", "binary", s.Binary, "config", conf)
	cmd := exec.CommandContext(cctx, s.Binary, "-t", "-c", conf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx config test failed: %w: %s", err, string(out))
	}
	slog.Info("nginx config check completed", "elapsed", time.Since(started).String())
	return nil
}

func (s *Service) Reload(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	configPath := filepath.Join(s.cfg.Paths.NginxConfDir, "nginx.conf")
	slog.Info("nginx reload starting", "binary", s.Binary, "config", configPath)
	cmd := exec.CommandContext(cctx, s.Binary, "-s", "reload", "-c", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx reload failed: %w: %s", err, string(out))
	}
	slog.Info("nginx reload completed", "elapsed", time.Since(started).String())
	return nil
}

func safeName(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

const nginxTemplate = `
worker_processes auto;
pid {{.PidFile}};
error_log {{.ErrorLog}} warn;

events { worker_connections 4096; }

http {
    include       mime.types;
    default_type  application/octet-stream;
    access_log {{.AccessLog}};

    map $http_upgrade $connection_upgrade {
        default upgrade;
        '' close;
    }

    server {
        listen {{.HTTPPort}} default_server;
        server_name _;
        location ^~ /.well-known/acme-challenge/ {
            proxy_pass http://{{.GoInternalAddr}};
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
        location / { return 301 https://$host$request_uri; }
    }

    server {
        listen {{.ManagedHTTPSAddr}} ssl;
        server_name {{if .ManagementDomain}}{{.ManagementDomain}}{{else}}_{{end}};
        ssl_certificate {{.CertDir}}/default/fullchain.pem;
        ssl_certificate_key {{.CertDir}}/default/privkey.pem;
        location / {
            proxy_pass http://{{.GoInternalAddr}};
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
        }
    }

    {{- range $rule := .Rules }}
    server {
        listen {{$.ManagedHTTPSAddr}} ssl;
        server_name {{ $rule.Domain }};
        ssl_certificate {{$.CertDir}}/{{ $rule.Domain }}/fullchain.pem;
        ssl_certificate_key {{$.CertDir}}/{{ $rule.Domain }}/privkey.pem;
        location / {
            proxy_pass {{ $rule.TargetScheme }}://{{ $rule.TargetHost }}:{{ $rule.TargetPort }};
            {{- if $rule.PreserveHost }}proxy_set_header Host $host;{{ else }}proxy_set_header Host {{ $rule.TargetHost }};{{ end }}
            {{- if $rule.PassRealIP }}
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto https;
            proxy_set_header X-Forwarded-Host $host;
            {{- end }}
            {{- if $rule.WebSocket }}
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection $connection_upgrade;
            {{- end }}
        }
    }
    {{- end }}
}
`
