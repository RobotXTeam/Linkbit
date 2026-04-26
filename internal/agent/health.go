package agent

import (
	"context"

	"github.com/linkbit/linkbit/internal/models"
)

type DeviceHealthClient interface {
	ReportHealth(ctx context.Context, report models.DeviceHealthReport) (models.Device, error)
}

type ControllerHealthReporter struct {
	client   DeviceHealthClient
	endpoint string
}

func NewControllerHealthReporter(client DeviceHealthClient, endpoint ...string) *ControllerHealthReporter {
	value := ""
	if len(endpoint) > 0 {
		value = endpoint[0]
	}
	return &ControllerHealthReporter{client: client, endpoint: value}
}

func (r *ControllerHealthReporter) CheckAndReport(ctx context.Context, registration models.DeviceRegistrationResponse) error {
	if r.client == nil || registration.Device.ID == "" {
		return nil
	}
	_, err := r.client.ReportHealth(ctx, models.DeviceHealthReport{
		Status:   models.DeviceStatusOnline,
		Endpoint: r.endpoint,
	})
	return err
}
