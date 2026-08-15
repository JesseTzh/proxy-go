package wireguard

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/proxy-go/proxy-go/internal/models"
)

func RenderServer(server models.WireGuardServer, clients []models.WireGuardClient) (string, error) {
	prefix, err := netip.ParsePrefix(server.Address)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}
	network := prefix.Masked().String()
	var b strings.Builder
	fmt.Fprintf(&b, "[Interface]\nAddress = %s\nListenPort = %d\nPrivateKey = %s\n", server.Address, server.ListenPort, server.PrivateKey)
	if server.MTU > 0 {
		fmt.Fprintf(&b, "MTU = %d\n", server.MTU)
	}
	fmt.Fprintf(&b, "PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -A FORWARD -o %%i -j ACCEPT; iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE\n", network, server.EgressInterface)
	fmt.Fprintf(&b, "PostDown = iptables -D FORWARD -i %%i -j ACCEPT; iptables -D FORWARD -o %%i -j ACCEPT; iptables -t nat -D POSTROUTING -s %s -o %s -j MASQUERADE\n", network, server.EgressInterface)
	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		fmt.Fprintf(&b, "\n# %s\n[Peer]\nPublicKey = %s\nPresharedKey = %s\nAllowedIPs = %s\n", sanitizeComment(client.Name), client.PublicKey, client.PresharedKey, client.Address)
	}
	return b.String(), nil
}

func RenderClient(server models.WireGuardServer, client models.WireGuardClient) (string, error) {
	if server.Domain == nil || strings.TrimSpace(server.Domain.Domain) == "" {
		return "", fmt.Errorf("wireguard endpoint domain is not configured")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Interface]\nPrivateKey = %s\nAddress = %s\n", client.PrivateKey, client.Address)
	if server.DNS != "" {
		fmt.Fprintf(&b, "DNS = %s\n", server.DNS)
	}
	if server.MTU > 0 {
		fmt.Fprintf(&b, "MTU = %d\n", server.MTU)
	}
	fmt.Fprintf(&b, "\n[Peer]\nPublicKey = %s\nPresharedKey = %s\nAllowedIPs = 0.0.0.0/0\nEndpoint = %s:%d\nPersistentKeepalive = 25\n", server.PublicKey, client.PresharedKey, server.Domain.Domain, server.ListenPort)
	return b.String(), nil
}

func sanitizeComment(value string) string {
	return strings.NewReplacer("\n", " ", "\r", " ").Replace(value)
}
