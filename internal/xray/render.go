package xray

import (
	"encoding/json"
	"fmt"

	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
)

func Render(snapshot runtimeconfig.Snapshot) ([]byte, error) {
	inbounds := make([]any, 0, len(snapshot.ProxyInbounds))
	publicXHTTPCount := 0
	for _, inbound := range snapshot.ProxyInbounds {
		if inbound.Template == "vless-xhttp" {
			publicXHTTPCount++
		}
		if publicXHTTPCount > 1 {
			return nil, fmt.Errorf("only one enabled vless-xhttp inbound can listen on public https port")
		}
		rendered, err := RenderInbound(inbound)
		if err != nil {
			return nil, err
		}
		inbounds = append(inbounds, rendered)
	}

	cfg := map[string]any{
		"log":      map[string]any{"loglevel": "info"},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
		},
		"routing": map[string]any{
			"domainStrategy": "AsIs",
			"rules":          []any{},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func RenderInbound(in runtimeconfig.ProxyInbound) (map[string]any, error) {
	switch in.Template {
	case "vless-xhttp":
		return renderVLESSXHTTP(in), nil
	default:
		return nil, fmt.Errorf("unsupported inbound template %q", in.Template)
	}
}

func renderVLESSXHTTP(in runtimeconfig.ProxyInbound) map[string]any {
	stream := map[string]any{
		"network": "xhttp",
		"xhttpSettings": map[string]any{
			"path": in.XHTTPPath,
			"mode": in.XHTTPMode,
		},
		"security": "reality",
		"realitySettings": map[string]any{
			"show":        false,
			"dest":        realityDest(in),
			"serverNames": []any{realityServerName(in)},
			"privateKey":  in.RealityPrivateKey,
			"shortIds":    []any{in.RealityShortID},
			"maxTimeDiff": in.RealityMaxTimeDiff,
		},
	}
	return baseVLESSInbound(publicInbound(in), stream)
}

func baseVLESSInbound(in runtimeconfig.ProxyInbound, streamSettings map[string]any) map[string]any {
	client := map[string]any{
		"id":    in.UUID,
		"email": fmt.Sprintf("inbound-%d@proxy-go.local", in.ID),
	}
	if in.Flow != "" {
		client["flow"] = in.Flow
	}
	return map[string]any{
		"tag":      fmt.Sprintf("inbound-%d", in.ID),
		"listen":   in.ListenAddr,
		"port":     in.ListenPort,
		"protocol": "vless",
		"settings": map[string]any{
			"clients":    []any{client},
			"decryption": "none",
		},
		"streamSettings": streamSettings,
	}
}

func publicInbound(in runtimeconfig.ProxyInbound) runtimeconfig.ProxyInbound {
	in.ListenAddr = "0.0.0.0"
	if in.PublicHTTPSPort != 0 {
		in.ListenPort = in.PublicHTTPSPort
	}
	return in
}

func realityDest(in runtimeconfig.ProxyInbound) string {
	if in.ManagedHTTPSAddr != "" {
		return in.ManagedHTTPSAddr
	}
	return fmt.Sprintf("%s:%d", realityServerName(in), realityHandshakePort(in))
}

func realityServerName(in runtimeconfig.ProxyInbound) string {
	if in.RealityHandshakeServer != "" {
		return in.RealityHandshakeServer
	}
	return in.Domain
}

func realityHandshakePort(in runtimeconfig.ProxyInbound) int {
	if in.RealityHandshakePort != 0 {
		return in.RealityHandshakePort
	}
	return 443
}
