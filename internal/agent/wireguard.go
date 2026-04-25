package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type WireGuardManager struct {
	cfg    config.AgentConfig
	runner CommandRunner
}

func NewWireGuardManager(cfg config.AgentConfig, runner CommandRunner) *WireGuardManager {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &WireGuardManager{cfg: cfg, runner: runner}
}

func (m *WireGuardManager) Apply(ctx context.Context, registration models.DeviceRegistrationResponse) error {
	if m.cfg.WireGuardDryRun {
		return m.applyCommands(ctx, registration, "")
	}
	if runtime.GOOS != "linux" {
		return errors.New("wireguard command manager currently supports linux only")
	}
	if m.cfg.WireGuardPrivateKey == "" {
		return errors.New("LINKBIT_WG_PRIVATE_KEY is required")
	}
	keyFile, err := writePrivateKeyFile(m.cfg.WireGuardPrivateKey)
	if err != nil {
		return err
	}
	defer os.Remove(keyFile)
	return m.applyCommands(ctx, registration, keyFile)
}

func (m *WireGuardManager) Destroy(ctx context.Context) error {
	if m.cfg.WireGuardInterface == "" {
		return errors.New("wireguard interface is required")
	}
	if m.cfg.WireGuardDryRun || runtime.GOOS == "linux" {
		return m.runner.Run(ctx, "ip", "link", "del", m.cfg.WireGuardInterface)
	}
	return nil
}

func (m *WireGuardManager) applyCommands(ctx context.Context, registration models.DeviceRegistrationResponse, keyFile string) error {
	if m.cfg.WireGuardInterface == "" {
		return errors.New("wireguard interface is required")
	}
	if registration.Device.VirtualIP == "" {
		return errors.New("registered device virtual IP is required")
	}
	_ = m.runner.Run(ctx, "ip", "link", "del", m.cfg.WireGuardInterface)
	if err := m.runner.Run(ctx, "ip", "link", "add", "dev", m.cfg.WireGuardInterface, "type", "wireguard"); err != nil {
		return err
	}
	if keyFile != "" {
		if err := m.runner.Run(ctx, "wg", "set", m.cfg.WireGuardInterface, "private-key", keyFile); err != nil {
			return err
		}
	}
	if err := m.runner.Run(ctx, "ip", "address", "add", registration.Device.VirtualIP+"/32", "dev", m.cfg.WireGuardInterface); err != nil {
		return err
	}
	if err := m.runner.Run(ctx, "ip", "link", "set", "dev", m.cfg.WireGuardInterface, "mtu", "1280"); err != nil {
		return err
	}
	return m.runner.Run(ctx, "ip", "link", "set", "up", "dev", m.cfg.WireGuardInterface)
}

func writePrivateKeyFile(privateKey string) (string, error) {
	file, err := os.CreateTemp("", "linkbit-wg-key-*")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if _, err := file.WriteString(privateKey + "\n"); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}
