package certificates

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

func TestIssueCertificatePersistsFilesAndRow(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := testutil.NewConfig(t)
	svc := New(db, acme.New(db), cfg)

	if err := svc.Issue("proxy.example.com", staticIssuer{resource: testCertificate(t)}); err != nil {
		t.Fatalf("issue certificate: %v", err)
	}
	var cert models.Certificate
	if err := db.Where("primary_domain = ?", "proxy.example.com").First(&cert).Error; err != nil {
		t.Fatalf("load certificate row: %v", err)
	}
	if cert.Status != "valid" || cert.CertFilePath == "" || cert.KeyFilePath == "" || cert.ExpiresAt == nil {
		t.Fatalf("unexpected certificate row: %#v", cert)
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
