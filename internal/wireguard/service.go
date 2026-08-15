package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

const serverID = 1

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

type UpdateRequest struct {
	Enabled         bool   `json:"enabled"`
	Address         string `json:"address"`
	ListenPort      int    `json:"listenPort"`
	DNS             string `json:"dns"`
	MTU             int    `json:"mtu"`
	EgressInterface string `json:"egressInterface"`
}

type State struct {
	Server  models.WireGuardServer   `json:"server"`
	Clients []models.WireGuardClient `json:"clients"`
	Runtime map[string]any           `json:"runtime"`
}

func New(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) EnsureServer() (models.WireGuardServer, error) {
	var server models.WireGuardServer
	err := s.db.Preload("Domain").First(&server, serverID).Error
	if err == nil {
		return server, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return server, err
	}
	privateKey, publicKey, err := generateKeyPair()
	if err != nil {
		return server, err
	}
	server = models.WireGuardServer{
		ID: serverID, InterfaceName: "wg0", Address: "10.8.0.1/24", ListenPort: 51820,
		DNS: "1.1.1.1", MTU: 1420, EgressInterface: "eth0", PrivateKey: privateKey, PublicKey: publicKey,
	}
	if err := s.db.Create(&server).Error; err != nil {
		return server, err
	}
	return server, nil
}

func (s *Service) State(ctx context.Context) (State, error) {
	server, err := s.EnsureServer()
	if err != nil {
		return State{}, err
	}
	clients, err := s.ListClients()
	if err != nil {
		return State{}, err
	}
	return State{Server: server, Clients: clients, Runtime: s.Status(ctx)}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (models.WireGuardServer, error) {
	server, err := s.EnsureServer()
	if err != nil {
		return server, err
	}
	server.Enabled = req.Enabled
	server.Address = strings.TrimSpace(req.Address)
	server.ListenPort = req.ListenPort
	server.DNS = strings.TrimSpace(req.DNS)
	server.MTU = req.MTU
	server.EgressInterface = strings.TrimSpace(req.EgressInterface)
	if err := validateServer(server); err != nil {
		return server, err
	}
	if server.Enabled && server.DomainID == nil {
		return server, fmt.Errorf("wireguard endpoint domain must be configured before enabling the service")
	}
	if err := s.validateClientAddresses(server.Address); err != nil {
		return server, err
	}
	if err := s.db.Save(&server).Error; err != nil {
		return server, err
	}
	if err := s.Apply(ctx); err != nil {
		return server, err
	}
	return s.EnsureServer()
}

func (s *Service) SetDomain(ctx context.Context, domainID uint) (models.WireGuardServer, error) {
	server, err := s.EnsureServer()
	if err != nil {
		return server, err
	}
	var domain models.Domain
	if err := s.db.First(&domain, domainID).Error; err != nil {
		return server, err
	}
	server.DomainID = &domain.ID
	server.Domain = &domain
	if err := s.db.Save(&server).Error; err != nil {
		return server, err
	}
	if server.Enabled {
		if err := s.Apply(ctx); err != nil {
			return server, err
		}
	}
	return server, nil
}

func (s *Service) CreateClient(ctx context.Context, name string) (models.WireGuardClient, error) {
	server, err := s.EnsureServer()
	if err != nil {
		return models.WireGuardClient{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return models.WireGuardClient{}, fmt.Errorf("client name is required")
	}
	address, err := s.nextClientAddress(server.Address)
	if err != nil {
		return models.WireGuardClient{}, err
	}
	privateKey, publicKey, err := generateKeyPair()
	if err != nil {
		return models.WireGuardClient{}, err
	}
	presharedKey, err := generatePresharedKey()
	if err != nil {
		return models.WireGuardClient{}, err
	}
	client := models.WireGuardClient{ServerID: server.ID, Name: name, Address: address, PrivateKey: privateKey, PublicKey: publicKey, PresharedKey: presharedKey, Enabled: true}
	if err := s.db.Create(&client).Error; err != nil {
		return client, err
	}
	if server.Enabled {
		if err := s.Apply(ctx); err != nil {
			return client, err
		}
	}
	return client, nil
}

func (s *Service) ListClients() ([]models.WireGuardClient, error) {
	var clients []models.WireGuardClient
	return clients, s.db.Where("server_id = ?", serverID).Order("id desc").Find(&clients).Error
}

func (s *Service) SetClientEnabled(ctx context.Context, id uint, enabled bool) error {
	result := s.db.Model(&models.WireGuardClient{}).Where("id = ? AND server_id = ?", id, serverID).Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	server, err := s.EnsureServer()
	if err != nil {
		return err
	}
	if server.Enabled {
		return s.Apply(ctx)
	}
	return nil
}

func (s *Service) DeleteClient(ctx context.Context, id uint) error {
	result := s.db.Where("id = ? AND server_id = ?", id, serverID).Delete(&models.WireGuardClient{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	server, err := s.EnsureServer()
	if err != nil {
		return err
	}
	if server.Enabled {
		return s.Apply(ctx)
	}
	return nil
}

func (s *Service) ClientConfig(id uint) (models.WireGuardClient, string, error) {
	server, err := s.EnsureServer()
	if err != nil {
		return models.WireGuardClient{}, "", err
	}
	var client models.WireGuardClient
	if err := s.db.Where("id = ? AND server_id = ?", id, serverID).First(&client).Error; err != nil {
		return client, "", err
	}
	content, err := RenderClient(server, client)
	return client, content, err
}

func (s *Service) Apply(ctx context.Context) error {
	server, err := s.EnsureServer()
	if err != nil {
		return err
	}
	if !server.Enabled {
		return s.Stop(ctx)
	}
	clients, err := s.ListClients()
	if err != nil {
		return err
	}
	content, err := RenderServer(server, clients)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.cfg.Paths.WireGuardConfDir, 0700); err != nil {
		return err
	}
	path := s.configPath(server.InterfaceName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if s.isRunning(ctx, server.InterfaceName) {
		if output, err := exec.CommandContext(ctx, s.cfg.Runtime.WGQuickBinary, "down", path).CombinedOutput(); err != nil {
			return fmt.Errorf("wireguard down: %w: %s", err, strings.TrimSpace(string(output)))
		}
	}
	output, err := exec.CommandContext(ctx, s.cfg.Runtime.WGQuickBinary, "up", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wireguard up: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	server, err := s.EnsureServer()
	if err != nil {
		return err
	}
	if !s.isRunning(ctx, server.InterfaceName) {
		return nil
	}
	output, err := exec.CommandContext(ctx, s.cfg.Runtime.WGQuickBinary, "down", s.configPath(server.InterfaceName)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wireguard down: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) Status(ctx context.Context) map[string]any {
	server, err := s.EnsureServer()
	if err != nil {
		return map[string]any{"running": false, "error": err.Error()}
	}
	return map[string]any{"running": s.isRunning(ctx, server.InterfaceName), "interface": server.InterfaceName, "configPath": s.configPath(server.InterfaceName)}
}

func (s *Service) isRunning(ctx context.Context, interfaceName string) bool {
	return exec.CommandContext(ctx, s.cfg.Runtime.WGBinary, "show", interfaceName).Run() == nil
}

func (s *Service) configPath(interfaceName string) string {
	return filepath.Join(s.cfg.Paths.WireGuardConfDir, interfaceName+".conf")
}

func (s *Service) nextClientAddress(serverAddress string) (string, error) {
	prefix, err := netip.ParsePrefix(serverAddress)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}
	if !prefix.Addr().Is4() {
		return "", fmt.Errorf("only IPv4 wireguard networks are supported")
	}
	clients, err := s.ListClients()
	if err != nil {
		return "", err
	}
	used := map[netip.Addr]bool{prefix.Addr(): true}
	for _, client := range clients {
		if p, err := netip.ParsePrefix(client.Address); err == nil {
			used[p.Addr()] = true
		}
	}
	for addr := prefix.Masked().Addr().Next(); prefix.Contains(addr); addr = addr.Next() {
		if !used[addr] && addr.Next().IsValid() && prefix.Contains(addr.Next()) {
			return addr.String() + "/32", nil
		}
	}
	return "", fmt.Errorf("wireguard address pool is exhausted")
}

func validateServer(server models.WireGuardServer) error {
	prefix, err := netip.ParsePrefix(server.Address)
	if err != nil || !prefix.Addr().Is4() || prefix.Bits() > 30 {
		return fmt.Errorf("address must be an IPv4 CIDR with at least two client addresses")
	}
	if prefix.Addr() == prefix.Masked().Addr() || !prefix.Contains(prefix.Addr().Next()) {
		return fmt.Errorf("address must use a host address inside the IPv4 network")
	}
	if server.ListenPort < 1 || server.ListenPort > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
	}
	if server.MTU < 576 || server.MTU > 9000 {
		return fmt.Errorf("MTU must be between 576 and 9000")
	}
	if server.EgressInterface == "" || strings.ContainsAny(server.EgressInterface, " ;\t\r\n") {
		return fmt.Errorf("invalid egress interface")
	}
	return nil
}

func (s *Service) validateClientAddresses(serverAddress string) error {
	prefix, err := netip.ParsePrefix(serverAddress)
	if err != nil {
		return err
	}
	clients, err := s.ListClients()
	if err != nil {
		return err
	}
	for _, client := range clients {
		address, err := netip.ParsePrefix(client.Address)
		if err != nil || !prefix.Contains(address.Addr()) {
			return fmt.Errorf("server network does not contain existing client address %s", client.Address)
		}
	}
	return nil
}

func ConfigFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	if name == "" {
		name = "wireguard-client"
	}
	return name + ".conf"
}
