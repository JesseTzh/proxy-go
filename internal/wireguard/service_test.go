package wireguard

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestEnsureServerCreatesWireGuardKeyPair(t *testing.T) {
	svc := New(testutil.NewDB(t), testutil.NewConfig(t))
	server, err := svc.EnsureServer()
	if err != nil {
		t.Fatalf("ensure server: %v", err)
	}
	for name, value := range map[string]string{"private": server.PrivateKey, "public": server.PublicKey} {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(decoded) != 32 {
			t.Fatalf("invalid %s key %q: %v", name, value, err)
		}
	}
	if server.Address != "10.8.0.1/24" || server.ListenPort != 51820 || server.Enabled {
		t.Fatalf("unexpected defaults: %#v", server)
	}
}

func TestParseDumpReturnsPeerTrafficAndHandshake(t *testing.T) {
	dump := "server-private\tserver-public\t51820\toff\nclient-public\tpsk\t198.51.100.8:54321\t10.8.0.2/32\t1700000000\t1234\t5678\t25\n"
	peers, err := parseDump(dump)
	if err != nil {
		t.Fatalf("parse dump: %v", err)
	}
	peer := peers["client-public"]
	if peer.Endpoint != "198.51.100.8:54321" || peer.ReceiveBytes != 1234 || peer.TransmitBytes != 5678 {
		t.Fatalf("unexpected peer runtime: %#v", peer)
	}
	if peer.LastHandshakeAt == nil || !peer.LastHandshakeAt.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("unexpected handshake: %v", peer.LastHandshakeAt)
	}
}

func TestParseDumpTreatsZeroHandshakeAsNeverConnected(t *testing.T) {
	dump := "server-private\tserver-public\t51820\toff\nclient-public\t(none)\t(none)\t10.8.0.2/32\t0\t0\t0\toff\n"
	peers, err := parseDump(dump)
	if err != nil {
		t.Fatalf("parse dump: %v", err)
	}
	if peers["client-public"].LastHandshakeAt != nil {
		t.Fatalf("expected no handshake, got %v", peers["client-public"].LastHandshakeAt)
	}
}

func TestCreateClientsAllocatesAddressesAndRendersConfig(t *testing.T) {
	db := testutil.NewDB(t)
	svc := New(db, testutil.NewConfig(t))
	domain := models.Domain{Domain: "vpn.example.com", Status: "enabled"}
	if err := db.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if _, err := svc.SetDomain(context.Background(), domain.ID); err != nil {
		t.Fatalf("set domain: %v", err)
	}
	first, err := svc.CreateClient(context.Background(), "MacBook")
	if err != nil {
		t.Fatalf("create first client: %v", err)
	}
	second, err := svc.CreateClient(context.Background(), "iPhone")
	if err != nil {
		t.Fatalf("create second client: %v", err)
	}
	if first.Address != "10.8.0.2/32" || second.Address != "10.8.0.3/32" {
		t.Fatalf("unexpected client addresses: %q %q", first.Address, second.Address)
	}
	_, content, err := svc.ClientConfig(first.ID)
	if err != nil {
		t.Fatalf("render client config: %v", err)
	}
	for _, expected := range []string{"Address = 10.8.0.2/32", "Endpoint = vpn.example.com:51820", "AllowedIPs = 0.0.0.0/0", "PersistentKeepalive = 25"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("missing %q in client config:\n%s", expected, content)
		}
	}
}

func TestRenderServerIncludesOnlyEnabledClientsAndNAT(t *testing.T) {
	server := models.WireGuardServer{Address: "10.8.0.1/24", ListenPort: 51820, PrivateKey: "server-private", MTU: 1420, EgressInterface: "eth0"}
	content, err := RenderServer(server, []models.WireGuardClient{
		{Name: "enabled", Address: "10.8.0.2/32", PublicKey: "public-1", PresharedKey: "psk-1", Enabled: true},
		{Name: "disabled", Address: "10.8.0.3/32", PublicKey: "public-2", PresharedKey: "psk-2", Enabled: false},
	})
	if err != nil {
		t.Fatalf("render server: %v", err)
	}
	for _, expected := range []string{"ListenPort = 51820", "-s 10.8.0.0/24 -o eth0 -j MASQUERADE", "PublicKey = public-1"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("missing %q in server config:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "public-2") {
		t.Fatalf("disabled client was rendered:\n%s", content)
	}
}

func TestClientConfigRequiresEndpointDomain(t *testing.T) {
	svc := New(testutil.NewDB(t), testutil.NewConfig(t))
	client, err := svc.CreateClient(context.Background(), "Laptop")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, _, err := svc.ClientConfig(client.ID); err == nil || !strings.Contains(err.Error(), "endpoint domain") {
		t.Fatalf("expected endpoint domain error, got %v", err)
	}
}

func TestUpdateRejectsEnableWithoutDomainAndIncompatibleNetwork(t *testing.T) {
	svc := New(testutil.NewDB(t), testutil.NewConfig(t))
	_, err := svc.Update(context.Background(), UpdateRequest{Enabled: true, Address: "10.8.0.1/24", ListenPort: 51820, DNS: "1.1.1.1", MTU: 1420, EgressInterface: "eth0"})
	if err == nil || !strings.Contains(err.Error(), "endpoint domain") {
		t.Fatalf("expected endpoint domain validation error, got %v", err)
	}
	if _, err := svc.CreateClient(context.Background(), "Laptop"); err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = svc.Update(context.Background(), UpdateRequest{Address: "10.9.0.1/24", ListenPort: 51820, DNS: "1.1.1.1", MTU: 1420, EgressInterface: "eth0"})
	if err == nil || !strings.Contains(err.Error(), "existing client address") {
		t.Fatalf("expected incompatible network error, got %v", err)
	}
}
