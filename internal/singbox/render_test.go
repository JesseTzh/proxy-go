package singbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
)

func TestRenderVLESSRealityVisionAndAnyTLS(t *testing.T) {
	out, err := Render(runtimeconfig.Snapshot{
		CertDir: "/certs",
		ProxyInbounds: []runtimeconfig.ProxyInbound{
			{
				ID:                     1,
				Name:                   "vision",
				Template:               "vless-reality-vision",
				Protocol:               "vless",
				Domain:                 "proxy.example.com",
				ListenAddr:             "127.0.0.1",
				ListenPort:             31001,
				UUID:                   "11111111-1111-1111-1111-111111111111",
				Network:                "tcp",
				Security:               "reality",
				Flow:                   "xtls-rprx-vision",
				RouteSNI:               "apple.com",
				RealityPrivateKey:      "private-key",
				RealityPublicKey:       "public-key",
				RealityShortID:         "abcd1234",
				RealityHandshakeServer: "apple.com",
				RealityHandshakePort:   443,
				RealityMaxTimeDiff:     60000,
			},
			{
				ID:         2,
				Name:       "anytls",
				Template:   "anytls",
				Protocol:   "anytls",
				Domain:     "any.example.com",
				ListenAddr: "127.0.0.1",
				ListenPort: 31002,
				Network:    "tcp",
				Security:   "tls",
				RouteSNI:   "any.example.com",
				Password:   "secret-password",
			},
		},
	})
	if err != nil {
		t.Fatalf("render sing-box: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "streamSettings") {
		t.Fatalf("sing-box config must not contain legacy stream settings:\n%s", text)
	}

	var cfg map[string]any
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("decode config: %v\n%s", err, text)
	}
	inbounds := cfg["inbounds"].([]any)
	if len(inbounds) != 2 {
		t.Fatalf("expected 2 inbounds: %#v", inbounds)
	}

	vision := inbounds[0].(map[string]any)
	if vision["type"] != "vless" || vision["listen"] != "127.0.0.1" || int(vision["listen_port"].(float64)) != 31001 {
		t.Fatalf("unexpected vision inbound basics: %#v", vision)
	}
	visionUser := vision["users"].([]any)[0].(map[string]any)
	if visionUser["uuid"] != "11111111-1111-1111-1111-111111111111" || visionUser["flow"] != "xtls-rprx-vision" {
		t.Fatalf("unexpected vision user: %#v", visionUser)
	}
	visionTLS := vision["tls"].(map[string]any)
	reality := visionTLS["reality"].(map[string]any)
	handshake := reality["handshake"].(map[string]any)
	if visionTLS["enabled"] != true || reality["enabled"] != true || handshake["server"] != "apple.com" || int(handshake["server_port"].(float64)) != 443 {
		t.Fatalf("unexpected reality tls: %#v", visionTLS)
	}
	if reality["private_key"] != "private-key" || reality["max_time_difference"] != "1m" {
		t.Fatalf("unexpected reality credentials/timing: %#v", reality)
	}

	anytls := inbounds[1].(map[string]any)
	if anytls["type"] != "anytls" || anytls["listen"] != "127.0.0.1" || int(anytls["listen_port"].(float64)) != 31002 {
		t.Fatalf("unexpected anytls inbound basics: %#v", anytls)
	}
	anytlsUser := anytls["users"].([]any)[0].(map[string]any)
	if anytlsUser["password"] != "secret-password" {
		t.Fatalf("unexpected anytls user: %#v", anytlsUser)
	}
	anytlsTLS := anytls["tls"].(map[string]any)
	if anytlsTLS["enabled"] != true || anytlsTLS["server_name"] != "any.example.com" {
		t.Fatalf("unexpected anytls tls: %#v", anytlsTLS)
	}
	if anytlsTLS["certificate_path"] != "/certs/any.example.com/fullchain.pem" || anytlsTLS["key_path"] != "/certs/any.example.com/privkey.pem" {
		t.Fatalf("unexpected anytls certificate paths: %#v", anytlsTLS)
	}
}

func TestRenderRejectsUnsupportedInboundTemplate(t *testing.T) {
	_, err := Render(runtimeconfig.Snapshot{
		ProxyInbounds: []runtimeconfig.ProxyInbound{{Template: "legacy-template"}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported inbound template") {
		t.Fatalf("expected unsupported template error, got %v", err)
	}
}
