package nginx

import (
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
)

func TestRenderMapsDomainsAndHTTPChallenge(t *testing.T) {
	out, err := Render(RenderInput{
		Snapshot: runtimeconfig.Snapshot{
			ManagementDomain: "admin.example.com",
			ReverseProxies: []runtimeconfig.ReverseProxy{{
				Domain:       "app.example.com",
				TargetScheme: "http",
				TargetHost:   "127.0.0.1",
				TargetPort:   8080,
				PreserveHost: true,
				PassRealIP:   true,
			}},
			ProxyInbounds: []runtimeconfig.ProxyInbound{
				{
					Template:               "vless-xhttp",
					Domain:                 "app.example.com",
					ListenAddr:             "127.0.0.1",
					ListenPort:             31002,
					XHTTPPath:              "/xhttp",
					RealityHandshakeServer: "apple.com",
				},
			},
		},
		PidFile:          "/run/nginx.pid",
		AccessLog:        "/log/access.log",
		ErrorLog:         "/log/error.log",
		HTTPPort:         80,
		HTTPSPort:        443,
		GoInternalAddr:   "127.0.0.1:30081",
		ManagedHTTPSAddr: "127.0.0.1:30443",
		CertDir:          "/certs",
	})
	if err != nil {
		t.Fatalf("render nginx: %v", err)
	}
	for _, want := range []string{
		"proxy_pass http://127.0.0.1:30081;",
		"ssl_certificate /certs/app.example.com/fullchain.pem;",
		"listen 127.0.0.1:30443 ssl;",
		"listen 443;",
		"ssl_preread on;",
		"apple.com 127.0.0.1:31002;",
		"default 127.0.0.1:30443;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "app.example.com 127.0.0.1:31002;") {
		t.Fatalf("xhttp inbound should be routed by reality handshake SNI only:\n%s", out)
	}
	if strings.Contains(out, "location ^~ /xhttp") || strings.Contains(out, "proxy_pass http://127.0.0.1:31002;") {
		t.Fatalf("nginx should not route xhttp when xray owns the public https entrypoint:\n%s", out)
	}
}
