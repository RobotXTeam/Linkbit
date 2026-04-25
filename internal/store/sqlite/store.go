package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkbit/linkbit/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	path string
	db   *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &Store{path: path, db: db}, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS relays (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	region TEXT NOT NULL,
	public_url TEXT NOT NULL,
	ipv4 TEXT NOT NULL DEFAULT '',
	ipv6 TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	load REAL NOT NULL DEFAULT 0,
	last_seen_at TEXT NOT NULL,
	registered TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS invitations (
	id TEXT PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL,
	group_id TEXT NOT NULL,
	reusable INTEGER NOT NULL DEFAULT 0,
	expires_at TEXT NOT NULL,
	used_at TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS devices (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	group_id TEXT NOT NULL,
	name TEXT NOT NULL,
	virtual_ip TEXT NOT NULL UNIQUE,
	public_key TEXT NOT NULL,
	status TEXT NOT NULL,
	last_seen_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	fingerprint TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS policies (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	source_id TEXT NOT NULL,
	target_id TEXT NOT NULL,
	ports_json TEXT NOT NULL,
	protocol TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`)
	return err
}

func (s *Store) UpsertRelay(ctx context.Context, relay models.RelayNode) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO relays (id, name, region, public_url, ipv4, ipv6, status, load, last_seen_at, registered)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	region = excluded.region,
	public_url = excluded.public_url,
	ipv4 = excluded.ipv4,
	ipv6 = excluded.ipv6,
	status = excluded.status,
	load = excluded.load,
	last_seen_at = excluded.last_seen_at
`, relay.ID, relay.Name, relay.Region, relay.PublicURL, relay.IPv4, relay.IPv6, relay.Status, relay.Load, formatTime(relay.LastSeenAt), formatTime(relay.Registered))
	return err
}

func (s *Store) HeartbeatRelay(ctx context.Context, id string, load float64) (models.RelayNode, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
UPDATE relays SET status = ?, load = ?, last_seen_at = ? WHERE id = ?
`, models.RelayStatusHealthy, load, formatTime(now), id)
	if err != nil {
		return models.RelayNode{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return models.RelayNode{}, err
	}
	if affected == 0 {
		return models.RelayNode{}, sql.ErrNoRows
	}
	return s.getRelay(ctx, id)
}

func (s *Store) ListRelays(ctx context.Context) ([]models.RelayNode, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, region, public_url, ipv4, ipv6, status, load, last_seen_at, registered
FROM relays ORDER BY region, name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relays []models.RelayNode
	for rows.Next() {
		relay, err := scanRelay(rows)
		if err != nil {
			return nil, err
		}
		relays = append(relays, relay)
	}
	return relays, rows.Err()
}

func (s *Store) CreateInvitation(ctx context.Context, invitation models.Invitation) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO invitations (id, token_hash, user_id, group_id, reusable, expires_at, used_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, invitation.ID, invitation.TokenHash, invitation.UserID, invitation.GroupID, boolInt(invitation.Reusable), formatTime(invitation.ExpiresAt), formatOptionalTime(invitation.UsedAt), formatTime(invitation.CreatedAt))
	return err
}

func (s *Store) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (models.Invitation, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, token_hash, user_id, group_id, reusable, expires_at, used_at, created_at
FROM invitations WHERE token_hash = ?
`, tokenHash)
	return scanInvitation(row)
}

func (s *Store) MarkInvitationUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE invitations SET used_at = ? WHERE id = ?
`, formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) CreateDevice(ctx context.Context, device models.Device) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO devices (id, user_id, group_id, name, virtual_ip, public_key, status, last_seen_at, created_at, fingerprint)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, device.ID, device.UserID, device.GroupID, device.Name, device.VirtualIP, device.PublicKey, device.Status, formatTime(device.LastSeenAt), formatTime(device.CreatedAt), device.Fingerprint)
	return err
}

func (s *Store) ListDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, group_id, name, virtual_ip, public_key, status, last_seen_at, created_at, fingerprint
FROM devices ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

func (s *Store) CreatePolicy(ctx context.Context, policy models.NetworkPolicy) error {
	ports, err := json.Marshal(policy.Ports)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO policies (id, name, source_id, target_id, ports_json, protocol, enabled, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, policy.ID, policy.Name, policy.SourceID, policy.TargetID, string(ports), policy.Protocol, boolInt(policy.Enabled), policy.Description, formatTime(policy.CreatedAt), formatTime(policy.UpdatedAt))
	return err
}

func (s *Store) ListPolicies(ctx context.Context) ([]models.NetworkPolicy, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, source_id, target_id, ports_json, protocol, enabled, description, created_at, updated_at
FROM policies ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.NetworkPolicy
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func (s *Store) Overview(ctx context.Context) (models.Overview, error) {
	var overview models.Overview
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices`).Scan(&overview.TotalDevices); err != nil {
		return overview, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE status = ?`, models.DeviceStatusOnline).Scan(&overview.OnlineDevices); err != nil {
		return overview, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relays`).Scan(&overview.RelayNodes); err != nil {
		return overview, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relays WHERE status = ?`, models.RelayStatusHealthy).Scan(&overview.HealthyRelays); err != nil {
		return overview, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM policies`).Scan(&overview.PolicyCount); err != nil {
		return overview, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(AVG(load), 0) FROM relays`).Scan(&overview.AverageLoad); err != nil {
		return overview, err
	}
	overview.NetworkHealth = "unknown"
	if overview.RelayNodes > 0 && overview.HealthyRelays == overview.RelayNodes {
		overview.NetworkHealth = "healthy"
	} else if overview.HealthyRelays > 0 {
		overview.NetworkHealth = "degraded"
	}
	return overview, nil
}

func (s *Store) getRelay(ctx context.Context, id string) (models.RelayNode, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, region, public_url, ipv4, ipv6, status, load, last_seen_at, registered
FROM relays WHERE id = ?
`, id)
	return scanRelay(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRelay(row scanner) (models.RelayNode, error) {
	var relay models.RelayNode
	var lastSeenAt, registered string
	err := row.Scan(&relay.ID, &relay.Name, &relay.Region, &relay.PublicURL, &relay.IPv4, &relay.IPv6, &relay.Status, &relay.Load, &lastSeenAt, &registered)
	if err != nil {
		return relay, err
	}
	relay.LastSeenAt = parseTime(lastSeenAt)
	relay.Registered = parseTime(registered)
	return relay, nil
}

func scanInvitation(row scanner) (models.Invitation, error) {
	var invitation models.Invitation
	var reusable int
	var expiresAt, usedAt, createdAt string
	err := row.Scan(&invitation.ID, &invitation.TokenHash, &invitation.UserID, &invitation.GroupID, &reusable, &expiresAt, &usedAt, &createdAt)
	if err != nil {
		return invitation, err
	}
	invitation.Reusable = reusable == 1
	invitation.ExpiresAt = parseTime(expiresAt)
	invitation.UsedAt = parseTime(usedAt)
	invitation.CreatedAt = parseTime(createdAt)
	return invitation, nil
}

func scanDevice(row scanner) (models.Device, error) {
	var device models.Device
	var lastSeenAt, createdAt string
	err := row.Scan(&device.ID, &device.UserID, &device.GroupID, &device.Name, &device.VirtualIP, &device.PublicKey, &device.Status, &lastSeenAt, &createdAt, &device.Fingerprint)
	if err != nil {
		return device, err
	}
	device.LastSeenAt = parseTime(lastSeenAt)
	device.CreatedAt = parseTime(createdAt)
	return device, nil
}

func scanPolicy(row scanner) (models.NetworkPolicy, error) {
	var policy models.NetworkPolicy
	var portsJSON, createdAt, updatedAt string
	var enabled int
	err := row.Scan(&policy.ID, &policy.Name, &policy.SourceID, &policy.TargetID, &portsJSON, &policy.Protocol, &enabled, &policy.Description, &createdAt, &updatedAt)
	if err != nil {
		return policy, err
	}
	if err := json.Unmarshal([]byte(portsJSON), &policy.Ports); err != nil {
		return policy, fmt.Errorf("decode policy ports: %w", err)
	}
	policy.Enabled = enabled == 1
	policy.CreatedAt = parseTime(createdAt)
	policy.UpdatedAt = parseTime(updatedAt)
	return policy, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatOptionalTime(t time.Time) string {
	return formatTime(t)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
