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
		Device: models.Device{VirtualIP: "100.96.1.2"},
		Peers:  []models.NetworkPeer{{VirtualIP: "100.96.1.3", PublicKey: "peer-public-key"}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ip link add dev linkbit0 type wireguard",
		"ip address add 100.96.1.2/32 dev linkbit0",
		"wg set linkbit0 peer peer-public-key allowed-ips 100.96.1.3/32",
		"ip link set dev linkbit0 mtu 1280",
		"ip link set up dev linkbit0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}
