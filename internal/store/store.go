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

	CreateUser(context.Context, models.User) error
	ListUsers(context.Context) ([]models.User, error)
	GetUser(context.Context, string) (models.User, error)

	CreateGroup(context.Context, models.DeviceGroup) error
	ListGroups(context.Context) ([]models.DeviceGroup, error)
	GetGroup(context.Context, string) (models.DeviceGroup, error)

	UpsertRelay(context.Context, models.RelayNode) error
	DeleteRelay(context.Context, string) error
	HeartbeatRelay(context.Context, string, float64) (models.RelayNode, error)
	ListRelays(context.Context) ([]models.RelayNode, error)

	CreateAPIKey(context.Context, models.APIKey) error
	GetAPIKeyByDigest(context.Context, string) (models.APIKey, error)
	TouchAPIKey(context.Context, string) error
	RevokeAPIKey(context.Context, string) error
	ListAPIKeys(context.Context) ([]models.APIKey, error)

	CreateInvitation(context.Context, models.Invitation) error
	GetInvitationByTokenHash(context.Context, string) (models.Invitation, error)
	MarkInvitationUsed(context.Context, string) error

	CreateDevice(context.Context, models.Device) error
	GetDeviceByIDAndTokenHash(context.Context, string, string) (models.Device, error)
	UpdateDeviceHealth(context.Context, string, string, models.DeviceHealthReport) (models.Device, error)
	ListDevices(context.Context) ([]models.Device, error)

	CreatePolicy(context.Context, models.NetworkPolicy) error
	DeletePolicy(context.Context, string) error
	ListPolicies(context.Context) ([]models.NetworkPolicy, error)

	Overview(context.Context) (models.Overview, error)
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
