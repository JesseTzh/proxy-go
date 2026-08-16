package testutil

import (
	"testing"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/database"
	"gorm.io/gorm"
)

func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := NewConfig(t)
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

func NewConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Server: config.ServerConfig{
			InitialAddr:      "127.0.0.1:0",
			InternalAddr:     "127.0.0.1:30081",
			PublicHTTPSPort:  443,
			PublicHTTPPort:   80,
			ManagedHTTPSAddr: "127.0.0.1:30443",
			StartInitialPort: true,
			CookieSecure:     false,
		},
		Paths: config.PathsConfig{
			DataDir:        dir,
			LogDir:         dir,
			DBFile:         dir + "/proxy-go-test.db",
			CertDir:        dir + "/certs",
			BinDir:         dir + "/bin",
			NginxConfDir:   dir + "/nginx",
			SingBoxConfDir: dir + "/sing-box",
			WebRoot:        dir + "/web",
		},
		Security: config.SecurityConfig{
			InitialPassword: "test-password",
			SessionTTLHours: 24,
			BcryptCost:      10,
		},
		ACME: config.ACMEConfig{
			Email:           "admin@example.com",
			DirectoryURL:    "https://acme-staging-v02.api.letsencrypt.org/directory",
			RenewBeforeDays: 30,
		},
		Runtime: config.RuntimeConfig{
			StartChildren: false,
			NginxBinary:   config.DockerNginxBinary,
			SingBoxBinary: config.DockerSingBoxBinary,
		},
	}
}
