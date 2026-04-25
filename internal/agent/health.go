package agent

import (
	"context"

	"github.com/linkbit/linkbit/internal/models"
)

type DeviceHealthClient interface {
	ReportHealth(ctx context.Context, report models.DeviceHealthReport) (models.Device, error)
}

type ControllerHealthReporter struct {
	client DeviceHealthClient
}

func NewControllerHealthReporter(client DeviceHealthClient) *ControllerHealthReporter {
	return &ControllerHealthReporter{client: client}
}

func (r *ControllerHealthReporter) CheckAndReport(ctx context.Context, registration models.DeviceRegistrationResponse) error {
	if r.client == nil || registration.Device.ID == "" {
		return nil
	}
	_, err := r.client.ReportHealth(ctx, models.DeviceHealthReport{
		Status: models.DeviceStatusOnline,
	})
	return err
}
