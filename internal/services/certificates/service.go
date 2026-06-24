package certificates

import (
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db   *gorm.DB
	acme *acme.Service
	cfg  *config.Config
}

func New(db *gorm.DB, acmeService *acme.Service, cfg *config.Config) *Service {
	return &Service{db: db, acme: acmeService, cfg: cfg}
}

func (s *Service) List() ([]models.Certificate, error) {
	var items []models.Certificate
	return items, s.db.Order("id desc").Find(&items).Error
}

func (s *Service) Issue(domain string, issuer acme.CertificateIssuer) error {
	return s.acme.IssueCertificateWithIssuer(s.cfg, issuer, domain)
}

func (s *Service) IssueDefault(domain string) error {
	return s.acme.IssueCertificate(domain)
}

func (s *Service) Renew(id uint) error {
	return s.acme.RenewCertificate(id)
}

func (s *Service) Delete(id uint) error {
	var n int64
	if err := s.db.Model(&models.Domain{}).Where("certificate_id=?", id).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return gorm.ErrInvalidData
	}
	return s.db.Delete(&models.Certificate{}, id).Error
}
