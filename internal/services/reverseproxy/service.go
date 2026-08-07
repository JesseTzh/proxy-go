package reverseproxy

import (
	"errors"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]models.ReverseProxyRule, error) {
	var items []models.ReverseProxyRule
	return items, s.db.Preload("Domain").Order("id desc").Find(&items).Error
}

func (s *Service) Create(rule models.ReverseProxyRule) (models.ReverseProxyRule, error) {
	if err := s.validateDomainMapping(rule.DomainID, 0); err != nil {
		return rule, err
	}
	if rule.TargetScheme == "" {
		rule.TargetScheme = "http"
	}
	if err := s.db.Create(&rule).Error; err != nil {
		return rule, err
	}
	_ = s.db.Preload("Domain").First(&rule, rule.ID).Error
	return rule, nil
}

func (s *Service) Update(id uint, rule models.ReverseProxyRule) (models.ReverseProxyRule, error) {
	var existing models.ReverseProxyRule
	if err := s.db.First(&existing, id).Error; err != nil {
		return existing, err
	}
	rule.ID = existing.ID
	if err := s.validateDomainMapping(rule.DomainID, existing.ID); err != nil {
		return rule, err
	}
	if rule.TargetScheme == "" {
		rule.TargetScheme = "http"
	}
	if err := s.db.Save(&rule).Error; err != nil {
		return rule, err
	}
	_ = s.db.Preload("Domain").First(&rule, rule.ID).Error
	return rule, nil
}

func (s *Service) validateDomainMapping(domainID, ruleID uint) error {
	var domain models.Domain
	if err := s.db.First(&domain, domainID).Error; err != nil {
		return err
	}
	var setting models.SystemSetting
	if err := s.db.First(&setting, 1).Error; err == nil && setting.ManagementDomain != "" && domain.Domain == setting.ManagementDomain {
		return errors.New("domain is reserved for management panel")
	}
	var count int64
	query := s.db.Model(&models.ReverseProxyRule{}).Where("domain_id = ?", domainID)
	if ruleID != 0 {
		query = query.Where("id <> ?", ruleID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("domain is already mapped")
	}
	if err := s.db.Model(&models.ProxyInbound{}).Where("domain_id = ?", domainID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("domain is already mapped")
	}
	return nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.ReverseProxyRule{}, id).Error
}

func (s *Service) SetEnabled(id uint, enabled bool) error {
	return s.db.Model(&models.ReverseProxyRule{}).Where("id=?", id).Update("enabled", enabled).Error
}
