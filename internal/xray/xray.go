package xray

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

var ErrConfigRequired = errors.New("xray config file is required before start")

func New(cfg *config.Config, db *gorm.DB, binary string) *Service {
	conf := filepath.Join(cfg.Paths.XrayConfDir, "config.json")
	return &Service{
		cfg:    cfg,
		db:     db,
		Binary: binary,
		Proc: &process.ManagedProcess{
			Name: "xray",
			Path: binary,
			Args: []string{"run", "-config", conf},
			Dir:  cfg.Paths.XrayConfDir,
		},
	}
}

func (s *Service) GenerateConfig() ([]byte, error) {
	snapshot, err := runtimeconfig.Load(s.db)
	if err != nil {
		return nil, err
	}
	return Render(snapshot)
}

func (s *Service) Apply(ctx context.Context) error {
	started := time.Now()
	slog.Info("xray apply starting", "binary", s.Binary, "confDir", s.cfg.Paths.XrayConfDir)
	b, err := s.GenerateConfig()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.cfg.Paths.XrayConfDir, 0755); err != nil {
		return err
	}
	tmp := filepath.Join(s.cfg.Paths.XrayConfDir, "config.json.tmp")
	final := filepath.Join(s.cfg.Paths.XrayConfDir, "config.json")
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	slog.Info("xray config rendered", "path", tmp, "bytes", len(b))
	if err := s.Check(ctx, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		return err
	}
	if err := s.Proc.Restart(ctx); err != nil {
		return err
	}
	slog.Info("xray apply completed", "config", final, "elapsed", time.Since(started).String())
	return nil
}

func (s *Service) Start(ctx context.Context) error {
	conf := filepath.Join(s.cfg.Paths.XrayConfDir, "config.json")
	if err := requireExistingConfig(conf); err != nil {
		if s.Proc != nil {
			s.Proc.AppendLog(err.Error() + "\n")
		}
		return err
	}
	if err := s.Check(ctx, conf); err != nil {
		return err
	}
	return s.Proc.Start(ctx)
}

func (s *Service) Stop(ctx context.Context) error {
	return s.Proc.Stop(ctx)
}

func (s *Service) Restart(ctx context.Context) error {
	conf := filepath.Join(s.cfg.Paths.XrayConfDir, "config.json")
	if err := requireExistingConfig(conf); err != nil {
		if s.Proc != nil {
			s.Proc.AppendLog(err.Error() + "\n")
		}
		return err
	}
	if err := s.Check(ctx, conf); err != nil {
		return err
	}
	return s.Proc.Restart(ctx)
}

func (s *Service) Status() any {
	return s.Proc.Status()
}

func (s *Service) Logs() []string {
	return s.Proc.Logs()
}

func (s *Service) Check(ctx context.Context, conf string) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	slog.Info("xray config check starting", "binary", s.Binary, "config", conf)
	cmd := exec.CommandContext(cctx, s.Binary, "run", "-test", "-config", conf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		checkErr := fmt.Errorf("xray config test failed: %w: %s", err, string(out))
		if s.Proc != nil {
			s.Proc.AppendLog(checkErr.Error() + "\n")
		}
		return checkErr
	}
	slog.Info("xray config check completed", "elapsed", time.Since(started).String())
	return nil
}

func requireExistingConfig(conf string) error {
	info, err := os.Stat(conf)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrConfigRequired, conf)
		}
		return fmt.Errorf("check xray config file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %s is a directory", ErrConfigRequired, conf)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%w: %s is empty", ErrConfigRequired, conf)
	}
	return nil
}
