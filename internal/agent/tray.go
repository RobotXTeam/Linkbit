package agent

import (
	"context"
	"log/slog"
)

type TrayController interface {
	Run(context.Context) error
}

type NoopTray struct {
	logger *slog.Logger
}

func NewNoopTray(logger *slog.Logger) *NoopTray {
	if logger == nil {
		logger = slog.Default()
	}
	return &NoopTray{logger: logger}
}

func (t *NoopTray) Run(ctx context.Context) error {
	t.logger.Info("tray integration is not enabled for this build")
	<-ctx.Done()
	return ctx.Err()
}
