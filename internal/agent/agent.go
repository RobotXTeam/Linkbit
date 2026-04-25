package agent

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type RegistrationClient interface {
	Register(ctx context.Context, enrollmentKey string, deviceName string) (models.DeviceRegistrationResponse, error)
}

type TunnelManager interface {
	Apply(ctx context.Context, registration models.DeviceRegistrationResponse) error
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
	logger       *slog.Logger
	device       models.DeviceRegistrationResponse
}

func NewService(cfg config.AgentConfig, registration RegistrationClient, tunnel TunnelManager, health HealthReporter, logger *slog.Logger) (*Service, error) {
	if cfg.ControllerURL == "" || cfg.EnrollmentKey == "" {
		return nil, errors.New("controller url and enrollment key are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		cfg:          cfg,
		registration: registration,
		tunnel:       tunnel,
		health:       health,
		logger:       logger,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	if s.registration != nil {
		registration, err := s.registration.Register(ctx, s.cfg.EnrollmentKey, s.cfg.DeviceName)
		if err != nil {
			return err
		}
		s.device = registration
		s.logger.Info("device registered", "device_id", registration.Device.ID, "virtual_ip", registration.Device.VirtualIP)
	}
	if s.tunnel != nil {
		if err := s.tunnel.Apply(ctx, s.device); err != nil {
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
