package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/audit"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	"github.com/proxy-go/proxy-go/internal/testutil"
	"github.com/proxy-go/proxy-go/internal/xray"
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

func TestXrayLogsRouteReturnsLogSummaryAndSingboxRouteIsGone(t *testing.T) {
	cfg := testutil.NewConfig(t)
	db := testutil.NewDB(t)
	router := Router(Deps{
		Cfg:       cfg,
		DB:        db,
		Audit:     audit.New(db),
		ACME:      acme.New(db),
		Xray:      xray.New(cfg, db, cfg.Runtime.XrayBinary),
		Limiter:   security.NewLoginLimiter(),
		Validator: validator.New(),
	})
	token := createSession(t, db)

	assertStatus(t, router, http.MethodGet, "/api/runtime/sing-box/logs", token, http.StatusNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/xray/logs", nil)
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
		Template:               "vless-xhttp",
		Protocol:               "vless",
		DomainID:               1,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "xhttp",
		Security:               "reality",
		XHTTPPath:              "/xhttp",
		XHTTPMode:              "auto",
		RealityPrivateKey:      "private",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60,
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
		Template:               "vless-xhttp",
		Protocol:               "vless",
		DomainID:               1,
		UUID:                   "11111111-1111-1111-1111-111111111111",
		ListenAddr:             "127.0.0.1",
		ListenPort:             31001,
		Network:                "xhttp",
		Security:               "reality",
		XHTTPPath:              "/xhttp",
		XHTTPMode:              "auto",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "apple.com",
		RealityHandshakePort:   443,
		RealityMaxTimeDiff:     60,
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
	if body.Data.URI != "vless://11111111-1111-1111-1111-111111111111@proxy.example.com:443?encryption=none&fp=chrome&mode=auto&path=%2Fxhttp&pbk=public&security=reality&sid=abcd1234&sni=apple.com&type=xhttp#main" {
		t.Fatalf("unexpected uri: %s", body.Data.URI)
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
