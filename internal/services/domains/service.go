package domains

import (
	"errors"
	"net"

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

type Usage struct {
	ReverseProxyRules int64 `json:"reverseProxyRules"`
	ProxyInbounds     int64 `json:"proxyInbounds"`
}

type DNSResult struct {
	Domain               string   `json:"domain"`
	Records              []string `json:"records"`
	MatchedCurrentServer bool     `json:"matchedCurrentServer"`
	Note                 string   `json:"note"`
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func NewWithCertificateIssuer(db *gorm.DB, acmeService *acme.Service, cfg *config.Config) *Service {
	return &Service{db: db, acme: acmeService, cfg: cfg}
}

func (s *Service) List() ([]models.Domain, error) {
	var items []models.Domain
	return items, s.db.Preload("Certificate").Order("id desc").Find(&items).Error
}

func (s *Service) Create(domain, remark, status string) (models.Domain, error) {
	if status == "" {
		status = "enabled"
	}
	item := models.Domain{Domain: domain, Remark: remark, Status: status}
	if err := s.db.Create(&item).Error; err != nil {
		return item, err
	}
	return item, nil
}

func (s *Service) Get(id uint) (models.Domain, error) {
	var item models.Domain
	return item, s.db.Preload("Certificate").First(&item, id).Error
}

func (s *Service) Update(id uint, remark, status string, certificateID *uint) (models.Domain, error) {
	item, err := s.Get(id)
	if err != nil {
		return item, err
	}
	item.Remark = remark
	if status != "" {
		item.Status = status
	}
	item.CertificateID = certificateID
	return item, s.db.Save(&item).Error
}

func (s *Service) Delete(id uint) error {
	usage, err := s.Usage(id)
	if err != nil {
		return err
	}
	if usage.ReverseProxyRules > 0 || usage.ProxyInbounds > 0 {
		return errors.New("domain is in use")
	}
	return s.db.Delete(&models.Domain{}, id).Error
}

func (s *Service) Usage(id uint) (Usage, error) {
	var usage Usage
	if err := s.db.Model(&models.ReverseProxyRule{}).Where("domain_id=?", id).Count(&usage.ReverseProxyRules).Error; err != nil {
		return usage, err
	}
	if err := s.db.Model(&models.ProxyInbound{}).Where("domain_id=?", id).Count(&usage.ProxyInbounds).Error; err != nil {
		return usage, err
	}
	return usage, nil
}

func (s *Service) DNSCheck(id uint) (DNSResult, error) {
	item, err := s.Get(id)
	if err != nil {
		return DNSResult{}, err
	}
	ips, _ := net.LookupHost(item.Domain)
	return DNSResult{
		Domain:               item.Domain,
		Records:              ips,
		MatchedCurrentServer: false,
		Note:                 "current public IP detection is environment dependent",
	}, nil
}

func (s *Service) IssueCertificate(id uint) error {
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	if s.acme == nil {
		return errors.New("ACME issuer is not configured")
	}
	if err := s.acme.IssueCertificate(item.Domain); err != nil {
		return err
	}
	return s.attachCertificate(item.ID, item.Domain)
}

func (s *Service) IssueCertificateWithIssuer(id uint, issuer acme.CertificateIssuer) error {
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	if s.acme == nil || s.cfg == nil {
		return errors.New("ACME issuer is not configured")
	}
	if err := s.acme.IssueCertificateWithIssuer(s.cfg, issuer, item.Domain); err != nil {
		return err
	}
	return s.attachCertificate(item.ID, item.Domain)
}

func (s *Service) RenewCertificate(id uint) error {
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	if item.CertificateID == nil {
		return gorm.ErrRecordNotFound
	}
	if s.acme == nil {
		return errors.New("ACME issuer is not configured")
	}
	if err := s.acme.RenewCertificate(*item.CertificateID); err != nil {
		return err
	}
	return s.attachCertificate(item.ID, item.Domain)
}

func (s *Service) DeleteCertificate(id uint) error {
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	if item.CertificateID == nil {
		return nil
	}
	certID := *item.CertificateID
	if err := s.db.Model(&models.Domain{}).Where("id = ?", item.ID).Update("certificate_id", nil).Error; err != nil {
		return err
	}
	return s.db.Delete(&models.Certificate{}, certID).Error
}

func (s *Service) attachCertificate(domainID uint, domain string) error {
	var cert models.Certificate
	if err := s.db.Where("primary_domain = ?", domain).First(&cert).Error; err != nil {
		return err
	}
	return s.db.Model(&models.Domain{}).Where("id = ?", domainID).Update("certificate_id", cert.ID).Error
}
