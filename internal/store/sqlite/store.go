package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS groups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
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
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	digest TEXT NOT NULL UNIQUE,
	scope TEXT NOT NULL,
	created_at TEXT NOT NULL,
	last_used_at TEXT NOT NULL DEFAULT '',
	revoked_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS devices (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	group_id TEXT NOT NULL,
	name TEXT NOT NULL,
	virtual_ip TEXT NOT NULL UNIQUE,
	public_key TEXT NOT NULL,
	token_hash TEXT NOT NULL DEFAULT '',
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
	if err != nil {
		return err
	}
	// SQLite lacks IF NOT EXISTS for ADD COLUMN on older versions, so ignore the
	// duplicate-column error while still surfacing any unexpected migration issue.
	for _, statement := range []string{
		`ALTER TABLE devices ADD COLUMN token_hash TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil && !isDuplicateColumn(err) {
			return err
		}
	}
	return err
}

func (s *Store) CreateUser(ctx context.Context, user models.User) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO users (id, name, email, role, created_at)
VALUES (?, ?, ?, ?, ?)
`, user.ID, user.Name, user.Email, user.Role, formatTime(user.CreatedAt))
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, email, role, created_at FROM users ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(ctx context.Context, id string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, role, created_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) CreateGroup(ctx context.Context, group models.DeviceGroup) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO groups (id, name, description, created_at)
VALUES (?, ?, ?, ?)
`, group.ID, group.Name, group.Description, formatTime(group.CreatedAt))
	return err
}

func (s *Store) ListGroups(ctx context.Context) ([]models.DeviceGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, description, created_at FROM groups ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.DeviceGroup
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (s *Store) GetGroup(ctx context.Context, id string) (models.DeviceGroup, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, description, created_at FROM groups WHERE id = ?`, id)
	return scanGroup(row)
}

func (s *Store) CreateAPIKey(ctx context.Context, apiKey models.APIKey) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO api_keys (id, name, digest, scope, created_at, last_used_at, revoked_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, apiKey.ID, apiKey.Name, apiKey.Digest, apiKey.Scope, formatTime(apiKey.CreatedAt), formatOptionalTime(apiKey.LastUsedAt), formatOptionalTime(apiKey.RevokedAt))
	return err
}

func (s *Store) GetAPIKeyByDigest(ctx context.Context, digest string) (models.APIKey, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, digest, scope, created_at, last_used_at, revoked_at
FROM api_keys WHERE digest = ? AND revoked_at = ''
`, digest)
	return scanAPIKey(row)
}

func (s *Store) TouchAPIKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) ListAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, digest, scope, created_at, last_used_at, revoked_at
FROM api_keys ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		apiKey, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, apiKey)
	}
	return apiKeys, rows.Err()
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

func (s *Store) DeleteRelay(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM relays WHERE id = ?`, id)
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
INSERT INTO devices (id, user_id, group_id, name, virtual_ip, public_key, token_hash, status, last_seen_at, created_at, fingerprint)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, device.ID, device.UserID, device.GroupID, device.Name, device.VirtualIP, device.PublicKey, device.TokenHash, device.Status, formatTime(device.LastSeenAt), formatTime(device.CreatedAt), device.Fingerprint)
	return err
}

func (s *Store) GetDeviceByIDAndTokenHash(ctx context.Context, id string, tokenHash string) (models.Device, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, user_id, group_id, name, virtual_ip, public_key, token_hash, status, last_seen_at, created_at, fingerprint
FROM devices WHERE id = ? AND token_hash = ?
`, id, tokenHash)
	return scanDevice(row)
}

func (s *Store) UpdateDeviceHealth(ctx context.Context, id string, tokenHash string, report models.DeviceHealthReport) (models.Device, error) {
	status := report.Status
	if status == "" {
		status = models.DeviceStatusOnline
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE devices SET status = ?, last_seen_at = ? WHERE id = ? AND token_hash = ?
`, status, formatTime(time.Now().UTC()), id, tokenHash)
	if err != nil {
		return models.Device{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return models.Device{}, err
	}
	if affected == 0 {
		return models.Device{}, sql.ErrNoRows
	}
	return s.getDevice(ctx, id)
}

func (s *Store) ListDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, group_id, name, virtual_ip, public_key, token_hash, status, last_seen_at, created_at, fingerprint
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

func (s *Store) getDevice(ctx context.Context, id string) (models.Device, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, user_id, group_id, name, virtual_ip, public_key, token_hash, status, last_seen_at, created_at, fingerprint
FROM devices WHERE id = ?
`, id)
	return scanDevice(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (models.User, error) {
	var user models.User
	var createdAt string
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &createdAt)
	if err != nil {
		return user, err
	}
	user.CreatedAt = parseTime(createdAt)
	return user, nil
}

func scanGroup(row scanner) (models.DeviceGroup, error) {
	var group models.DeviceGroup
	var createdAt string
	err := row.Scan(&group.ID, &group.Name, &group.Description, &createdAt)
	if err != nil {
		return group, err
	}
	group.CreatedAt = parseTime(createdAt)
	return group, nil
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

func scanAPIKey(row scanner) (models.APIKey, error) {
	var apiKey models.APIKey
	var createdAt, lastUsedAt, revokedAt string
	err := row.Scan(&apiKey.ID, &apiKey.Name, &apiKey.Digest, &apiKey.Scope, &createdAt, &lastUsedAt, &revokedAt)
	if err != nil {
		return apiKey, err
	}
	apiKey.CreatedAt = parseTime(createdAt)
	apiKey.LastUsedAt = parseTime(lastUsedAt)
	apiKey.RevokedAt = parseTime(revokedAt)
	return apiKey, nil
}

func scanDevice(row scanner) (models.Device, error) {
	var device models.Device
	var lastSeenAt, createdAt string
	err := row.Scan(&device.ID, &device.UserID, &device.GroupID, &device.Name, &device.VirtualIP, &device.PublicKey, &device.TokenHash, &device.Status, &lastSeenAt, &createdAt, &device.Fingerprint)
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

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
