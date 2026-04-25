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
	registration := agent.NewHTTPRegistrationClient(cfg.ControllerURL, os.Getenv("LINKBIT_WG_PUBLIC_KEY"), os.Getenv("LINKBIT_DEVICE_FINGERPRINT"))
	service, err := agent.NewService(cfg, registration, nil, nil, logger)
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
