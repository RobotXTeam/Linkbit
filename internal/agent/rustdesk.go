package agent

import (
	"context"
	"errors"
	"os/exec"
)

type RustDeskLauncher struct {
	binary string
}

func NewRustDeskLauncher(binary string) *RustDeskLauncher {
	return &RustDeskLauncher{binary: binary}
}

func (l *RustDeskLauncher) Launch(ctx context.Context, args ...string) error {
	if l.binary == "" {
		return errors.New("rustdesk binary is not configured")
	}
	return exec.CommandContext(ctx, l.binary, args...).Start()
}
