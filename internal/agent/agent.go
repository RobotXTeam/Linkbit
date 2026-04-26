package agent

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type RegistrationClient interface {
	Register(ctx context.Context, enrollmentKey string, deviceName string) (models.DeviceRegistrationResponse, error)
}

type NetworkConfigClient interface {
	GetNetworkConfig(ctx context.Context) (models.NetworkConfig, error)
}

type TunnelManager interface {
	Apply(ctx context.Context, network models.NetworkConfig) error
	Destroy(ctx context.Context) error
}

type HealthReporter interface {
	CheckAndReport(ctx context.Context, registration models.DeviceRegistrationResponse) error
}

type Service struct {
	cfg          config.AgentConfig
	registration RegistrationClient
	tunnel       TunnelManager
	health       HealthReporter
	state        StateStore
	logger       *slog.Logger
	device       models.DeviceRegistrationResponse
}

func NewService(cfg config.AgentConfig, registration RegistrationClient, tunnel TunnelManager, health HealthReporter, logger *slog.Logger) (*Service, error) {
	if cfg.ControllerURL == "" {
		return nil, errors.New("controller url is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	var state StateStore
	if cfg.StatePath != "" {
		state = NewFileStateStore(cfg.StatePath)
	}
	return &Service{
		cfg:          cfg,
		registration: registration,
		tunnel:       tunnel,
		health:       health,
		state:        state,
		logger:       logger,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	if s.registration != nil {
		if s.state != nil {
			registration, err := s.state.Load()
			if err == nil {
				s.device = registration
				if setter, ok := s.registration.(DeviceCredentialSetter); ok {
					setter.SetDeviceCredentials(registration.Device.ID, registration.Device.DeviceToken)
				}
				s.logger.Info("loaded device state", "device_id", registration.Device.ID, "state_path", s.cfg.StatePath)
			} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNoDeviceCredentials) {
				s.logger.Warn("failed to load agent state, registering again", "err", err, "state_path", s.cfg.StatePath)
			}
		}
		if s.device.Device.ID == "" {
			if s.cfg.EnrollmentKey == "" {
				return errors.New("enrollment key is required for first registration")
			}
			registration, err := s.registration.Register(ctx, s.cfg.EnrollmentKey, s.cfg.DeviceName)
			if err != nil {
				return err
			}
			s.device = registration
			if s.state != nil {
				if err := s.state.Save(registration); err != nil {
					return err
				}
			}
			s.logger.Info("device registered", "device_id", registration.Device.ID, "virtual_ip", registration.Device.VirtualIP)
		}
	}
	if s.tunnel != nil {
		network := models.NetworkConfig{
			Device: s.device.Device,
			Relays: s.device.Relays,
		}
		if client, ok := s.registration.(NetworkConfigClient); ok {
			latest, err := client.GetNetworkConfig(ctx)
			if err != nil {
				return err
			}
			network = latest
		}
		if err := s.tunnel.Apply(ctx, network); err != nil {
			return err
		}
		defer func() {
			if err := s.tunnel.Destroy(context.Background()); err != nil {
				s.logger.Warn("failed to destroy tunnel", "err", err)
			}
		}()
	}

	ticker := time.NewTicker(s.cfg.HealthEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.health != nil {
				if err := s.health.CheckAndReport(ctx, s.device); err != nil {
					s.logger.Warn("health report failed", "err", err)
				}
			}
		}
	}
}
