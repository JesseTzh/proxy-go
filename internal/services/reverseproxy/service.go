package reverseproxy

import (
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
	if rule.TargetScheme == "" {
		rule.TargetScheme = "http"
	}
	if err := s.db.Save(&rule).Error; err != nil {
		return rule, err
	}
	_ = s.db.Preload("Domain").First(&rule, rule.ID).Error
	return rule, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.ReverseProxyRule{}, id).Error
}

func (s *Service) SetEnabled(id uint, enabled bool) error {
	return s.db.Model(&models.ReverseProxyRule{}).Where("id=?", id).Update("enabled", enabled).Error
}
