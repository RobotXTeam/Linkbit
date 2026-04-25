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

	err := manager.Apply(t.Context(), models.DeviceRegistrationResponse{
		Device: models.Device{VirtualIP: "100.96.1.2"},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ip link add dev linkbit0 type wireguard",
		"ip address add 100.96.1.2/32 dev linkbit0",
		"ip link set dev linkbit0 mtu 1280",
		"ip link set up dev linkbit0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}
