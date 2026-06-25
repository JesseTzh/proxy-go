package inbounds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/runtimeconfig"
	"github.com/proxy-go/proxy-go/internal/singbox"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	cfg       *config.Config
	generator singbox.CredentialGenerator
}

const (
	defaultListenAddr = "127.0.0.1"
	defaultListenPort = 31001
)

type CreateRequest struct {
	Template               string `json:"template"`
	Name                   string `json:"name"`
	DomainID               uint   `json:"domainId"`
	RealityHandshakeServer string `json:"realityHandshakeServer"`
}

type ShareDetails struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	URI    string `json:"uri"`
}

func New(db *gorm.DB, cfg *config.Config, generator singbox.CredentialGenerator) *Service {
	return &Service{db: db, cfg: cfg, generator: generator}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (models.ProxyInbound, error) {
	item := models.ProxyInbound{
		Template:               req.Template,
		Name:                   req.Name,
		DomainID:               req.DomainID,
		RealityHandshakeServer: req.RealityHandshakeServer,
		Enabled:                true,
	}
	if err := s.applyDefaults(&item); err != nil {
		return item, err
	}
	if err := s.assignListenPort(&item); err != nil {
		return item, err
	}
	if err := s.populateCredentials(ctx, &item); err != nil {
		return item, err
	}
	if err := validate(&item); err != nil {
		return item, err
	}
	if err := s.validateStreamSNI(&item); err != nil {
		return item, err
	}
	if err := s.validateRouteSNIUniqueness(&item); err != nil {
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
	item.Template = req.Template
	item.Name = req.Name
	item.DomainID = req.DomainID
	item.Domain = models.Domain{}
	item.RouteSNI = ""
	item.RealityHandshakeServer = req.RealityHandshakeServer
	if err := s.applyDefaults(&item); err != nil {
		return item, err
	}
	if err := s.populateMissingCredentials(ctx, &item); err != nil {
		return item, err
	}
	if err := validate(&item); err != nil {
		return item, err
	}
	if err := s.validateStreamSNI(&item); err != nil {
		return item, err
	}
	if err := s.validateRouteSNIUniqueness(&item); err != nil {
		return item, err
	}
	if err := s.db.Save(&item).Error; err != nil {
		return item, err
	}
	_ = s.db.Preload("Domain").First(&item, item.ID).Error
	return item, nil
}

func (s *Service) populateMissingCredentials(ctx context.Context, item *models.ProxyInbound) error {
	switch item.Template {
	case "vless-reality-vision":
		if item.UUID == "" || item.RealityPrivateKey == "" || item.RealityPublicKey == "" || item.RealityShortID == "" {
			return s.populateCredentials(ctx, item)
		}
	case "anytls":
		if item.Password == "" {
			password, err := s.generator.Password()
			if err != nil {
				return err
			}
			item.Password = password
		}
	}
	return nil
}

func (s *Service) SetEnabled(id uint, enabled bool) error {
	if !enabled {
		return s.db.Model(&models.ProxyInbound{}).Where("id = ?", id).Update("enabled", false).Error
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Enabled = true
	if err := s.applyDefaults(&item); err != nil {
		return err
	}
	if err := validate(&item); err != nil {
		return err
	}
	if err := s.validateStreamSNI(&item); err != nil {
		return err
	}
	if err := s.validateRouteSNIUniqueness(&item); err != nil {
		return err
	}
	return s.db.Model(&models.ProxyInbound{}).Where("id = ?", id).Updates(map[string]any{
		"enabled":   true,
		"route_sni": item.RouteSNI,
	}).Error
}

func (s *Service) populateCredentials(ctx context.Context, item *models.ProxyInbound) error {
	switch item.Template {
	case "vless-reality-vision":
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
	case "anytls":
		password, err := s.generator.Password()
		if err != nil {
			return err
		}
		item.Password = password
	}
	return nil
}

func (s *Service) assignListenPort(item *models.ProxyInbound) error {
	if item.ListenPort != 0 {
		return nil
	}
	var maxPort int
	row := s.db.Model(&models.ProxyInbound{}).Select("COALESCE(MAX(listen_port), ?)", defaultListenPort-1).Row()
	if err := row.Scan(&maxPort); err != nil {
		return err
	}
	item.ListenPort = maxPort + 1
	if item.ListenPort < defaultListenPort {
		item.ListenPort = defaultListenPort
	}
	return nil
}

func (s *Service) domainName(id uint) (string, error) {
	var domain models.Domain
	if err := s.db.First(&domain, id).Error; err != nil {
		return "", err
	}
	return normalizeDNSName(domain.Domain), nil
}

func (s *Service) applyDefaults(item *models.ProxyInbound) error {
	if item.Template == "" {
		item.Template = "vless-reality-vision"
	}
	if item.ListenAddr == "" {
		item.ListenAddr = defaultListenAddr
	}
	item.Network = "tcp"
	switch item.Template {
	case "vless-reality-vision":
		if item.Name == "" {
			item.Name = "VLESS Reality Vision"
		}
		item.Protocol = "vless"
		item.Security = "reality"
		item.Flow = "xtls-rprx-vision"
		if item.RealityMaxTimeDiff == 0 {
			item.RealityMaxTimeDiff = 60000
		}
		if item.RealityHandshakePort == 0 {
			item.RealityHandshakePort = 443
		}
		item.RealityHandshakeServer = normalizeDNSName(item.RealityHandshakeServer)
		item.RouteSNI = item.RealityHandshakeServer
	case "anytls":
		if item.Name == "" {
			item.Name = "AnyTLS"
		}
		item.Protocol = "anytls"
		item.Security = "tls"
		item.Flow = ""
		item.RealityHandshakeServer = ""
		item.RealityHandshakePort = 0
		item.RealityMaxTimeDiff = 0
		if item.RouteSNI == "" {
			domain, err := s.domainName(item.DomainID)
			if err != nil {
				return err
			}
			item.RouteSNI = domain
		}
		item.RouteSNI = normalizeDNSName(item.RouteSNI)
	default:
		return fmt.Errorf("unsupported inbound template %q", item.Template)
	}
	return nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.ProxyInbound{}, id).Error
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
	return singbox.RenderInbound(s.toRuntimeInbound(item), s.cfg.Paths.CertDir)
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
		Name:   item.Name,
		Domain: item.Domain.Domain,
		URI:    uri,
	}, nil
}

func validate(item *models.ProxyInbound) error {
	if item.DomainID == 0 {
		return errors.New("domainId required")
	}
	if item.ListenAddr != defaultListenAddr {
		return fmt.Errorf("listenAddr must be %s", defaultListenAddr)
	}
	if item.ListenPort < defaultListenPort {
		return fmt.Errorf("listenPort must be >= %d", defaultListenPort)
	}
	if item.Security == "reality" && item.RealityHandshakeServer == "" {
		return errors.New("realityHandshakeServer required")
	}
	if item.Security == "reality" && !isDNSName(item.RealityHandshakeServer) {
		return errors.New("realityHandshakeServer must be a valid domain name")
	}
	if item.Security == "reality" && item.RealityHandshakePort != 443 {
		return errors.New("realityHandshakePort must be 443")
	}
	if item.RouteSNI == "" {
		return errors.New("routeSni required")
	}
	if !isDNSName(item.RouteSNI) {
		return errors.New("routeSni must be a valid domain name")
	}
	if item.Template == "anytls" && item.Password == "" {
		return errors.New("anytls password required")
	}
	return nil
}

func isDNSName(name string) bool {
	name = normalizeDNSName(name)
	if len(name) == 0 || len(name) > 253 || !strings.Contains(name, ".") {
		return false
	}
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

func (s *Service) validateStreamSNI(item *models.ProxyInbound) error {
	if item.Security != "reality" || item.RealityHandshakeServer == "" {
		return nil
	}
	handshakeServer := normalizeDNSName(item.RealityHandshakeServer)
	var count int64
	if err := s.db.Model(&models.Domain{}).Where("lower(domain) = ?", handshakeServer).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("realityHandshakeServer must not be a managed domain")
	}
	var setting models.SystemSetting
	if err := s.db.First(&setting, 1).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if normalizeDNSName(setting.ManagementDomain) == handshakeServer {
		return errors.New("realityHandshakeServer must not be the management domain")
	}
	return nil
}

func (s *Service) validateRouteSNIUniqueness(item *models.ProxyInbound) error {
	if !item.Enabled || item.RouteSNI == "" {
		return nil
	}
	var count int64
	routeSNI := normalizeDNSName(item.RouteSNI)
	query := s.db.Model(&models.ProxyInbound{}).
		Where("lower(route_sni) = ? AND enabled = ?", routeSNI, true)
	if item.ID != 0 {
		query = query.Where("id <> ?", item.ID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("routeSni must be unique among enabled inbounds")
	}
	return nil
}

func (s *Service) toRuntimeInbound(item models.ProxyInbound) runtimeconfig.ProxyInbound {
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
		RouteSNI:               item.RouteSNI,
		Password:               item.Password,
		RealityPrivateKey:      item.RealityPrivateKey,
		RealityPublicKey:       item.RealityPublicKey,
		RealityShortID:         item.RealityShortID,
		RealityHandshakeServer: item.RealityHandshakeServer,
		RealityHandshakePort:   item.RealityHandshakePort,
		RealityMaxTimeDiff:     item.RealityMaxTimeDiff,
	}
}

func shareURI(item models.ProxyInbound) (string, error) {
	if item.Domain.Domain == "" {
		return "", errors.New("inbound domain missing")
	}
	switch item.Template {
	case "vless-reality-vision":
		return shareVLESSRealityVisionURI(item)
	case "anytls":
		return shareAnyTLSURI(item)
	default:
		return "", fmt.Errorf("unsupported inbound template %q", item.Template)
	}
}

func shareVLESSRealityVisionURI(item models.ProxyInbound) (string, error) {
	if item.UUID == "" {
		return "", errors.New("inbound uuid missing")
	}
	if item.RealityPublicKey == "" || item.RealityShortID == "" {
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
	return (&url.URL{
		Scheme:   "vless",
		User:     url.User(item.UUID),
		Host:     net.JoinHostPort(item.Domain.Domain, strconv.Itoa(443)),
		RawQuery: query.Encode(),
		Fragment: item.Name,
	}).String(), nil
}

func shareAnyTLSURI(item models.ProxyInbound) (string, error) {
	if item.Password == "" {
		return "", errors.New("inbound anytls password missing")
	}
	query := url.Values{}
	query.Set("security", item.Security)
	query.Set("sni", shareSNI(item))
	return (&url.URL{
		Scheme:   "anytls",
		User:     url.User(item.Password),
		Host:     net.JoinHostPort(item.Domain.Domain, strconv.Itoa(443)),
		RawQuery: query.Encode(),
		Fragment: item.Name,
	}).String(), nil
}

func transportType(item models.ProxyInbound) string {
	return item.Network
}

func shareSNI(item models.ProxyInbound) string {
	if item.RouteSNI != "" {
		return item.RouteSNI
	}
	return item.RealityHandshakeServer
}
