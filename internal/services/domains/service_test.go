package domains

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestCreateDefaultsStatusToEnabled(t *testing.T) {
	db := testutil.NewDB(t)
	svc := New(db)

	item, err := svc.Create("proxy.example.com", "main", "")
	if err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if item.Status != "enabled" {
		t.Fatalf("expected enabled status, got %q", item.Status)
	}
}

func TestDeleteRejectsReferencedDomain(t *testing.T) {
	db := testutil.NewDB(t)
	svc := New(db)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	db.Create(&domain)
	db.Create(&models.ReverseProxyRule{DomainID: domain.ID, TargetScheme: "http", TargetHost: "127.0.0.1", TargetPort: 8080})

	err := svc.Delete(domain.ID)
	if err == nil {
		t.Fatalf("expected in-use error")
	}
}

func TestUsageCountsReverseProxyAndInboundReferences(t *testing.T) {
	db := testutil.NewDB(t)
	svc := New(db)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	db.Create(&domain)
	db.Create(&models.ReverseProxyRule{DomainID: domain.ID, TargetScheme: "http", TargetHost: "127.0.0.1", TargetPort: 8080})
	db.Create(&models.ProxyInbound{
		DomainID:   domain.ID,
		Name:       "main",
		Template:   "vless-xhttp",
		UUID:       "uuid",
		ListenAddr: "127.0.0.1",
		ListenPort: 31001,
	})

	usage, err := svc.Usage(domain.ID)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if usage.ReverseProxyRules != 1 || usage.ProxyInbounds != 1 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}

func TestIssueCertificateAttachesCertificateToDomain(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	domain := models.Domain{Domain: "proxy.example.com", Status: "enabled"}
	db.Create(&domain)
	svc := NewWithCertificateIssuer(db, acme.New(db), cfg)

	if err := svc.IssueCertificateWithIssuer(domain.ID, staticIssuer{resource: testCertificate(t)}); err != nil {
		t.Fatalf("issue domain certificate: %v", err)
	}

	got, err := svc.Get(domain.ID)
	if err != nil {
		t.Fatalf("get domain: %v", err)
	}
	if got.CertificateID == nil || got.Certificate == nil {
		t.Fatalf("expected certificate attached to domain: %#v", got)
	}
	if got.Certificate.PrimaryDomain != domain.Domain || got.Certificate.Status != "valid" {
		t.Fatalf("unexpected certificate: %#v", got.Certificate)
	}
}

type staticIssuer struct {
	resource *certificate.Resource
}

func (s staticIssuer) Obtain(domain string) (*certificate.Resource, error) {
	return s.resource, nil
}

func testCertificate(t *testing.T) *certificate.Resource {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("serial: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "proxy.example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"proxy.example.com"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return &certificate.Resource{Certificate: certPEM, PrivateKey: keyPEM}
}
