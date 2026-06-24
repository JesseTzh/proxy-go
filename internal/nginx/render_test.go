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
					Domain:     "vless.example.com",
					ListenAddr: "127.0.0.1",
					ListenPort: 31001,
				},
				{
					Template:   "vless-xhttp",
					Domain:     "app.example.com",
					ListenAddr: "127.0.0.1",
					ListenPort: 31002,
					XHTTPPath:  "/xhttp",
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
		"app.example.com 127.0.0.1:30443;",
		"vless.example.com 127.0.0.1:31001;",
		"location ^~ /xhttp",
		"proxy_pass http://127.0.0.1:31002;",
		"proxy_pass http://127.0.0.1:30081;",
		"ssl_certificate /certs/app.example.com/fullchain.pem;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered config missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "app.example.com 127.0.0.1:31002;") {
		t.Fatalf("xhttp inbound should not be routed by stream SNI:\n%s", out)
	}
}
