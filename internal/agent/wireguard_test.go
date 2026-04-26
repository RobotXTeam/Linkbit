package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type recordingRunner struct {
	commands []string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	r.commands = append(r.commands, name+" "+strings.Join(args, " "))
	return nil
}

func TestWireGuardManagerDryRun(t *testing.T) {
	runner := &recordingRunner{}
	manager := NewWireGuardManager(config.AgentConfig{
		WireGuardInterface: "linkbit0",
		WireGuardDryRun:    true,
	}, runner)

	err := manager.Apply(t.Context(), models.NetworkConfig{
		Device: models.Device{VirtualIP: "10.88.1.2"},
		Peers:  []models.NetworkPeer{{VirtualIP: "10.88.1.3", PublicKey: "peer-public-key", Endpoint: "198.51.100.10:41641"}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(runner.commands) != 0 {
		t.Fatalf("dry-run executed commands: %v", runner.commands)
	}
	if err := manager.Destroy(t.Context()); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("dry-run destroy executed commands: %v", runner.commands)
	}
}

func TestWireGuardManagerBuildsLinuxCommands(t *testing.T) {
	runner := &recordingRunner{}
	manager := NewWireGuardManager(config.AgentConfig{
		WireGuardInterface:  "linkbit0",
		WireGuardPrivateKey: "private-key",
	}, runner)

	err := manager.Apply(t.Context(), models.NetworkConfig{
		Device: models.Device{VirtualIP: "10.88.1.2"},
		Peers:  []models.NetworkPeer{{VirtualIP: "10.88.1.3", PublicKey: "peer-public-key", Endpoint: "198.51.100.10:41641"}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ip link add dev linkbit0 type wireguard",
		"ip address add 10.88.1.2/32 dev linkbit0",
		"wg set linkbit0 peer peer-public-key allowed-ips 10.88.1.3/32",
		"wg set linkbit0 peer peer-public-key endpoint 198.51.100.10:41641",
		"ip route replace 10.88.1.3/32 dev linkbit0",
		"ip link set dev linkbit0 mtu 1280",
		"ip link set up dev linkbit0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}

func TestWireGuardManagerBuildsHubRoute(t *testing.T) {
	runner := &recordingRunner{}
	manager := NewWireGuardManager(config.AgentConfig{
		WireGuardInterface:  "linkbit0",
		WireGuardPrivateKey: "private-key",
	}, runner)

	err := manager.Apply(t.Context(), models.NetworkConfig{
		Device: models.Device{VirtualIP: "10.88.1.2"},
		Peers: []models.NetworkPeer{{
			VirtualIP:  "10.88.0.1",
			PublicKey:  "hub-public-key",
			Endpoint:   "203.0.113.10:41641",
			AllowedIPs: []string{"10.88.0.0/16"},
		}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"wg set linkbit0 peer hub-public-key allowed-ips 10.88.0.0/16",
		"wg set linkbit0 peer hub-public-key endpoint 203.0.113.10:41641",
		"ip route replace 10.88.0.0/16 dev linkbit0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}
