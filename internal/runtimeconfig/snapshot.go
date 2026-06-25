package runtimeconfig

import (
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

func Load(db *gorm.DB) (Snapshot, error) {
	return LoadWithConfig(db, nil)
}

func LoadWithConfig(db *gorm.DB, cfg *config.Config) (Snapshot, error) {
	var snapshot Snapshot
	if cfg != nil {
		snapshot.PublicHTTPSPort = cfg.Server.PublicHTTPSPort
		snapshot.ManagedHTTPSAddr = cfg.Server.ManagedHTTPSAddr
		snapshot.CertDir = cfg.Paths.CertDir
		snapshot.LogDir = cfg.Paths.LogDir
	}
	var setting models.SystemSetting
	if err := db.First(&setting, 1).Error; err != nil && err != gorm.ErrRecordNotFound {
		return snapshot, err
	}
	snapshot.ManagementDomain = setting.ManagementDomain
	snapshot.SingBoxDebugEnabled = setting.SingBoxDebugEnabled

	var rules []models.ReverseProxyRule
	if err := db.Preload("Domain").Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return snapshot, err
	}
	snapshot.ReverseProxies = make([]ReverseProxy, 0, len(rules))
	for _, rule := range rules {
		snapshot.ReverseProxies = append(snapshot.ReverseProxies, ReverseProxy{
			Domain:       rule.Domain.Domain,
			TargetScheme: rule.TargetScheme,
			TargetHost:   rule.TargetHost,
			TargetPort:   rule.TargetPort,
			PreserveHost: rule.PreserveHost,
			WebSocket:    rule.WebSocket,
			PassRealIP:   rule.PassRealIP,
		})
	}

	var inbounds []models.ProxyInbound
	if err := db.Preload("Domain").Where("enabled = ?", true).Find(&inbounds).Error; err != nil {
		return snapshot, err
	}
	snapshot.ProxyInbounds = make([]ProxyInbound, 0, len(inbounds))
	for _, inbound := range inbounds {
		snapshot.ProxyInbounds = append(snapshot.ProxyInbounds, ProxyInbound{
			ID:                     inbound.ID,
			Name:                   inbound.Name,
			Template:               inbound.Template,
			Protocol:               inbound.Protocol,
			Domain:                 inbound.Domain.Domain,
			ListenAddr:             inbound.ListenAddr,
			ListenPort:             inbound.ListenPort,
			UUID:                   inbound.UUID,
			Network:                inbound.Network,
			Security:               inbound.Security,
			Flow:                   inbound.Flow,
			RouteSNI:               inbound.RouteSNI,
			Password:               inbound.Password,
			RealityPrivateKey:      inbound.RealityPrivateKey,
			RealityPublicKey:       inbound.RealityPublicKey,
			RealityShortID:         inbound.RealityShortID,
			RealityHandshakeServer: inbound.RealityHandshakeServer,
			RealityHandshakePort:   inbound.RealityHandshakePort,
			RealityMaxTimeDiff:     inbound.RealityMaxTimeDiff,
		})
	}

	return snapshot, nil
}
