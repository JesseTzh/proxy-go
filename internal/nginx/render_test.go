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
					Template:   "vless-reality-vision",
					Domain:     "proxy.example.com",
					ListenAddr: "127.0.0.1",
					ListenPort: 31001,
					RouteSNI:   "apple.com",
				},
				{
					Template:   "anytls",
					Domain:     "any.example.com",
					ListenAddr: "127.0.0.1",
					ListenPort: 31002,
					RouteSNI:   "any.example.com",
				},
			},
		},
		PidFile:           "/run/nginx.pid",
		AccessLog:         "/log/access.log",
		ErrorLog:          "/log/error.log",
		HTTPPort:          80,
		HTTPSPort:         443,
		GoInternalAddr:    "127.0.0.1:30081",
		ManagedHTTPSAddr:  "127.0.0.1:30443",
		CertDir:           "/certs",
		ClientMaxBodySize: "0",
		GzipEnabled:       true,
	})
	if err != nil {
		t.Fatalf("render nginx: %v", err)
	}
	for _, want := range []string{
		"proxy_pass http://127.0.0.1:30081;",
		"ssl_certificate /certs/app.example.com/fullchain.pem;",
		"listen 127.0.0.1:30443 ssl;",
		"listen 443;",
		"client_max_body_size 0;",
		"gzip on;",
		"gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml application/rss+xml image/svg+xml;",
		"ssl_preread on;",
		"apple.com 127.0.0.1:31001;",
		"any.example.com 127.0.0.1:31002;",
		"default 127.0.0.1:30443;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "proxy.example.com 127.0.0.1:31001;") {
		t.Fatalf("vision inbound should be routed by explicit route SNI only:\n%s", out)
	}
	if strings.Contains(out, "proxy_pass http://127.0.0.1:31002;") {
		t.Fatalf("nginx should only stream proxy protocol inbounds:\n%s", out)
	}
}
