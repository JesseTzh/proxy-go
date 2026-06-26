package nginx

import (
	"bytes"
	"text/template"

	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
)

type RenderInput struct {
	Snapshot          runtimeconfig.Snapshot
	PidFile           string
	AccessLog         string
	ErrorLog          string
	HTTPPort          int
	HTTPSPort         int
	GoInternalAddr    string
	ManagedHTTPSAddr  string
	CertDir           string
	ClientMaxBodySize string
	GzipEnabled       bool
}

func Render(input RenderInput) (string, error) {
	data := map[string]any{
		"PidFile":           input.PidFile,
		"AccessLog":         input.AccessLog,
		"ErrorLog":          input.ErrorLog,
		"HTTPPort":          input.HTTPPort,
		"HTTPSPort":         input.HTTPSPort,
		"GoInternalAddr":    input.GoInternalAddr,
		"ManagedHTTPSAddr":  input.ManagedHTTPSAddr,
		"ClientMaxBodySize": input.ClientMaxBodySize,
		"GzipEnabled":       input.GzipEnabled,
		"ManagementDomain":  input.Snapshot.ManagementDomain,
		"CertDir":           input.CertDir,
		"Rules":             input.Snapshot.ReverseProxies,
		"Inbounds":          input.Snapshot.ProxyInbounds,
	}
	tpl, err := template.New("nginx").Funcs(template.FuncMap{"safeName": safeName}).Parse(nginxTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
