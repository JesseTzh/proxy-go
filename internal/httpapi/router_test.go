package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-playground/validator/v10"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/audit"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	"github.com/proxy-go/proxy-go/internal/singbox"
	"github.com/proxy-go/proxy-go/internal/testutil"
	"gorm.io/gorm"
)

func TestProtectedRoutesRequireAuth(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	var body struct {
		OK    bool `json:"ok"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.OK || body.Error.Message != "unauthorized" {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestCertificateRoutesAreScopedUnderDomains(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	if err := db.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	token := createSession(t, db)

	assertStatus(t, router, http.MethodGet, "/api/certificates", token, http.StatusNotFound)
	assertStatus(t, router, http.MethodPost, "/api/certificates/issue", token, http.StatusNotFound)
	assertStatus(t, router, http.MethodPost, "/api/domains/999/certificate/issue", token, http.StatusNotFound)
	assertStatus(t, router, http.MethodPost, "/api/domains/"+itoa(domain.ID)+"/certificate/issue", token, http.StatusNotImplemented)
}

func TestDomainCertificateChangesApplyNginx(t *testing.T) {
	for _, operation := range []string{"issue", "renew"} {
		for _, failure := range []string{"", "issuer", "check", "reload"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				cfg := testutil.NewConfig(t)
				db := testutil.NewDB(t)
				binary, logPath := fakeNginxBinary(t)
				if failure == "check" || failure == "reload" {
					argument := "-t"
					if failure == "reload" {
						argument = "-s"
					}
					script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + strconv.Quote(logPath) + "\nif [ \"$1\" = " + strconv.Quote(argument) + " ]; then exit 1; fi\nexit 0\n"
					if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
						t.Fatalf("write failing nginx: %v", err)
					}
				}
				domain := models.Domain{Domain: "dashboard.proxy.example.com", Status: "enabled"}
				if err := db.Create(&domain).Error; err != nil {
					t.Fatalf("create domain: %v", err)
				}
				if err := db.Model(&models.SystemSetting{}).Where("id=1").Update("management_domain", domain.Domain).Error; err != nil {
					t.Fatalf("set management domain: %v", err)
				}
				if operation == "renew" {
					cert := models.Certificate{PrimaryDomain: domain.Domain, Status: "valid"}
					if err := db.Create(&cert).Error; err != nil {
						t.Fatalf("create certificate: %v", err)
					}
					if err := db.Model(&domain).Update("certificate_id", cert.ID).Error; err != nil {
						t.Fatalf("attach certificate: %v", err)
					}
				}
				issuer := &routerCertificateIssuer{}
				if failure == "issuer" {
					issuer.err = errors.New("issuer failed")
				}
				router := Router(Deps{
					Cfg:       cfg,
					DB:        db,
					Audit:     audit.New(db),
					ACME:      acme.NewWithIssuer(db, cfg, issuer),
					Nginx:     nginx.New(cfg, db, binary),
					Limiter:   security.NewLoginLimiter(),
					Validator: validator.New(),
				})
				token := createSession(t, db)
				req := httptest.NewRequest(http.MethodPost, "/api/domains/"+itoa(domain.ID)+"/certificate/"+operation, nil)
				req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				wantStatus := http.StatusOK
				if failure != "" {
					wantStatus = http.StatusInternalServerError
					if failure == "issuer" && operation == "issue" {
						wantStatus = http.StatusNotImplemented
					}
				}
				if rec.Code != wantStatus {
					t.Fatalf("expected %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
				}
				if issuer.domain != domain.Domain {
					t.Fatalf("expected issuance for %s, got %s", domain.Domain, issuer.domain)
				}
				var setting models.SystemSetting
				if err := db.First(&setting, 1).Error; err != nil {
					t.Fatalf("load settings: %v", err)
				}
				if (setting.LastNginxReloadAt != nil) != (failure == "") {
					t.Fatalf("unexpected nginx reload timestamp: %v", setting.LastNginxReloadAt)
				}
				if failure == "issuer" {
					if _, err := os.Stat(logPath); !os.IsNotExist(err) {
						t.Fatalf("nginx should not run after issuance failure: %v", err)
					}
					return
				}
				if err := db.Preload("Certificate").First(&domain, domain.ID).Error; err != nil {
					t.Fatalf("load domain: %v", err)
				}
				if domain.Certificate == nil || domain.Certificate.Status != "valid" {
					t.Fatalf("certificate should remain bound after issuance: %#v", domain.Certificate)
				}
				if failure == "check" || failure == "reload" {
					if !bytes.Contains(rec.Body.Bytes(), []byte("nginx apply failed")) {
						t.Fatalf("expected partial-success error: %s", rec.Body.String())
					}
					wantReloads := 0
					if failure == "reload" {
						wantReloads = 1
					}
					assertFakeNginxReloads(t, logPath, wantReloads)
					return
				}
				assertFakeNginxReloads(t, logPath, 1)
				conf, err := os.ReadFile(filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf"))
				if err != nil {
					t.Fatalf("read nginx config: %v", err)
				}
				certPath := filepath.Join(cfg.Paths.CertDir, domain.Domain, "fullchain.pem")
				if !bytes.Contains(conf, []byte("ssl_certificate "+certPath+";")) || bytes.Contains(conf, []byte("/default/fullchain.pem")) {
					t.Fatalf("nginx did not switch to management certificate:\n%s", conf)
				}
				var body struct {
					OK   bool          `json:"ok"`
					Data models.Domain `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if !body.OK || body.Data.Certificate == nil || body.Data.Certificate.Status != "valid" {
					t.Fatalf("unexpected certificate response: %s", rec.Body.String())
				}
			})
		}
	}
}

type routerCertificateIssuer struct {
	domain string
	err    error
}

func (s *routerCertificateIssuer) Obtain(domain string) (*certificate.Resource, error) {
	s.domain = domain
	if s.err != nil {
		return nil, s.err
	}
	return &certificate.Resource{Certificate: []byte("test certificate"), PrivateKey: []byte("test key")}, nil
}

func TestSingBoxLogsRouteReturnsLogSummaryAndLegacyCoreRouteIsGone(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		SingBox:   singbox.New(cfg, db, cfg.Runtime.SingBoxBinary),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)

	assertStatus(t, router, http.MethodGet, "/api/runtime/"+("x"+"ray")+"/logs", token, http.StatusNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/sing-box/logs", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Logs []string `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Logs == nil {
		t.Fatalf("expected logs field to be present")
	}
}

func TestNginxConfigRouteReturnsRenderedConfig(t *testing.T) {
	cfg := testutil.NewConfig(t)
	if err := os.MkdirAll(cfg.Paths.NginxConfDir, 0755); err != nil {
		t.Fatalf("create nginx dir: %v", err)
	}
	confPath := filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf")
	if err := os.WriteFile(confPath, []byte("stream { apple.com 127.0.0.1:31001; }"), 0644); err != nil {
		t.Fatalf("write nginx config: %v", err)
	}
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/nginx/config", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Path != confPath || body.Data.Content != "stream { apple.com 127.0.0.1:31001; }" {
		t.Fatalf("unexpected nginx config response: %s", rec.Body.String())
	}
}

func TestInboundRoutesReplaceVLESSRoutes(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)
	if err := db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"}).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		ID:                     1,
		Name:                   "main",
		Template:               "vless-reality-vision",
		Protocol:               "vless",
		DomainID:               1,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "tcp",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RouteSNI:               "apple.com",
		RealityPrivateKey:      "private",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60000,
		Enabled:                true,
	}).Error; err != nil {
		t.Fatalf("create vless: %v", err)
	}

	assertStatus(t, router, http.MethodGet, "/api/vless", token, http.StatusNotFound)
	assertStatus(t, router, http.MethodGet, "/api/inbounds", token, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/inbounds/1/config", token, http.StatusOK)
}

func TestInboundShareRouteReturnsVLESSURI(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)
	if err := db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"}).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		ID:                     1,
		Name:                   "main",
		Template:               "vless-reality-vision",
		Protocol:               "vless",
		DomainID:               1,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "tcp",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RouteSNI:               "apple.com",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60000,
		Enabled:                true,
	}).Error; err != nil {
		t.Fatalf("create vless: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/inbounds/1/share", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			URI string `json:"uri"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.URI != "vless://11111111-1111-1111-1111-111111111111@proxy.example.com:443?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=public&security=reality&sid=abcd1234&sni=apple.com&type=tcp#main" {
		t.Fatalf("unexpected uri: %s", body.Data.URI)
	}
}

func TestUpdatingInboundSNIImmediatelyAppliesNginxAndSingBox(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	nginxBinary, _ := fakeNginxBinary(t)
	singBoxBinary, singBoxLogPath := fakeSingBoxRuntimeBinary(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Nginx:     nginx.New(cfg, db, nginxBinary),
		SingBox:   singbox.New(cfg, db, singBoxBinary),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)
	if err := db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"}).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if err := db.Create(&models.ProxyInbound{
		ID:                     1,
		Name:                   "main",
		Template:               "vless-reality-vision",
		Protocol:               "vless",
		DomainID:               1,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "tcp",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RouteSNI:               "apple.com",
		RealityPrivateKey:      "private",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60000,
		Enabled:                true,
	}).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	body := []byte(`{"template":"vless-reality-vision","name":"main","domainId":1,"realityHandshakeServer":"cloudflare.com"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/inbounds/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update inbound: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	nginxConfig, err := os.ReadFile(filepath.Join(cfg.Paths.NginxConfDir, "nginx.conf"))
	if err != nil {
		t.Fatalf("read nginx config: %v", err)
	}
	if !bytes.Contains(nginxConfig, []byte("cloudflare.com 127.0.0.1:31001;")) || bytes.Contains(nginxConfig, []byte("apple.com 127.0.0.1:31001;")) {
		t.Fatalf("nginx route SNI was not updated:\n%s", nginxConfig)
	}
	singBoxConfig, err := os.ReadFile(filepath.Join(cfg.Paths.SingBoxConfDir, "config.json"))
	if err != nil {
		t.Fatalf("read sing-box config: %v", err)
	}
	if !bytes.Contains(singBoxConfig, []byte(`"server_name": "cloudflare.com"`)) {
		t.Fatalf("sing-box SNI was not updated:\n%s", singBoxConfig)
	}
	logData, err := os.ReadFile(singBoxLogPath)
	if err != nil {
		t.Fatalf("read fake sing-box log: %v", err)
	}
	if !bytes.Contains(logData, []byte("check -c")) {
		t.Fatalf("sing-box config was not applied:\n%s", logData)
	}
}

func TestReverseProxyMutationsApplyNginx(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	binary, logPath := fakeNginxBinary(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Nginx:     nginx.New(cfg, db, binary),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)
	if err := db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"}).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}

	body := []byte(`{"domainId":1,"targetScheme":"http","targetHost":"127.0.0.1","targetPort":8080,"preserveHost":true,"webSocket":true,"passRealIp":true,"enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/reverse-proxies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create reverse proxy: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertFakeNginxReloads(t, logPath, 1)

	req = httptest.NewRequest(http.MethodPost, "/api/reverse-proxies/1/disable", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable reverse proxy: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertFakeNginxReloads(t, logPath, 2)

	req = httptest.NewRequest(http.MethodPost, "/api/reverse-proxies/1/enable", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("enable reverse proxy: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertFakeNginxReloads(t, logPath, 3)

	update := []byte(`{"domainId":1,"targetScheme":"https","targetHost":"10.0.0.2","targetPort":8443,"preserveHost":false,"webSocket":false,"passRealIp":true,"enabled":true}`)
	req = httptest.NewRequest(http.MethodPut, "/api/reverse-proxies/1", bytes.NewReader(update))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update reverse proxy: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertFakeNginxReloads(t, logPath, 4)

	req = httptest.NewRequest(http.MethodDelete, "/api/reverse-proxies/1", nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete reverse proxy: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	assertFakeNginxReloads(t, logPath, 5)
}

func createSession(t *testing.T, db *gorm.DB) string {
	t.Helper()
	token := "test-token"
	hash := security.HashToken(token)
	if err := db.Create(&models.Session{TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	return token
}

func assertStatus(t *testing.T, router http.Handler, method, path, token string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "proxy_go_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s: expected %d, got %d", method, path, want, rec.Code)
	}
}

func itoa(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func fakeNginxBinary(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "nginx")
	logPath := filepath.Join(dir, "nginx-args.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + strconv.Quote(logPath) + "\nexit 0\n"
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write fake nginx: %v", err)
	}
	return binary, logPath
}

func fakeSingBoxRuntimeBinary(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "sing-box")
	logPath := filepath.Join(dir, "sing-box-args.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + strconv.Quote(logPath) + "\nexit 0\n"
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("write fake sing-box: %v", err)
	}
	return binary, logPath
}

func assertFakeNginxReloads(t *testing.T, logPath string, want int) {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake nginx log: %v", err)
	}
	got := bytes.Count(b, []byte("-s reload"))
	if got != want {
		t.Fatalf("expected %d nginx reloads, got %d:\n%s", want, got, string(b))
	}
}
