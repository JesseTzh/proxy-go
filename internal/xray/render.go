package xray

import (
	"encoding/json"
	"fmt"
	"path/filepath"

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
		rendered, err := RenderInboundWithDebug(inbound, snapshot.XrayDebugEnabled)
		if err != nil {
			return nil, err
		}
		inbounds = append(inbounds, rendered)
	}

	logConfig := map[string]any{"loglevel": "info"}
	if snapshot.XrayDebugEnabled {
		logConfig["loglevel"] = "debug"
		if snapshot.LogDir != "" {
			logConfig["access"] = filepath.Join(snapshot.LogDir, "xray-access.log")
			logConfig["error"] = filepath.Join(snapshot.LogDir, "xray-error.log")
		}
	}

	cfg := map[string]any{
		"log":      logConfig,
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
	return RenderInboundWithDebug(in, false)
}

func RenderInboundWithDebug(in runtimeconfig.ProxyInbound, debug bool) (map[string]any, error) {
	switch in.Template {
	case "vless-xhttp":
		return renderVLESSXHTTP(in, debug), nil
	default:
		return nil, fmt.Errorf("unsupported inbound template %q", in.Template)
	}
}

func renderVLESSXHTTP(in runtimeconfig.ProxyInbound, debug bool) map[string]any {
	stream := map[string]any{
		"network": "xhttp",
		"xhttpSettings": map[string]any{
			"path": in.XHTTPPath,
			"mode": in.XHTTPMode,
		},
		"security": "reality",
		"realitySettings": map[string]any{
			"show":        debug,
			"target":      realityTarget(in),
			"serverNames": []any{realityServerName(in)},
			"privateKey":  in.RealityPrivateKey,
			"shortIds":    []any{in.RealityShortID},
			"maxTimeDiff": in.RealityMaxTimeDiff,
		},
	}
	return baseVLESSInbound(in, stream)
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

func realityTarget(in runtimeconfig.ProxyInbound) string {
	return fmt.Sprintf("%s:%d", realityServerName(in), realityHandshakePort(in))
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
