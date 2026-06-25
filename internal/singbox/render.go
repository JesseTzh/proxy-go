package singbox

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
)

func Render(snapshot runtimeconfig.Snapshot) ([]byte, error) {
	inbounds := make([]any, 0, len(snapshot.ProxyInbounds))
	for _, inbound := range snapshot.ProxyInbounds {
		rendered, err := RenderInbound(inbound, snapshot.CertDir)
		if err != nil {
			return nil, err
		}
		inbounds = append(inbounds, rendered)
	}
	cfg := map[string]any{
		"log": map[string]any{
			"level": "info",
		},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{
			"rules": []any{},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func RenderInbound(in runtimeconfig.ProxyInbound, certDir string) (map[string]any, error) {
	switch in.Template {
	case "vless-reality-vision":
		return renderVLESSRealityVision(in), nil
	case "anytls":
		return renderAnyTLS(in, certDir), nil
	default:
		return nil, fmt.Errorf("unsupported inbound template %q", in.Template)
	}
}

func renderVLESSRealityVision(in runtimeconfig.ProxyInbound) map[string]any {
	return map[string]any{
		"type":        "vless",
		"tag":         inboundTag(in),
		"listen":      in.ListenAddr,
		"listen_port": in.ListenPort,
		"users": []any{
			map[string]any{
				"name": inboundUserName(in),
				"uuid": in.UUID,
				"flow": "xtls-rprx-vision",
			},
		},
		"tls": map[string]any{
			"enabled":     true,
			"server_name": in.RealityHandshakeServer,
			"reality": map[string]any{
				"enabled": true,
				"handshake": map[string]any{
					"server":      realityServerName(in),
					"server_port": realityHandshakePort(in),
				},
				"private_key":         in.RealityPrivateKey,
				"short_id":            []any{in.RealityShortID},
				"max_time_difference": realityMaxTimeDifference(in),
			},
		},
	}
}

func renderAnyTLS(in runtimeconfig.ProxyInbound, certDir string) map[string]any {
	domain := in.RouteSNI
	if domain == "" {
		domain = in.Domain
	}
	return map[string]any{
		"type":        "anytls",
		"tag":         inboundTag(in),
		"listen":      in.ListenAddr,
		"listen_port": in.ListenPort,
		"users": []any{
			map[string]any{
				"name":     inboundUserName(in),
				"password": in.Password,
			},
		},
		"tls": map[string]any{
			"enabled":          true,
			"server_name":      domain,
			"certificate_path": filepath.Join(certDir, domain, "fullchain.pem"),
			"key_path":         filepath.Join(certDir, domain, "privkey.pem"),
		},
	}
}

func inboundTag(in runtimeconfig.ProxyInbound) string {
	return fmt.Sprintf("inbound-%d", in.ID)
}

func inboundUserName(in runtimeconfig.ProxyInbound) string {
	if in.Name != "" {
		return in.Name
	}
	return inboundTag(in)
}

func realityServerName(in runtimeconfig.ProxyInbound) string {
	return in.RealityHandshakeServer
}

func realityHandshakePort(in runtimeconfig.ProxyInbound) int {
	if in.RealityHandshakePort != 0 {
		return in.RealityHandshakePort
	}
	return 443
}

func realityMaxTimeDifference(in runtimeconfig.ProxyInbound) string {
	if in.RealityMaxTimeDiff <= 0 {
		return ""
	}
	d := time.Duration(in.RealityMaxTimeDiff) * time.Millisecond
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", int(d/time.Second))
	}
	return d.String()
}
