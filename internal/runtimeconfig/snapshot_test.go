package runtimeconfig

import (
	"testing"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestLoadIncludesEnabledResourcesAndSystemSetting(t *testing.T) {
	db := testutil.NewDB(t)
	db.Model(&models.SystemSetting{}).Where("id=1").Update("management_domain", "admin.example.com")
	enabledDomain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	disabledDomain := models.Domain{Domain: "disabled.example.com", Status: "enabled"}
	db.Create(&enabledDomain)
	db.Create(&disabledDomain)
	db.Create(&models.ReverseProxyRule{
		DomainID:     enabledDomain.ID,
		TargetScheme: "http",
		TargetHost:   "127.0.0.1",
		TargetPort:   8080,
		Enabled:      true,
	})
	db.Create(&models.ReverseProxyRule{
		DomainID:     disabledDomain.ID,
		TargetScheme: "http",
		TargetHost:   "127.0.0.1",
		TargetPort:   8081,
		Enabled:      false,
	})
	db.Create(&models.ProxyInbound{
		DomainID:               enabledDomain.ID,
		Name:                   "main",
		Template:               "vless-reality-vision",
		Protocol:               "vless",
		UUID:                   "uuid",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "tcp",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RouteSNI:               "apple.com",
		RealityPrivateKey:      "private",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60000,
		Enabled:                true,
	})
	db.Create(&models.ProxyInbound{
		DomainID:   disabledDomain.ID,
		Name:       "disabled",
		Template:   "anytls",
		UUID:       "disabled-uuid",
		ListenPort: 31002,
		Enabled:    false,
	})

	snapshot, err := Load(db)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.ManagementDomain != "admin.example.com" {
		t.Fatalf("unexpected management domain: %q", snapshot.ManagementDomain)
	}
	if len(snapshot.ReverseProxies) != 1 {
		t.Fatalf("expected 1 reverse proxy, got %d", len(snapshot.ReverseProxies))
	}
	if snapshot.ReverseProxies[0].Domain != "proxy.example.com" {
		t.Fatalf("unexpected reverse proxy domain: %#v", snapshot.ReverseProxies[0])
	}
	if len(snapshot.ProxyInbounds) != 1 {
		t.Fatalf("expected 1 proxy inbound, got %d", len(snapshot.ProxyInbounds))
	}
	if snapshot.ProxyInbounds[0].Domain != "proxy.example.com" || snapshot.ProxyInbounds[0].ListenPort != 31001 {
		t.Fatalf("unexpected proxy inbound: %#v", snapshot.ProxyInbounds[0])
	}
	if got := snapshot.ProxyInbounds[0].Template; got != "vless-reality-vision" {
		t.Fatalf("template = %q", got)
	}
	if got := snapshot.ProxyInbounds[0].RouteSNI; got != "apple.com" {
		t.Fatalf("route sni = %q", got)
	}
	if got := snapshot.ProxyInbounds[0].RealityPrivateKey; got != "private" {
		t.Fatalf("private key = %q", got)
	}
}
