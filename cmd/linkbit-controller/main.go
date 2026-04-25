package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/controller"
	sqlitestore "github.com/linkbit/linkbit/internal/store/sqlite"
	"github.com/linkbit/linkbit/internal/version"
)

func main() {
	cfg, err := config.LoadController()
	if err != nil {
		log.Fatalf("load controller config: %v", err)
	}

	bootstrapKey := os.Getenv("LINKBIT_BOOTSTRAP_API_KEY")
	if bootstrapKey == "" {
		log.Fatal("LINKBIT_BOOTSTRAP_API_KEY is required")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	storage, err := sqlitestore.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer storage.Close()
	if err := storage.Migrate(context.Background()); err != nil {
		log.Fatalf("migrate store: %v", err)
	}

	server, err := controller.NewServer(cfg, logger, bootstrapKey, storage)
	if err != nil {
		log.Fatalf("create controller: %v", err)
	}

	logger.Info("starting linkbit-controller", "addr", cfg.ListenAddr, "version", version.Version, "web_dir", cfg.WebDir)
	if err := http.ListenAndServe(cfg.ListenAddr, server.WithStatic(server.Handler())); err != nil {
		log.Fatalf("controller stopped: %v", err)
	}
}
