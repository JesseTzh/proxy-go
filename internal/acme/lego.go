package acme

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type HTTPProvider struct {
	service *Service
}

func NewHTTPProvider(service *Service) *HTTPProvider {
	return &HTTPProvider{service: service}
}

func (p *HTTPProvider) Present(domain, token, keyAuth string) error {
	return p.service.PutChallenge(domain, token, keyAuth, 30*time.Minute)
}

func (p *HTTPProvider) CleanUp(domain, token, keyAuth string) error {
	return p.service.db.Where("domain = ? AND token = ?", domain, token).Delete(&models.ACMEChallenge{}).Error
}

type legoUser struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *legoUser) GetEmail() string                        { return u.Email }
func (u *legoUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *legoUser) GetPrivateKey() crypto.PrivateKey        { return u.Key }

type CertificateIssuer interface {
	Obtain(domain string) (*certificate.Resource, error)
}

type legoIssuer struct {
	cfg     *config.Config
	service *Service
}

func NewLegoIssuer(cfg *config.Config, service *Service) CertificateIssuer {
	return &legoIssuer{cfg: cfg, service: service}
}

func (i *legoIssuer) Obtain(domain string) (*certificate.Resource, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	user := &legoUser{Email: i.cfg.ACME.Email, Key: key}
	legoCfg := lego.NewConfig(user)
	legoCfg.CADirURL = i.cfg.ACME.DirectoryURL
	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return nil, err
	}
	if err := client.Challenge.SetHTTP01Provider(NewHTTPProvider(i.service)); err != nil {
		return nil, err
	}
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg
	return client.Certificate.Obtain(certificate.ObtainRequest{Domains: []string{domain}, Bundle: true})
}

func (s *Service) IssueCertificateWithIssuer(cfg *config.Config, issuer CertificateIssuer, domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" || strings.Contains(domain, "/") {
		return errors.New("domain required")
	}
	res, err := issuer.Obtain(domain)
	if err != nil {
		s.saveFailedCertificate(domain, err)
		return err
	}
	return s.saveCertificateResource(cfg, domain, res)
}

func (s *Service) RenewCertificateWithIssuer(cfg *config.Config, issuer CertificateIssuer, id uint) error {
	var cert models.Certificate
	if err := s.db.First(&cert, id).Error; err != nil {
		return err
	}
	res, err := issuer.Obtain(cert.PrimaryDomain)
	if err != nil {
		s.saveFailedCertificate(cert.PrimaryDomain, err)
		return err
	}
	return s.saveCertificateResource(cfg, cert.PrimaryDomain, res)
}

func (s *Service) saveCertificateResource(cfg *config.Config, domain string, res *certificate.Resource) error {
	if res == nil || len(res.Certificate) == 0 || len(res.PrivateKey) == 0 {
		return errors.New("empty certificate resource")
	}
	dir := filepath.Join(cfg.Paths.CertDir, domain)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	if err := writeAtomic(certPath, res.Certificate, 0644); err != nil {
		return err
	}
	if err := writeAtomic(keyPath, res.PrivateKey, 0600); err != nil {
		return err
	}
	expiresAt := certificateExpiry(res.Certificate)
	var cert models.Certificate
	err := s.db.Where("primary_domain = ?", domain).First(&cert).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cert = models.Certificate{PrimaryDomain: domain}
	} else if err != nil {
		return err
	}
	cert.Issuer = "letsencrypt"
	cert.CertFilePath = certPath
	cert.KeyFilePath = keyPath
	cert.ExpiresAt = expiresAt
	cert.AutoRenew = true
	cert.Status = "valid"
	cert.ErrorMessage = ""
	return s.db.Save(&cert).Error
}

func (s *Service) saveFailedCertificate(domain string, cause error) {
	var cert models.Certificate
	err := s.db.Where("primary_domain = ?", domain).First(&cert).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cert = models.Certificate{PrimaryDomain: domain}
	}
	cert.Status = "failed"
	cert.ErrorMessage = cause.Error()
	_ = s.db.Save(&cert).Error
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func certificateExpiry(pemBytes []byte) *time.Time {
	for {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			return nil
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				return &cert.NotAfter
			}
			return nil
		}
		pemBytes = rest
	}
}

func (s *Service) RenewDue(cfg *config.Config, issuer CertificateIssuer, now time.Time) error {
	cutoff := now.Add(time.Duration(cfg.ACME.RenewBeforeDays) * 24 * time.Hour)
	var certs []models.Certificate
	if err := s.db.Where("auto_renew = ? AND expires_at <= ?", true, cutoff).Find(&certs).Error; err != nil {
		return err
	}
	for _, cert := range certs {
		if err := s.RenewCertificateWithIssuer(cfg, issuer, cert.ID); err != nil {
			return fmt.Errorf("renew %s: %w", cert.PrimaryDomain, err)
		}
	}
	return nil
}
