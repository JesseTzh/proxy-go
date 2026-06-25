package database

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Paths.DBFile), 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(cfg.Paths.DBFile), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&models.AuthConfig{},
		&models.Session{},
		&models.SystemSetting{},
		&models.ACMEChallenge{},
		&models.Domain{},
		&models.Certificate{},
		&models.ReverseProxyRule{},
		&models.ProxyInbound{},
		&models.AuditLog{},
		&models.ProtocolCapability{},
	); err != nil {
		return nil, err
	}
	return db, seedDefaults(db, cfg)
}

func seedDefaults(db *gorm.DB, cfg *config.Config) error {
	var setting models.SystemSetting
	err := db.First(&setting, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.SystemSetting{ID: 1, InitialPortEnabled: cfg.Server.StartInitialPort, ACMEEmail: cfg.ACME.Email, RuntimeConfigStatus: "new"}
		if err := db.Create(&setting).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	caps := []struct {
		name    string
		enabled bool
	}{
		{"VLESS XHTTP Reality", true},
	}
	for _, c := range caps {
		var cap models.ProtocolCapability
		if err := db.Where("name = ?", c.name).First(&cap).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&models.ProtocolCapability{Name: c.name, Enabled: c.enabled}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	if err := db.Model(&models.ProxyInbound{}).
		Where("reality_max_time_diff = ?", 60).
		Update("reality_max_time_diff", 60000).Error; err != nil {
		return err
	}
	return nil
}
