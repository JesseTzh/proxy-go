package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type NginxApplier interface {
	Apply(context.Context) error
	Reload(context.Context) error
	Start(context.Context) error
	Stop(context.Context) error
	Restart(context.Context) error
	Status() any
}

type SingBoxApplier interface {
	Apply(context.Context) error
	Start(context.Context) error
	Stop(context.Context) error
	Restart(context.Context) error
	Status() any
	Logs() []string
}

type Service struct {
	DB      *gorm.DB
	Cfg     *config.Config
	Nginx   NginxApplier
	SingBox SingBoxApplier
}

type Status struct {
	ProxyGo                  any        `json:"proxyGo"`
	Nginx                    any        `json:"nginx"`
	SingBox                  any        `json:"singBox"`
	GoInternalAddr           string     `json:"goInternalAddr"`
	NginxPublicHTTPPort      int        `json:"nginxPublicHttpPort"`
	NginxPublicHTTPSPort     int        `json:"nginxPublicHttpsPort"`
	NginxManagedHTTPSAddr    string     `json:"nginxManagedHttpsAddr"`
	SingBoxInboundListen     string     `json:"singBoxInboundListen"`
	SingBoxDebugEnabled      bool       `json:"singBoxDebugEnabled"`
	DomainCount              int64      `json:"domainCount"`
	CertificateCount         int64      `json:"certificateCount"`
	ReverseProxyCount        int64      `json:"reverseProxyCount"`
	InboundCount             int64      `json:"inboundCount"`
	ExpiringCertificateCount int64      `json:"expiringCertificateCount"`
	LastNginxReloadAt        *time.Time `json:"lastNginxReloadAt"`
	LastSingBoxRestartAt     *time.Time `json:"lastSingBoxRestartAt"`
	LastCertificateRenewalAt *time.Time `json:"lastCertificateRenewalAt"`
}

type LogSummary struct {
	Logs []string `json:"logs"`
}

type ConfigSnapshot struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s *Service) Status() (Status, error) {
	var status Status
	status.ProxyGo = map[string]any{"running": true}
	status.Nginx = s.Nginx.Status()
	status.SingBox = s.SingBox.Status()
	status.GoInternalAddr = s.Cfg.Server.InternalAddr
	status.NginxPublicHTTPPort = s.Cfg.Server.PublicHTTPPort
	status.NginxPublicHTTPSPort = s.Cfg.Server.PublicHTTPSPort
	status.NginxManagedHTTPSAddr = s.Cfg.Server.ManagedHTTPSAddr
	if err := s.DB.Model(&models.Domain{}).Where("status=?", "enabled").Count(&status.DomainCount).Error; err != nil {
		return status, err
	}
	if err := s.DB.Model(&models.Certificate{}).Count(&status.CertificateCount).Error; err != nil {
		return status, err
	}
	if err := s.DB.Model(&models.ReverseProxyRule{}).Where("enabled=?", true).Count(&status.ReverseProxyCount).Error; err != nil {
		return status, err
	}
	if err := s.DB.Model(&models.ProxyInbound{}).Where("enabled=?", true).Count(&status.InboundCount).Error; err != nil {
		return status, err
	}
	var inbound models.ProxyInbound
	if err := s.DB.Where("enabled = ?", true).Order("id asc").First(&inbound).Error; err == nil {
		status.SingBoxInboundListen = fmt.Sprintf("%s:%d", inbound.ListenAddr, inbound.ListenPort)
	}
	if err := s.DB.Model(&models.Certificate{}).Where("expires_at <= ?", time.Now().Add(30*24*time.Hour)).Count(&status.ExpiringCertificateCount).Error; err != nil {
		return status, err
	}
	var setting models.SystemSetting
	if err := s.DB.First(&setting, 1).Error; err != nil {
		return status, err
	}
	status.LastNginxReloadAt = setting.LastNginxReloadAt
	status.LastSingBoxRestartAt = setting.LastSingBoxRestartAt
	status.LastCertificateRenewalAt = setting.LastCertificateRenewalAt
	status.SingBoxDebugEnabled = setting.SingBoxDebugEnabled
	return status, nil
}

func (s *Service) Apply(ctx context.Context) error {
	if err := s.SingBox.Apply(ctx); err != nil {
		s.saveApply("failed")
		return err
	}
	if err := s.Nginx.Apply(ctx); err != nil {
		s.saveApply("failed")
		return err
	}
	s.saveApply("success")
	return nil
}

func (s *Service) ReloadNginx(ctx context.Context) error {
	if err := s.Nginx.Reload(ctx); err != nil {
		return err
	}
	now := time.Now()
	return s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_nginx_reload_at", &now).Error
}

func (s *Service) StartNginx(ctx context.Context) error {
	return s.Nginx.Start(ctx)
}

func (s *Service) StopNginx(ctx context.Context) error {
	return s.Nginx.Stop(ctx)
}

func (s *Service) RestartNginx(ctx context.Context) error {
	if err := s.Nginx.Restart(ctx); err != nil {
		return err
	}
	now := time.Now()
	return s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_nginx_reload_at", &now).Error
}

func (s *Service) StartSingBox(ctx context.Context) error {
	return s.SingBox.Start(ctx)
}

func (s *Service) StopSingBox(ctx context.Context) error {
	return s.SingBox.Stop(ctx)
}

func (s *Service) RestartSingBox(ctx context.Context) error {
	if err := s.SingBox.Restart(ctx); err != nil {
		return err
	}
	now := time.Now()
	return s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_sing_box_restart_at", &now).Error
}

func (s *Service) SetSingBoxDebug(ctx context.Context, enabled bool) error {
	if err := s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("sing_box_debug_enabled", enabled).Error; err != nil {
		return err
	}
	if err := s.SingBox.Apply(ctx); err != nil {
		return err
	}
	now := time.Now()
	return s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_sing_box_restart_at", &now).Error
}

func (s *Service) Logs() LogSummary {
	return LogSummary{Logs: []string{"logs are written under " + filepath.Clean(s.Cfg.Paths.LogDir)}}
}

func (s *Service) NginxConfig() (ConfigSnapshot, error) {
	path := filepath.Join(s.Cfg.Paths.NginxConfDir, "nginx.conf")
	content, err := os.ReadFile(path)
	if err != nil {
		return ConfigSnapshot{Path: path}, err
	}
	return ConfigSnapshot{Path: path, Content: string(content)}, nil
}

func (s *Service) SingBoxLogs() LogSummary {
	logs := append([]string{}, s.SingBox.Logs()...)
	logs = appendLogFile(logs, "sing-box-error.log", filepath.Join(s.Cfg.Paths.LogDir, "sing-box-error.log"))
	logs = appendLogFile(logs, "sing-box-access.log", filepath.Join(s.Cfg.Paths.LogDir, "sing-box-access.log"))
	return LogSummary{Logs: logs}
}

func appendLogFile(logs []string, name, path string) []string {
	lines, err := tailLogFile(path, 64*1024)
	if err != nil {
		if os.IsNotExist(err) {
			return logs
		}
		return append(logs, "==== "+name+" read failed ====", err.Error())
	}
	if len(lines) == 0 {
		return logs
	}
	logs = append(logs, "==== "+name+" ====")
	return append(logs, lines...)
}

func tailLogFile(path string, limit int64) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		content = content[int64(len(content))-limit:]
	}
	text := strings.TrimRight(string(content), "\n")
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}

func (s *Service) saveApply(status string) {
	_ = s.DB.Model(&models.SystemSetting{}).Where("id=1").Updates(map[string]any{
		"runtime_config_status": status,
	}).Error
}

func (s Status) String() string {
	return fmt.Sprintf("domains=%d reverseProxies=%d inbounds=%d", s.DomainCount, s.ReverseProxyCount, s.InboundCount)
}
