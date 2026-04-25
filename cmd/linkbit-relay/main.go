package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/relay"
	"github.com/linkbit/linkbit/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	node, err := relay.NewNode(config.LoadRelay(), nil, logger)
	if err != nil {
		log.Fatalf("create relay: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting linkbit-relay", "version", version.Version)
	if err := node.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("relay stopped: %v", err)
	}
}
