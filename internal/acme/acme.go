package acme

import (
	"errors"
	"time"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	cfg    *config.Config
	issuer CertificateIssuer
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

func NewWithConfig(db *gorm.DB, cfg *config.Config) *Service {
	s := &Service{db: db, cfg: cfg}
	s.issuer = NewLegoIssuer(cfg, s)
	return s
}

func (s *Service) GetKeyAuthorization(token string) (string, bool) {
	var ch models.ACMEChallenge
	if err := s.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&ch).Error; err != nil {
		return "", false
	}
	return ch.KeyAuthorization, true
}

func (s *Service) PutChallenge(domain, token, keyAuth string, ttl time.Duration) error {
	return s.db.Create(&models.ACMEChallenge{Domain: domain, Token: token, KeyAuthorization: keyAuth, ExpiresAt: time.Now().Add(ttl)}).Error
}

func (s *Service) CleanupExpired() error {
	return s.db.Where("expires_at < ?", time.Now()).Delete(&models.ACMEChallenge{}).Error
}

func (s *Service) IssueCertificate(domain string) error {
	if s.cfg == nil || s.issuer == nil {
		return errors.New("ACME issuer is not configured")
	}
	return s.IssueCertificateWithIssuer(s.cfg, s.issuer, domain)
}

func (s *Service) RenewCertificate(id uint) error {
	if s.cfg == nil || s.issuer == nil {
		return errors.New("ACME issuer is not configured")
	}
	return s.RenewCertificateWithIssuer(s.cfg, s.issuer, id)
}

func (s *Service) RenewDueCertificates(now time.Time) error {
	if s.cfg == nil || s.issuer == nil {
		return nil
	}
	return s.RenewDue(s.cfg, s.issuer, now)
}
