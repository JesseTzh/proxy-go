package inbounds

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/singbox"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestCreateVLESSRealityVisionGeneratesHiddenSecrets(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "apple.com",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if item.Template != "vless-reality-vision" || item.Network != "tcp" || item.Security != "reality" || item.Flow != "xtls-rprx-vision" {
		t.Fatalf("unexpected defaults: %#v", item)
	}
	if item.RouteSNI != "apple.com" {
		t.Fatalf("unexpected route sni: %#v", item)
	}
	if item.ListenAddr != "127.0.0.1" || item.ListenPort != 31001 {
		t.Fatalf("unexpected stream backend listen: %#v", item)
	}
	if item.UUID == "" || item.RealityPrivateKey == "" || item.RealityPublicKey == "" || item.RealityShortID == "" {
		t.Fatalf("missing generated credentials: %#v", item)
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	if strings.Contains(string(data), item.UUID) || strings.Contains(string(data), item.RealityPrivateKey) || strings.Contains(string(data), item.RealityShortID) {
		t.Fatalf("default json leaked secrets: %s", data)
	}
}

func TestCreateAnyTLSGeneratesPasswordAndUsesDomainSNI(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "vision.example.com", Status: "enabled"})
	db.Create(&models.Domain{ID: 2, Domain: "any.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	if _, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "apple.com",
	}); err != nil {
		t.Fatalf("create vision inbound: %v", err)
	}

	item, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls",
		DomainID: 2,
	})
	if err != nil {
		t.Fatalf("create anytls inbound: %v", err)
	}
	if item.Template != "anytls" || item.Protocol != "anytls" || item.Network != "tcp" || item.Security != "tls" {
		t.Fatalf("unexpected anytls defaults: %#v", item)
	}
	if item.RouteSNI != "any.example.com" || item.ListenPort != 31002 {
		t.Fatalf("unexpected anytls route/listen: %#v", item)
	}
	if item.Password != "secret-password" {
		t.Fatalf("missing anytls password: %#v", item)
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	if strings.Contains(string(data), item.Password) {
		t.Fatalf("default json leaked anytls password: %s", data)
	}
}

func TestUpdateAnyTLSRecomputesRouteSNIWhenDomainChanges(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "first.example.com", Status: "enabled"})
	db.Create(&models.Domain{ID: 2, Domain: "second.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls",
		DomainID: 1,
	})
	if err != nil {
		t.Fatalf("create anytls inbound: %v", err)
	}

	updated, err := svc.Update(context.Background(), item.ID, CreateRequest{
		Template: "anytls",
		Name:     "anytls",
		DomainID: 2,
	})
	if err != nil {
		t.Fatalf("update anytls inbound: %v", err)
	}
	if updated.RouteSNI != "second.example.com" {
		t.Fatalf("route sni was not recomputed: %#v", updated)
	}
}

func TestCreateRejectsManagedDomainAsRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "proxy.example.com",
	})
	if err == nil || !strings.Contains(err.Error(), "must not be a managed domain") {
		t.Fatalf("expected managed domain handshake error, got %v", err)
	}
}

func TestCreateRejectsManagementDomainAsRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	db.Model(&models.SystemSetting{}).Where("id = 1").Update("management_domain", "admin.example.com")
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "admin.example.com",
	})
	if err == nil || !strings.Contains(err.Error(), "must not be the management domain") {
		t.Fatalf("expected management domain handshake error, got %v", err)
	}
}

func TestCreateNormalizesRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: " Apple.COM. ",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if item.RealityHandshakeServer != "apple.com" || item.RouteSNI != "apple.com" {
		t.Fatalf("unexpected normalized handshake server: %#v", item)
	}
}

func TestCreateRejectsInvalidRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "apple.com;127.0.0.1:30443",
	})
	if err == nil || !strings.Contains(err.Error(), "valid domain name") {
		t.Fatalf("expected invalid handshake server error, got %v", err)
	}
}

func TestCreateRequiresRealityHandshakeServerForVision(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:     "vision",
		DomainID: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "realityHandshakeServer required") {
		t.Fatalf("expected missing handshake server error, got %v", err)
	}
}

func TestCreateRejectsDuplicateRouteSNI(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "any.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	if _, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls-a",
		DomainID: 1,
	}); err != nil {
		t.Fatalf("create first anytls inbound: %v", err)
	}

	_, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls-b",
		DomainID: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "routeSni must be unique") {
		t.Fatalf("expected duplicate route sni error, got %v", err)
	}
}

func TestConfigDetailsReturnsRenderedVisionInboundJSON(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "apple.com",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	details, err := svc.ConfigDetails(item.ID)
	if err != nil {
		t.Fatalf("config details: %v", err)
	}
	reality := details["tls"].(map[string]any)["reality"].(map[string]any)
	if details["type"] != "vless" || reality["private_key"] != "private-key" {
		t.Fatalf("unexpected rendered details: %#v", details)
	}
	if details["listen"] != "127.0.0.1" || details["listen_port"] != 31001 {
		t.Fatalf("unexpected rendered listen: %#v", details)
	}
}

func TestConfigDetailsReturnsRenderedAnyTLSInboundJSON(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "any.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls",
		DomainID: 1,
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	details, err := svc.ConfigDetails(item.ID)
	if err != nil {
		t.Fatalf("config details: %v", err)
	}
	tls := details["tls"].(map[string]any)
	if details["type"] != "anytls" || tls["server_name"] != "any.example.com" {
		t.Fatalf("unexpected anytls details: %#v", details)
	}
}

func TestShareDetailsBuildsVLESSRealityVisionURI(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "vision",
		DomainID:               1,
		RealityHandshakeServer: "apple.com",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	share, err := svc.ShareDetails(item.ID)
	if err != nil {
		t.Fatalf("share details: %v", err)
	}

	want := "vless://11111111-1111-1111-1111-111111111111@proxy.example.com:443?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=public-key&security=reality&sid=abcd1234&sni=apple.com&type=tcp#vision"
	if share.URI != want {
		t.Fatalf("unexpected share uri:\nwant %s\n got %s", want, share.URI)
	}
}

func TestShareDetailsBuildsAnyTLSURI(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "any.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Template: "anytls",
		Name:     "anytls",
		DomainID: 1,
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	share, err := svc.ShareDetails(item.ID)
	if err != nil {
		t.Fatalf("share details: %v", err)
	}

	want := "anytls://secret-password@any.example.com:443?security=tls&sni=any.example.com#anytls"
	if share.URI != want {
		t.Fatalf("unexpected share uri:\nwant %s\n got %s", want, share.URI)
	}
}

func fakeGenerator() singbox.StaticCredentialGenerator {
	return singbox.StaticCredentialGenerator{Credentials: singbox.Credentials{
		UUID:              "11111111-1111-1111-1111-111111111111",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
		Password:          "secret-password",
	}}
}
