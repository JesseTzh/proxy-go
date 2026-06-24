package xray

import (
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestRenderVLESSRealityVisionInbound(t *testing.T) {
	out, err := Render(runtimeconfig.Snapshot{
		ProxyInbounds: []runtimeconfig.ProxyInbound{testInbound("vless-reality-vision")},
	})
	if err != nil {
		t.Fatalf("render xray: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		`"protocol": "vless"`,
		`"network": "raw"`,
		`"security": "reality"`,
		`"flow": "xtls-rprx-vision"`,
		`"shortIds"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, text)
		}
	}
}

func TestRenderVLESSXHTTPRealityInbound(t *testing.T) {
	inbound := testInbound("vless-xhttp")
	inbound.Network = "xhttp"
	inbound.Flow = ""
	inbound.XHTTPPath = "/xhttp"
	inbound.XHTTPMode = "auto"

	out, err := Render(runtimeconfig.Snapshot{ProxyInbounds: []runtimeconfig.ProxyInbound{inbound}})
	if err != nil {
		t.Fatalf("render xray: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		`"protocol": "vless"`,
		`"network": "xhttp"`,
		`"xhttpSettings"`,
		`"path": "/xhttp"`,
		`"security": "reality"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, text)
		}
	}
}

func TestRenderSingleInboundIncludesSecrets(t *testing.T) {
	rendered, err := RenderInbound(testInbound("vless-reality-vision"))
	if err != nil {
		t.Fatalf("render inbound: %v", err)
	}
	data := rendered["streamSettings"].(map[string]any)["realitySettings"].(map[string]any)
	if data["privateKey"] != "private-key" {
		t.Fatalf("private key = %#v", data["privateKey"])
	}
	shortIDs := data["shortIds"].([]any)
	if shortIDs[0] != "abcd1234" {
		t.Fatalf("short id = %#v", shortIDs[0])
	}
}

func TestNewServiceUsesXrayProcessName(t *testing.T) {
	cfg := testConfig(t)
	svc := New(cfg, nil, cfg.Runtime.XrayBinary)
	if svc.Proc.Name != "xray" {
		t.Fatalf("process name = %q", svc.Proc.Name)
	}
	if svc.Proc.Path != cfg.Runtime.XrayBinary {
		t.Fatalf("process path = %q", svc.Proc.Path)
	}
}

func testInbound(template string) runtimeconfig.ProxyInbound {
	return runtimeconfig.ProxyInbound{
		ID:                     7,
		Name:                   "main",
		Template:               template,
		Protocol:               "vless",
		Domain:                 "proxy.example.com",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		Network:                "raw",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RealityPrivateKey:      "private-key",
		RealityPublicKey:       "public-key",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "www.cloudflare.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60,
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return testutil.NewConfig(t)
}
