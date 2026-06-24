package inbounds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
	"github.com/proxy-go/proxy-go/internal/xray"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	generator xray.CredentialGenerator
}

type CreateRequest struct {
	Name                   string `json:"name"`
	Template               string `json:"template"`
	DomainID               uint   `json:"domainId"`
	ListenPort             int    `json:"listenPort"`
	XHTTPPath              string `json:"xhttpPath"`
	XHTTPMode              string `json:"xhttpMode"`
	Security               string `json:"security"`
	RealityHandshakeServer string `json:"realityHandshakeServer"`
	RealityHandshakePort   int    `json:"realityHandshakePort"`
	RealityMaxTimeDiff     int    `json:"realityMaxTimeDiff"`
	Enabled                bool   `json:"enabled"`
}

type ShareDetails struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	Template string `json:"template"`
	URI      string `json:"uri"`
}

func New(db *gorm.DB, cfg *config.Config, generator xray.CredentialGenerator) *Service {
	_ = cfg
	return &Service{db: db, generator: generator}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (models.ProxyInbound, error) {
	item := models.ProxyInbound{
		Name:                   req.Name,
		Template:               req.Template,
		DomainID:               req.DomainID,
		ListenPort:             req.ListenPort,
		XHTTPPath:              req.XHTTPPath,
		XHTTPMode:              req.XHTTPMode,
		Security:               req.Security,
		RealityHandshakeServer: req.RealityHandshakeServer,
		RealityHandshakePort:   req.RealityHandshakePort,
		RealityMaxTimeDiff:     req.RealityMaxTimeDiff,
		Enabled:                req.Enabled,
	}
	if err := applyDefaults(&item); err != nil {
		return item, err
	}
	if err := s.populateCredentials(ctx, &item); err != nil {
		return item, err
	}
	if err := validate(&item); err != nil {
		return item, err
	}
	if err := s.db.Create(&item).Error; err != nil {
		return item, err
	}
	_ = s.db.Preload("Domain").First(&item, item.ID).Error
	return item, nil
}

func (s *Service) Update(ctx context.Context, id uint, req CreateRequest) (models.ProxyInbound, error) {
	item, err := s.Get(id)
	if err != nil {
		return item, err
	}
	item.Name = req.Name
	item.Template = req.Template
	item.DomainID = req.DomainID
	item.ListenPort = req.ListenPort
	item.XHTTPPath = req.XHTTPPath
	item.XHTTPMode = req.XHTTPMode
	item.Security = req.Security
	item.RealityHandshakeServer = req.RealityHandshakeServer
	item.RealityHandshakePort = req.RealityHandshakePort
	item.RealityMaxTimeDiff = req.RealityMaxTimeDiff
	item.Enabled = req.Enabled
	if err := applyDefaults(&item); err != nil {
		return item, err
	}
	if item.UUID == "" || item.RealityPrivateKey == "" || item.RealityPublicKey == "" || item.RealityShortID == "" {
		if err := s.populateCredentials(ctx, &item); err != nil {
			return item, err
		}
	}
	if err := validate(&item); err != nil {
		return item, err
	}
	if err := s.db.Save(&item).Error; err != nil {
		return item, err
	}
	_ = s.db.Preload("Domain").First(&item, item.ID).Error
	return item, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.ProxyInbound{}, id).Error
}

func (s *Service) SetEnabled(id uint, enabled bool) error {
	return s.db.Model(&models.ProxyInbound{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *Service) List() ([]models.ProxyInbound, error) {
	var items []models.ProxyInbound
	return items, s.db.Preload("Domain").Order("id desc").Find(&items).Error
}

func (s *Service) Get(id uint) (models.ProxyInbound, error) {
	var item models.ProxyInbound
	return item, s.db.Preload("Domain").First(&item, id).Error
}

func (s *Service) ConfigDetails(id uint) (map[string]any, error) {
	item, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return xray.RenderInbound(toRuntimeInbound(item))
}

func (s *Service) ShareDetails(id uint) (ShareDetails, error) {
	item, err := s.Get(id)
	if err != nil {
		return ShareDetails{}, err
	}
	uri, err := shareURI(item)
	if err != nil {
		return ShareDetails{}, err
	}
	return ShareDetails{
		Name:     item.Name,
		Domain:   item.Domain.Domain,
		Template: item.Template,
		URI:      uri,
	}, nil
}

func (s *Service) populateCredentials(ctx context.Context, item *models.ProxyInbound) error {
	id, err := s.generator.UUID(ctx)
	if err != nil {
		return err
	}
	privateKey, publicKey, err := s.generator.RealityKeyPair(ctx)
	if err != nil {
		return err
	}
	shortID, err := s.generator.ShortID()
	if err != nil {
		return err
	}
	item.UUID = id
	item.RealityPrivateKey = privateKey
	item.RealityPublicKey = publicKey
	item.RealityShortID = shortID
	return nil
}

func applyDefaults(item *models.ProxyInbound) error {
	if item.Template == "" {
		item.Template = "vless-reality-vision"
	}
	if item.Name == "" {
		item.Name = "VLESS Reality Vision"
	}
	item.Protocol = "vless"
	item.ListenAddr = "127.0.0.1"
	if item.RealityMaxTimeDiff == 0 {
		item.RealityMaxTimeDiff = 60
	}
	if item.RealityHandshakePort == 0 {
		item.RealityHandshakePort = 443
	}
	switch item.Template {
	case "vless-reality-vision":
		item.Network = "raw"
		item.Security = "reality"
		item.Flow = "xtls-rprx-vision"
		item.XHTTPPath = ""
		item.XHTTPMode = ""
	case "vless-xhttp":
		item.Network = "xhttp"
		item.Flow = ""
		if item.Security == "" {
			item.Security = "reality"
		}
		if item.Security != "reality" && item.Security != "tls" {
			return fmt.Errorf("unsupported xhttp security %q", item.Security)
		}
		if item.XHTTPPath == "" {
			item.XHTTPPath = "/xhttp"
		}
		if item.XHTTPMode == "" {
			item.XHTTPMode = "auto"
		}
	default:
		return fmt.Errorf("unsupported inbound template %q", item.Template)
	}
	return nil
}

func validate(item *models.ProxyInbound) error {
	if item.DomainID == 0 {
		return errors.New("domainId required")
	}
	if item.ListenAddr != "127.0.0.1" {
		return errors.New("listenAddr must be 127.0.0.1")
	}
	if item.ListenPort <= 0 {
		return errors.New("listenPort required")
	}
	return nil
}

func toRuntimeInbound(item models.ProxyInbound) runtimeconfig.ProxyInbound {
	return runtimeconfig.ProxyInbound{
		ID:                     item.ID,
		Name:                   item.Name,
		Template:               item.Template,
		Protocol:               item.Protocol,
		Domain:                 item.Domain.Domain,
		ListenAddr:             item.ListenAddr,
		ListenPort:             item.ListenPort,
		UUID:                   item.UUID,
		Network:                item.Network,
		Security:               item.Security,
		Flow:                   item.Flow,
		XHTTPPath:              item.XHTTPPath,
		XHTTPMode:              item.XHTTPMode,
		RealityPrivateKey:      item.RealityPrivateKey,
		RealityPublicKey:       item.RealityPublicKey,
		RealityShortID:         item.RealityShortID,
		RealityHandshakeServer: item.RealityHandshakeServer,
		RealityHandshakePort:   item.RealityHandshakePort,
		RealityMaxTimeDiff:     item.RealityMaxTimeDiff,
	}
}

func shareURI(item models.ProxyInbound) (string, error) {
	if item.UUID == "" {
		return "", errors.New("inbound uuid missing")
	}
	if item.Domain.Domain == "" {
		return "", errors.New("inbound domain missing")
	}
	if item.Security == "reality" && (item.RealityPublicKey == "" || item.RealityShortID == "") {
		return "", errors.New("inbound reality credentials missing")
	}

	query := url.Values{}
	query.Set("encryption", "none")
	query.Set("security", item.Security)
	query.Set("type", transportType(item))
	if item.Security == "reality" {
		query.Set("fp", "chrome")
		query.Set("pbk", item.RealityPublicKey)
		query.Set("sid", item.RealityShortID)
		query.Set("sni", shareSNI(item))
	}
	if item.Flow != "" {
		query.Set("flow", item.Flow)
	}
	if item.Template == "vless-xhttp" {
		query.Set("path", item.XHTTPPath)
		query.Set("mode", item.XHTTPMode)
	}

	return (&url.URL{
		Scheme:   "vless",
		User:     url.User(item.UUID),
		Host:     net.JoinHostPort(item.Domain.Domain, strconv.Itoa(443)),
		RawQuery: query.Encode(),
		Fragment: item.Name,
	}).String(), nil
}

func transportType(item models.ProxyInbound) string {
	if item.Template == "vless-reality-vision" {
		return "tcp"
	}
	return item.Network
}

func shareSNI(item models.ProxyInbound) string {
	if item.RealityHandshakeServer != "" {
		return item.RealityHandshakeServer
	}
	return item.Domain.Domain
}
