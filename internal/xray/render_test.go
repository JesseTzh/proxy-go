package xray

import (
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestRenderVLESSXHTTPRealityInboundUsesLocalStreamBackend(t *testing.T) {
	out, err := Render(runtimeconfig.Snapshot{
		ProxyInbounds: []runtimeconfig.ProxyInbound{testInbound()},
	})
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
		`"listen": "127.0.0.1"`,
		`"port": 31001`,
		`"target": "apple.com:443"`,
		`"serverNames"`,
		`"apple.com"`,
		`"shortIds"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, `"mode": "auto"`) {
		t.Fatalf("server xhttp auto mode should be omitted so xray accepts all modes:\n%s", text)
	}
}

func TestRenderRejectsMultiplePublicXHTTPInbounds(t *testing.T) {
	_, err := Render(runtimeconfig.Snapshot{
		ProxyInbounds: []runtimeconfig.ProxyInbound{
			testInbound(),
			testInbound(),
		},
	})
	if err == nil {
		t.Fatalf("expected multiple public xhttp inbound error")
	}
	if !strings.Contains(err.Error(), "only one enabled vless-xhttp inbound") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderSingleInboundIncludesSecrets(t *testing.T) {
	rendered, err := RenderInbound(testInbound())
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

func testInbound() runtimeconfig.ProxyInbound {
	return runtimeconfig.ProxyInbound{
		ID:                     7,
		Name:                   "main",
		Template:               "vless-xhttp",
		Protocol:               "vless",
		Domain:                 "proxy.example.com",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		Network:                "xhttp",
		Security:               "reality",
		XHTTPPath:              "/xhttp",
		XHTTPMode:              "auto",
		RealityPrivateKey:      "private-key",
		RealityPublicKey:       "public-key",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60000,
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return testutil.NewConfig(t)
}
