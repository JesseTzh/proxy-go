package runtime

import (
	"context"
	"fmt"
	"path/filepath"
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

type XrayApplier interface {
	Apply(context.Context) error
	Start(context.Context) error
	Stop(context.Context) error
	Restart(context.Context) error
	Status() any
	Logs() []string
}

type Service struct {
	DB    *gorm.DB
	Cfg   *config.Config
	Nginx NginxApplier
	Xray  XrayApplier
}

type Status struct {
	ProxyGo                  any        `json:"proxyGo"`
	Nginx                    any        `json:"nginx"`
	Xray                     any        `json:"xray"`
	GoInternalAddr           string     `json:"goInternalAddr"`
	NginxPorts               []int      `json:"nginxPorts"`
	DomainCount              int64      `json:"domainCount"`
	CertificateCount         int64      `json:"certificateCount"`
	ReverseProxyCount        int64      `json:"reverseProxyCount"`
	InboundCount             int64      `json:"inboundCount"`
	ExpiringCertificateCount int64      `json:"expiringCertificateCount"`
	LastNginxReloadAt        *time.Time `json:"lastNginxReloadAt"`
	LastXrayRestartAt        *time.Time `json:"lastXrayRestartAt"`
	LastCertificateRenewalAt *time.Time `json:"lastCertificateRenewalAt"`
}

type LogSummary struct {
	Logs []string `json:"logs"`
}

func (s *Service) Status() (Status, error) {
	var status Status
	status.ProxyGo = map[string]any{"running": true}
	status.Nginx = s.Nginx.Status()
	status.Xray = s.Xray.Status()
	status.GoInternalAddr = s.Cfg.Server.InternalAddr
	status.NginxPorts = []int{s.Cfg.Server.PublicHTTPPort, s.Cfg.Server.PublicHTTPSPort}
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
	if err := s.DB.Model(&models.Certificate{}).Where("expires_at <= ?", time.Now().Add(30*24*time.Hour)).Count(&status.ExpiringCertificateCount).Error; err != nil {
		return status, err
	}
	var setting models.SystemSetting
	if err := s.DB.First(&setting, 1).Error; err != nil {
		return status, err
	}
	status.LastNginxReloadAt = setting.LastNginxReloadAt
	status.LastXrayRestartAt = setting.LastXrayRestartAt
	status.LastCertificateRenewalAt = setting.LastCertificateRenewalAt
	return status, nil
}

func (s *Service) Apply(ctx context.Context) error {
	if err := s.Nginx.Apply(ctx); err != nil {
		s.saveApply("failed")
		return err
	}
	if err := s.Xray.Apply(ctx); err != nil {
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

func (s *Service) StartXray(ctx context.Context) error {
	return s.Xray.Start(ctx)
}

func (s *Service) StopXray(ctx context.Context) error {
	return s.Xray.Stop(ctx)
}

func (s *Service) RestartXray(ctx context.Context) error {
	if err := s.Xray.Restart(ctx); err != nil {
		return err
	}
	now := time.Now()
	return s.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_xray_restart_at", &now).Error
}

func (s *Service) Logs() LogSummary {
	return LogSummary{Logs: []string{"logs are written under " + filepath.Clean(s.Cfg.Paths.LogDir)}}
}

func (s *Service) XrayLogs() LogSummary {
	return LogSummary{Logs: s.Xray.Logs()}
}

func (s *Service) saveApply(status string) {
	_ = s.DB.Model(&models.SystemSetting{}).Where("id=1").Updates(map[string]any{
		"runtime_config_status": status,
	}).Error
}

func (s Status) String() string {
	return fmt.Sprintf("domains=%d reverseProxies=%d inbounds=%d", s.DomainCount, s.ReverseProxyCount, s.InboundCount)
}
