package inbounds

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
	"github.com/proxy-go/proxy-go/internal/xray"
)

func TestCreateVLESSXHTTPRealityGeneratesHiddenSecrets(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "main",
		DomainID:               1,
		RealityHandshakeServer: "apple.com",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if item.Template != "vless-xhttp" || item.Network != "xhttp" || item.Security != "reality" || item.XHTTPPath != "/xhttp" {
		t.Fatalf("unexpected defaults: %#v", item)
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

func TestCreateRejectsManagedDomainAsRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "main",
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
		Name:                   "main",
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
		Name:                   "main",
		DomainID:               1,
		RealityHandshakeServer: " Apple.COM. ",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if item.RealityHandshakeServer != "apple.com" {
		t.Fatalf("unexpected normalized handshake server: %#v", item)
	}
}

func TestCreateRejectsInvalidRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "main",
		DomainID:               1,
		RealityHandshakeServer: "apple.com;127.0.0.1:30443",
	})
	if err == nil || !strings.Contains(err.Error(), "valid domain name") {
		t.Fatalf("expected invalid handshake server error, got %v", err)
	}
}

func TestCreateRequiresRealityHandshakeServer(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:     "main",
		DomainID: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "realityHandshakeServer required") {
		t.Fatalf("expected missing handshake server error, got %v", err)
	}
}

func TestCreateWithMinimalRequestAppliesHiddenDefaults(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())

	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "main",
		DomainID:               1,
		XHTTPPath:              "/xhttp",
		RealityHandshakeServer: "apple.com",
	})
	if err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if !item.Enabled || item.XHTTPMode != "auto" || item.RealityMaxTimeDiff != 60 || item.RealityHandshakePort != 443 {
		t.Fatalf("unexpected hidden defaults: %#v", item)
	}
}

func TestConfigDetailsReturnsRenderedInboundJSON(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "main",
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
	stream := details["streamSettings"].(map[string]any)
	reality := stream["realitySettings"].(map[string]any)
	if details["protocol"] != "vless" || reality["privateKey"] != "private-key" {
		t.Fatalf("unexpected rendered details: %#v", details)
	}
	if details["listen"] != "127.0.0.1" || details["port"] != 31001 || reality["target"] != "apple.com:443" {
		t.Fatalf("unexpected rendered stream backend details: %#v", details)
	}
}

func TestShareDetailsBuildsVLESSXHTTPURI(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	db.Create(&models.Domain{ID: 1, Domain: "proxy.example.com", Status: "enabled"})
	svc := New(db, cfg, fakeGenerator())
	item, err := svc.Create(context.Background(), CreateRequest{
		Name:                   "xhttp",
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

	want := "vless://11111111-1111-1111-1111-111111111111@proxy.example.com:443?encryption=none&fp=chrome&mode=auto&path=%2Fxhttp&pbk=public-key&security=reality&sid=abcd1234&sni=apple.com&type=xhttp#xhttp"
	if share.URI != want {
		t.Fatalf("unexpected share uri:\nwant %s\n got %s", want, share.URI)
	}
}

func fakeGenerator() xray.StaticCredentialGenerator {
	return xray.StaticCredentialGenerator{Credentials: xray.Credentials{
		UUID:              "11111111-1111-1111-1111-111111111111",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
	}}
}
