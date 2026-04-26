package main

import (
	"context"
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
	identity, err := agent.EnsureIdentity(cfg.StatePath, cfg.WireGuardPrivateKey, cfg.WireGuardPublicKey, os.Getenv("LINKBIT_DEVICE_FINGERPRINT"))
	if err != nil {
		log.Fatalf("load agent identity: %v", err)
	}
	cfg.WireGuardPrivateKey = identity.PrivateKey
	cfg.WireGuardPublicKey = identity.PublicKey
	registration := agent.NewHTTPRegistrationClient(cfg.ControllerURL, identity.PublicKey, identity.Fingerprint)
	tunnel := agent.NewWireGuardManager(cfg, nil)
	health := agent.NewControllerHealthReporter(registration)
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
