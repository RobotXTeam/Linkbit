package agent

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
)

type serviceClient struct {
	registerCalls int
	deviceID      string
	deviceToken   string
}

func (c *serviceClient) Register(_ context.Context, enrollmentKey string, deviceName string) (models.DeviceRegistrationResponse, error) {
	c.registerCalls++
	return models.DeviceRegistrationResponse{
		Device: models.Device{
			ID:          "device-id",
			Name:        deviceName,
			VirtualIP:   "100.96.1.2",
			DeviceToken: "device-token-" + enrollmentKey,
		},
	}, nil
}

func (c *serviceClient) SetDeviceCredentials(deviceID string, deviceToken string) {
	c.deviceID = deviceID
	c.deviceToken = deviceToken
}

func (c *serviceClient) GetNetworkConfig(_ context.Context) (models.NetworkConfig, error) {
	return models.NetworkConfig{Device: models.Device{ID: c.deviceID, VirtualIP: "100.96.1.2"}}, nil
}

func (c *serviceClient) ReportHealth(_ context.Context, _ models.DeviceHealthReport) (models.Device, error) {
	return models.Device{ID: c.deviceID, Status: models.DeviceStatusOnline}, nil
}

type serviceTunnel struct {
	applyCalls int
}

func (t *serviceTunnel) Apply(_ context.Context, _ models.NetworkConfig) error {
	t.applyCalls++
	return nil
}

func (t *serviceTunnel) Destroy(_ context.Context) error {
	return nil
}

func TestServicePersistsAndReusesDeviceState(t *testing.T) {
	statePath := t.TempDir() + "/agent-state.json"

	firstClient := &serviceClient{}
	firstTunnel := &serviceTunnel{}
	first, err := NewService(config.AgentConfig{
		ControllerURL:      "https://controller.example.com",
		EnrollmentKey:      "invite",
		DeviceName:         "device-1",
		HealthEvery:        time.Millisecond,
		WireGuardInterface: "linkbit0",
		StatePath:          statePath,
	}, firstClient, firstTunnel, NewControllerHealthReporter(firstClient), slog.Default())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	runBriefly(t, first)
	if firstClient.registerCalls != 1 {
		t.Fatalf("first register calls = %d, want 1", firstClient.registerCalls)
	}
	if firstTunnel.applyCalls != 1 {
		t.Fatalf("first apply calls = %d, want 1", firstTunnel.applyCalls)
	}

	secondClient := &serviceClient{}
	secondTunnel := &serviceTunnel{}
	second, err := NewService(config.AgentConfig{
		ControllerURL:      "https://controller.example.com",
		DeviceName:         "device-1",
		HealthEvery:        time.Millisecond,
		WireGuardInterface: "linkbit0",
		StatePath:          statePath,
	}, secondClient, secondTunnel, NewControllerHealthReporter(secondClient), slog.Default())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	runBriefly(t, second)
	if secondClient.registerCalls != 0 {
		t.Fatalf("second register calls = %d, want 0", secondClient.registerCalls)
	}
	if secondClient.deviceID != "device-id" || secondClient.deviceToken != "device-token-invite" {
		t.Fatalf("restored credentials = %s/%s", secondClient.deviceID, secondClient.deviceToken)
	}
}

func TestServiceRunOnceReportsAndExits(t *testing.T) {
	statePath := t.TempDir() + "/agent-state.json"
	client := &serviceClient{}
	tunnel := &serviceTunnel{}
	service, err := NewService(config.AgentConfig{
		ControllerURL:      "https://controller.example.com",
		EnrollmentKey:      "invite",
		DeviceName:         "device-1",
		HealthEvery:        time.Hour,
		WireGuardInterface: "linkbit0",
		StatePath:          statePath,
		RunOnce:            true,
	}, client, tunnel, NewControllerHealthReporter(client), slog.Default())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if client.registerCalls != 1 || tunnel.applyCalls != 1 {
		t.Fatalf("register/apply calls = %d/%d, want 1/1", client.registerCalls, tunnel.applyCalls)
	}
}

func runBriefly(t *testing.T, service *Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Millisecond)
	defer cancel()
	err := service.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Run() error = %v", err)
	}
}
