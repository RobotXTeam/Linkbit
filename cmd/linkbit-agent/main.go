package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/linkbit/linkbit/internal/agent"
	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.LoadAgent()
	flag.StringVar(&cfg.ControllerURL, "controller", cfg.ControllerURL, "Linkbit controller URL")
	flag.StringVar(&cfg.EnrollmentKey, "enrollment-key", cfg.EnrollmentKey, "one-time enrollment key for first registration")
	flag.StringVar(&cfg.DeviceName, "name", cfg.DeviceName, "device name")
	flag.StringVar(&cfg.StatePath, "state", cfg.StatePath, "agent state file path")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "optional WireGuard endpoint advertised to peers, host:port")
	flag.StringVar(&cfg.WireGuardInterface, "interface", cfg.WireGuardInterface, "WireGuard interface name")
	flag.BoolVar(&cfg.WireGuardDryRun, "dry-run", cfg.WireGuardDryRun, "validate controller flow without changing WireGuard interfaces")
	flag.BoolVar(&cfg.RunOnce, "once", cfg.RunOnce, "register, apply config, report once, then exit")
	flag.Parse()

	identity, err := agent.EnsureIdentity(cfg.StatePath, cfg.WireGuardPrivateKey, cfg.WireGuardPublicKey, cfg.DeviceFingerprint)
	if err != nil {
		log.Fatalf("load agent identity: %v", err)
	}
	cfg.WireGuardPrivateKey = identity.PrivateKey
	cfg.WireGuardPublicKey = identity.PublicKey
	registration := agent.NewHTTPRegistrationClient(cfg.ControllerURL, identity.PublicKey, identity.Fingerprint, cfg.Endpoint)
	tunnel := agent.NewWireGuardManager(cfg, nil)
	health := agent.NewControllerHealthReporter(registration, cfg.Endpoint)
	service, err := agent.NewService(cfg, registration, tunnel, health, logger)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting linkbit-agent", "version", version.Version)
	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("agent stopped: %v", err)
	}
}
