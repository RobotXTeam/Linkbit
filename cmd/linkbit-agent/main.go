package main

import (
	"context"
	"flag"
	"fmt"
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
	if len(os.Args) > 1 && os.Args[1] == "forward" {
		runForward(logger, os.Args[2:])
		return
	}
	cfg := config.LoadAgent()
	flag.StringVar(&cfg.ControllerURL, "controller", cfg.ControllerURL, "Linkbit controller URL")
	flag.StringVar(&cfg.EnrollmentKey, "enrollment-key", cfg.EnrollmentKey, "one-time enrollment key for first registration")
	flag.StringVar(&cfg.DeviceName, "name", cfg.DeviceName, "device name")
	flag.StringVar(&cfg.StatePath, "state", cfg.StatePath, "agent state file path")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "optional WireGuard endpoint advertised to peers, host:port")
	flag.StringVar(&cfg.WireGuardInterface, "interface", cfg.WireGuardInterface, "WireGuard interface name")
	flag.BoolVar(&cfg.RelayEnabled, "tcp-relay", cfg.RelayEnabled, "poll controller for TCP relay fallback sessions")
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

func runForward(logger *slog.Logger, args []string) {
	cfg := config.LoadAgent()
	fs := flag.NewFlagSet("forward", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:10022", "local TCP listen address")
	target := fs.String("target", "", "target device name, ID, or virtual IP plus port, for example friendlywrt:22")
	fs.StringVar(&cfg.ControllerURL, "controller", cfg.ControllerURL, "Linkbit controller URL")
	fs.StringVar(&cfg.StatePath, "state", cfg.StatePath, "agent state file path")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
	targetHost, targetPort, err := agent.ParseRelayTarget(*target)
	if err != nil {
		log.Fatalf("parse target: %v", err)
	}
	state, err := agent.NewFileStateStore(cfg.StatePath).Load()
	if err != nil {
		log.Fatalf("load agent state: %v", err)
	}
	client := agent.NewHTTPRegistrationClient(cfg.ControllerURL, "", "", "")
	client.SetDeviceCredentials(state.Device.ID, state.Device.DeviceToken)
	forwarder := agent.NewTCPForwarder(client, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger.Info("starting linkbit tcp forward", "listen", *listen, "target", fmt.Sprintf("%s:%d", targetHost, targetPort))
	if err := forwarder.ListenAndServe(ctx, *listen, targetHost, targetPort, state.Device.ID); err != nil && ctx.Err() == nil {
		log.Fatalf("forward stopped: %v", err)
	}
}
