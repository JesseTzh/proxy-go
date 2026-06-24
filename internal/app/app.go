package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/api"
	"github.com/proxy-go/proxy-go/internal/audit"
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/database"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	"github.com/proxy-go/proxy-go/internal/xray"
	"github.com/robfig/cron/v3"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
)

type Application struct {
	cfg          *config.Config
	db           *gorm.DB
	httpInitial  *http.Server
	httpInternal *http.Server
	nginx        *nginx.Service
	xray         *xray.Service
	cron         *cron.Cron
}

func New(cfg *config.Config) (*Application, error) {
	started := time.Now()
	setupLogger(cfg, os.Stdout)
	slog.Info("application init starting",
		"dataDir", cfg.Paths.DataDir,
		"logDir", cfg.Paths.LogDir,
		"dbFile", cfg.Paths.DBFile,
		"binDir", cfg.Paths.BinDir,
		"runtimeStartChildren", cfg.Runtime.StartChildren,
		"initialAddr", cfg.Server.InitialAddr,
		"internalAddr", cfg.Server.InternalAddr,
		"publicHTTPPort", cfg.Server.PublicHTTPPort,
		"publicHTTPSPort", cfg.Server.PublicHTTPSPort,
	)
	for _, dir := range []string{cfg.Paths.DataDir, cfg.Paths.LogDir, cfg.Paths.BinDir, cfg.Paths.CertDir, cfg.Paths.NginxConfDir, cfg.Paths.XrayConfDir} {
		stepStarted := time.Now()
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		slog.Info("startup directory ready", "path", dir, "elapsed", time.Since(stepStarted).String())
	}
	stepStarted := time.Now()
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	slog.Info("startup database ready", "dbFile", cfg.Paths.DBFile, "elapsed", time.Since(stepStarted).String())
	stepStarted = time.Now()
	if err := seedAuth(db, cfg); err != nil {
		return nil, err
	}
	slog.Info("startup auth ready", "elapsed", time.Since(stepStarted).String())
	stepStarted = time.Now()
	if err := ensureDefaultCertificate(cfg); err != nil {
		return nil, err
	}
	slog.Info("startup default certificate ready", "elapsed", time.Since(stepStarted).String())
	ng := nginx.New(cfg, db, cfg.Runtime.NginxBinary)
	xr := xray.New(cfg, db, cfg.Runtime.XrayBinary)
	aud := audit.New(db)
	ac := acme.NewWithConfig(db, cfg)
	deps := api.Deps{Cfg: cfg, DB: db, Audit: aud, ACME: ac, Nginx: ng, Xray: xr, Limiter: security.NewLoginLimiter(), Validator: validator.New()}
	gin.SetMode(gin.ReleaseMode)
	r := api.Router(deps)
	webRoot := resolveWebRoot(cfg.Paths.WebRoot)
	attachWeb(r, webRoot)
	internal := api.Router(deps)
	attachWeb(internal, webRoot)
	app := &Application{cfg: cfg, db: db, nginx: ng, xray: xr, httpInitial: &http.Server{Addr: cfg.Server.InitialAddr, Handler: r}, httpInternal: &http.Server{Addr: cfg.Server.InternalAddr, Handler: internal}, cron: cron.New()}
	app.registerCron(ac)
	slog.Info("application init completed", "elapsed", time.Since(started).String())
	return app, nil
}

func setupLogger(cfg *config.Config, console io.Writer) {
	w := &lumberjack.Logger{Filename: filepath.Join(cfg.Paths.LogDir, "proxy-go.log"), MaxSize: 50, MaxBackups: 10, MaxAge: 30, Compress: true}
	out := io.MultiWriter(w, console)
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo})))
}

func seedAuth(db *gorm.DB, cfg *config.Config) error {
	var count int64
	db.Model(&models.AuthConfig{}).Count(&count)
	if count > 0 {
		return nil
	}
	pwd := cfg.Security.InitialPassword
	if env := os.Getenv("PROXY_GO_INITIAL_PASSWORD"); env != "" {
		pwd = env
	}
	if pwd == "" {
		return fmt.Errorf("initial password is required for first startup: set security.initial_password or PROXY_GO_INITIAL_PASSWORD")
	}
	h, err := security.HashPassword(pwd, cfg.Security.BcryptCost)
	if err != nil {
		return err
	}
	return db.Create(&models.AuthConfig{ID: 1, PasswordHash: h}).Error
}

func attachWeb(r *gin.Engine, webRoot string) {
	r.StaticFS("/assets", http.Dir(filepath.Join(webRoot, "assets")))
	r.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || c.Request.Method == http.MethodGet {
			b, err := os.ReadFile(filepath.Join(webRoot, "index.html"))
			if err == nil {
				c.Data(200, "text/html; charset=utf-8", b)
				return
			}
		}
		c.JSON(404, gin.H{"error": "not found"})
	})
}

func resolveWebRoot(configured string) string {
	if _, err := os.Stat(filepath.Join(configured, "index.html")); err == nil {
		return configured
	}
	if _, err := os.Stat(filepath.Join("web", "dist", "index.html")); err == nil {
		slog.Warn("configured web root missing; using local web/dist fallback", "configured", configured)
		return filepath.Join("web", "dist")
	}
	return configured
}

func (a *Application) registerCron(ac *acme.Service) {
	_, _ = a.cron.AddFunc("0 3 * * *", func() { _ = ac.CleanupExpired() })
	_, _ = a.cron.AddFunc("0 4 * * *", func() {
		if err := ac.RenewDueCertificates(time.Now()); err != nil {
			slog.Warn("certificate renewal check failed", "error", err)
		}
	})
}

func (a *Application) Start(ctx context.Context) error {
	started := time.Now()
	slog.Info("application start beginning")

	internalListener, err := net.Listen("tcp", a.httpInternal.Addr)
	if err != nil {
		return fmt.Errorf("listen internal http server: %w", err)
	}

	var setting models.SystemSetting
	_ = a.db.First(&setting, 1).Error
	var initialListener net.Listener
	if setting.InitialPortEnabled {
		initialListener, err = net.Listen("tcp", a.httpInitial.Addr)
		if err != nil {
			_ = internalListener.Close()
			return fmt.Errorf("listen initial http server: %w", err)
		}
	} else {
		slog.Info("initial http server disabled by setting")
	}

	a.cron.Start()
	slog.Info("cron scheduler started")
	go serve("internal", a.httpInternal, internalListener)
	if initialListener != nil {
		go serve("initial", a.httpInitial, initialListener)
	}
	a.startManagedChildren(ctx)
	slog.Info("application start completed", "elapsed", time.Since(started).String())
	return nil
}

func (a *Application) startManagedChildren(ctx context.Context) {
	if !a.cfg.Runtime.StartChildren {
		slog.Info("managed child processes disabled by config")
		return
	}
	stepStarted := time.Now()
	if err := a.nginx.Start(ctx); err != nil {
		slog.Warn("start nginx failed", "error", err)
	} else {
		slog.Info("nginx process start requested", "elapsed", time.Since(stepStarted).String())
	}
	stepStarted = time.Now()
	if err := a.xray.Start(ctx); err != nil {
		slog.Warn("start xray failed", "error", err)
	} else {
		slog.Info("xray process start requested", "elapsed", time.Since(stepStarted).String())
	}
}

func serve(name string, srv *http.Server, listener net.Listener) {
	slog.Info("http server starting", "name", name, "addr", srv.Addr)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server failed", "name", name, "error", err)
	}
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.cron.Stop()
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = a.httpInitial.Shutdown(ctx2)
	_ = a.httpInternal.Shutdown(ctx2)
	_ = a.nginx.Proc.Stop(ctx2)
	_ = a.xray.Proc.Stop(ctx2)
	return nil
}

func ensureDefaultCertificate(cfg *config.Config) error {
	dir := filepath.Join(cfg.Paths.CertDir, "default")
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			return nil
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "proxy-go default certificate"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		_ = certFile.Close()
		return err
	}
	_ = certFile.Close()
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		_ = keyFile.Close()
		return err
	}
	return keyFile.Close()
}
