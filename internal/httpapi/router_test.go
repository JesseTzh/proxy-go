package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/audit"
	"github.com/proxy-go/proxy-go/internal/models"
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
		Logs []string `json:"logs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Logs == nil {
		t.Fatalf("expected logs field to be present")
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
		Network:                "raw",
		Security:               "reality",
		Flow:                   "xtls-rprx-vision",
		RealityPrivateKey:      "private",
		RealityPublicKey:       "public",
		RealityShortID:         "abcd1234",
		RealityHandshakeServer: "www.cloudflare.com",
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
