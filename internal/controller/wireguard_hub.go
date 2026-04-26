package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type WireGuardHub struct {
	cfg       config.ControllerConfig
	logger    *slog.Logger
	publicKey string
	endpoint  string
}

func NewWireGuardHub(cfg config.ControllerConfig, logger *slog.Logger) (*WireGuardHub, error) {
	if !cfg.HubWireGuardEnabled {
		return nil, nil
	}
	if strings.TrimSpace(cfg.HubWireGuardKey) == "" {
		return nil, errors.New("LINKBIT_HUB_WG_PRIVATE_KEY is required when LINKBIT_HUB_WG_ENABLED=true")
	}
	if cfg.HubWireGuardPort < 1 || cfg.HubWireGuardPort > 65535 {
		return nil, errors.New("LINKBIT_HUB_WG_PORT must be between 1 and 65535")
	}
	endpoint := strings.TrimSpace(cfg.HubWireGuardEndpoint)
	if endpoint == "" {
		endpoint = defaultHubEndpoint(cfg.PublicURL, cfg.HubWireGuardPort)
	}
	if !validWireGuardEndpoint(endpoint) {
		return nil, errors.New("LINKBIT_HUB_WG_ENDPOINT must be host:port")
	}
	publicKey, err := wireGuardPublicKey(cfg.HubWireGuardKey)
	if err != nil {
		return nil, fmt.Errorf("derive hub WireGuard public key: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WireGuardHub{cfg: cfg, logger: logger, publicKey: publicKey, endpoint: endpoint}, nil
}

func (h *WireGuardHub) NetworkPeer() models.NetworkPeer {
	if h == nil {
		return models.NetworkPeer{}
	}
	return models.NetworkPeer{
		ID:         "linkbit-hub",
		Name:       "Linkbit Hub",
		VirtualIP:  h.cfg.HubWireGuardIP,
		PublicKey:  h.publicKey,
		Endpoint:   h.endpoint,
		AllowedIPs: []string{h.cfg.HubWireGuardNetwork},
	}
}

func (h *WireGuardHub) Sync(ctx context.Context, devices []models.Device) error {
	if h == nil {
		return nil
	}
	keyFile, err := writeHubPrivateKey(h.cfg.HubWireGuardKey)
	if err != nil {
		return err
	}
	defer os.Remove(keyFile)

	_ = runCommand(ctx, "ip", "link", "del", h.cfg.HubWireGuardIface)
	if err := runCommand(ctx, "ip", "link", "add", "dev", h.cfg.HubWireGuardIface, "type", "wireguard"); err != nil {
		return err
	}
	if err := runCommand(ctx, "wg", "set", h.cfg.HubWireGuardIface, "private-key", keyFile, "listen-port", strconv.Itoa(h.cfg.HubWireGuardPort)); err != nil {
		return err
	}
	if err := runCommand(ctx, "ip", "address", "replace", h.cfg.HubWireGuardIP+"/16", "dev", h.cfg.HubWireGuardIface); err != nil {
		return err
	}
	if err := runCommand(ctx, "ip", "link", "set", "dev", h.cfg.HubWireGuardIface, "mtu", "1280"); err != nil {
		return err
	}
	if err := runCommand(ctx, "ip", "link", "set", "up", "dev", h.cfg.HubWireGuardIface); err != nil {
		return err
	}
	_ = runCommand(ctx, "sysctl", "-w", "net.ipv4.ip_forward=1")

	for _, device := range devices {
		if device.PublicKey == "" || device.VirtualIP == "" {
			continue
		}
		if !h.deviceInHubNetwork(device.VirtualIP) {
			continue
		}
		if err := runCommand(ctx, "wg", "set", h.cfg.HubWireGuardIface, "peer", device.PublicKey, "allowed-ips", device.VirtualIP+"/32"); err != nil {
			h.logger.Warn("skip invalid hub peer", "device_id", device.ID, "device_name", device.Name, "err", err)
			continue
		}
	}
	h.logger.Info("wireguard hub synced", "interface", h.cfg.HubWireGuardIface, "peers", len(devices), "endpoint", h.endpoint)
	return nil
}

func (h *WireGuardHub) deviceInHubNetwork(virtualIP string) bool {
	ip := net.ParseIP(strings.TrimSpace(virtualIP))
	if ip == nil {
		return false
	}
	_, network, err := net.ParseCIDR(h.cfg.HubWireGuardNetwork)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

func (h *WireGuardHub) SyncFromStore(ctx context.Context, listDevices func(context.Context) ([]models.Device, error)) {
	if h == nil {
		return
	}
	devices, err := listDevices(ctx)
	if err != nil {
		h.logger.Warn("list devices for hub sync failed", "err", err)
		return
	}
	if err := h.Sync(ctx, devices); err != nil {
		h.logger.Warn("wireguard hub sync failed", "err", err)
	}
}

func defaultHubEndpoint(publicURL string, port int) string {
	host := strings.TrimSpace(publicURL)
	if parsed, err := url.Parse(publicURL); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	if ip := net.ParseIP(host); ip != nil && strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if host == "" {
		return ""
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func wireGuardPublicKey(privateKey string) (string, error) {
	cmd := exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(strings.TrimSpace(privateKey) + "\n")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func writeHubPrivateKey(privateKey string) (string, error) {
	file, err := os.CreateTemp("", "linkbit-hub-wg-key-*")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if _, err := file.WriteString(strings.TrimSpace(privateKey) + "\n"); err != nil {
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

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
