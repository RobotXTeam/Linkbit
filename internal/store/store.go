package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/linkbit/linkbit/internal/models"
)

type Store interface {
	Migrate(context.Context) error
	Close() error

	UpsertRelay(context.Context, models.RelayNode) error
	HeartbeatRelay(context.Context, string, float64) (models.RelayNode, error)
	ListRelays(context.Context) ([]models.RelayNode, error)

	CreateInvitation(context.Context, models.Invitation) error
	GetInvitationByTokenHash(context.Context, string) (models.Invitation, error)
	MarkInvitationUsed(context.Context, string) error

	CreateDevice(context.Context, models.Device) error
	ListDevices(context.Context) ([]models.Device, error)

	CreatePolicy(context.Context, models.NetworkPolicy) error
	ListPolicies(context.Context) ([]models.NetworkPolicy, error)

	Overview(context.Context) (models.Overview, error)
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
