package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Paths    PathsConfig    `koanf:"paths"`
	Security SecurityConfig `koanf:"security"`
	ACME     ACMEConfig     `koanf:"acme"`
	Runtime  RuntimeConfig  `koanf:"runtime"`
}

type ServerConfig struct {
	InitialAddr      string `koanf:"initial_addr"`
	InternalAddr     string `koanf:"internal_addr"`
	PublicHTTPSPort  int    `koanf:"public_https_port"`
	PublicHTTPPort   int    `koanf:"public_http_port"`
	ManagedHTTPSAddr string `koanf:"managed_https_addr"`
	StartInitialPort bool   `koanf:"start_initial_port"`
	CookieSecure     bool   `koanf:"cookie_secure"`
}

type PathsConfig struct {
	DataDir      string `koanf:"data_dir"`
	LogDir       string `koanf:"log_dir"`
	DBFile       string `koanf:"db_file"`
	CertDir      string `koanf:"cert_dir"`
	BinDir       string `koanf:"bin_dir"`
	NginxConfDir string `koanf:"nginx_conf_dir"`
	XrayConfDir  string `koanf:"xray_conf_dir"`
	WebRoot      string `koanf:"web_root"`
}

type SecurityConfig struct {
	InitialPassword string `koanf:"initial_password"`
	SessionTTLHours int    `koanf:"session_ttl_hours"`
	BcryptCost      int    `koanf:"bcrypt_cost"`
}

type ACMEConfig struct {
	Email           string `koanf:"email"`
	DirectoryURL    string `koanf:"directory_url"`
	RenewBeforeDays int    `koanf:"renew_before_days"`
}

type RuntimeConfig struct {
	StartChildren bool   `koanf:"start_children"`
	NginxBinary   string `koanf:"nginx_binary"`
	XrayBinary    string `koanf:"xray_binary"`
}

const (
	DockerNginxBinary = "/usr/local/bin/nginx"
	DockerXrayBinary  = "/usr/local/bin/xray"
	DockerWebRoot     = "/usr/share/proxy-go/web"
)

func Load(path string) (*Config, error) {
	k := koanf.New(".")
	setDefaults(k)
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
				return nil, err
			}
		}
	}
	if err := k.Load(env.Provider("PROXY_GO_", ".", func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(s, "PROXY_GO_"), "_", "."))
	}), nil); err != nil {
		return nil, err
	}
	// Backward-compatible short environment variables required by the product spec.
	if v := os.Getenv("PROXY_GO_INITIAL_PASSWORD"); v != "" {
		_ = k.Set("security.initial_password", v)
	}
	if v := os.Getenv("PROXY_GO_ACME_EMAIL"); v != "" {
		_ = k.Set("acme.email", v)
	}
	_ = k.Set("runtime.nginx_binary", DockerNginxBinary)
	_ = k.Set("runtime.xray_binary", DockerXrayBinary)
	_ = k.Set("paths.web_root", DockerWebRoot)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(k *koanf.Koanf) {
	_ = k.Set("server.initial_addr", "0.0.0.0:30080")
	_ = k.Set("server.internal_addr", "127.0.0.1:30081")
	_ = k.Set("server.public_https_port", 443)
	_ = k.Set("server.public_http_port", 80)
	_ = k.Set("server.managed_https_addr", "127.0.0.1:30443")
	_ = k.Set("server.start_initial_port", true)
	_ = k.Set("server.cookie_secure", false)
	_ = k.Set("paths.data_dir", "/var/lib/proxy-go")
	_ = k.Set("paths.log_dir", "/var/log/proxy-go")
	_ = k.Set("paths.db_file", "/var/lib/proxy-go/proxy-go.db")
	_ = k.Set("paths.cert_dir", "/var/lib/proxy-go/certs")
	_ = k.Set("paths.bin_dir", "/var/lib/proxy-go/bin")
	_ = k.Set("paths.nginx_conf_dir", "/var/lib/proxy-go/nginx")
	_ = k.Set("paths.xray_conf_dir", "/var/lib/proxy-go/xray")
	_ = k.Set("paths.web_root", DockerWebRoot)
	_ = k.Set("security.session_ttl_hours", 24)
	_ = k.Set("security.bcrypt_cost", 12)
	_ = k.Set("acme.directory_url", "https://acme-v02.api.letsencrypt.org/directory")
	_ = k.Set("acme.renew_before_days", 30)
	_ = k.Set("runtime.start_children", true)
	_ = k.Set("runtime.nginx_binary", DockerNginxBinary)
	_ = k.Set("runtime.xray_binary", DockerXrayBinary)
}

func (c Config) Validate() error {
	if c.Security.BcryptCost < 10 {
		return fmt.Errorf("security.bcrypt_cost must be >= 10")
	}
	return nil
}
