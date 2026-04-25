package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

type DERPService interface {
	Start(context.Context) error
}

type Node struct {
	cfg    config.RelayConfig
	client *http.Client
	logger *slog.Logger
	derp   DERPService
}

func NewNode(cfg config.RelayConfig, derp DERPService, logger *slog.Logger) (*Node, error) {
	if cfg.ControllerURL == "" || cfg.APIKey == "" || cfg.RelayID == "" || cfg.PublicURL == "" {
		return nil, errors.New("controller url, api key, relay id, and public url are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Node{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
		derp:   derp,
	}, nil
}

func (n *Node) Run(ctx context.Context) error {
	if err := n.register(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	if n.derp != nil {
		go func() { errCh <- n.derp.Start(ctx) }()
	}

	ticker := time.NewTicker(n.cfg.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-ticker.C:
			if err := n.heartbeat(ctx); err != nil {
				n.logger.Warn("relay heartbeat failed", "err", err)
			}
		}
	}
}

func (n *Node) register(ctx context.Context) error {
	payload := models.RelayNode{
		ID:        n.cfg.RelayID,
		Name:      n.cfg.Name,
		Region:    n.cfg.Region,
		PublicURL: n.cfg.PublicURL,
	}
	return n.post(ctx, "/api/v1/relays/register", payload)
}

func (n *Node) heartbeat(ctx context.Context) error {
	payload := models.RelayNode{
		ID:     n.cfg.RelayID,
		Status: models.RelayStatusHealthy,
	}
	return n.post(ctx, "/api/v1/relays/heartbeat", payload)
}

func (n *Node) post(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.ControllerURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(linkbitapi.HeaderAPIKey, n.cfg.APIKey)
	req.Header.Set(linkbitapi.HeaderRelayID, n.cfg.RelayID)

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}
	return nil
}
