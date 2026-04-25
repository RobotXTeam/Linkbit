package models

import "time"

type DeviceStatus string

const (
	DeviceStatusPending DeviceStatus = "pending"
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
)

type Device struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId"`
	GroupID     string       `json:"groupId"`
	Name        string       `json:"name"`
	VirtualIP   string       `json:"virtualIp"`
	PublicKey   string       `json:"publicKey"`
	TokenHash   string       `json:"-"`
	DeviceToken string       `json:"deviceToken,omitempty"`
	Status      DeviceStatus `json:"status"`
	LastSeenAt  time.Time    `json:"lastSeenAt"`
	CreatedAt   time.Time    `json:"createdAt"`
	Fingerprint string       `json:"fingerprint"`
}

type DeviceRegistrationRequest struct {
	EnrollmentKey string `json:"enrollmentKey"`
	Name          string `json:"name"`
	PublicKey     string `json:"publicKey"`
	Fingerprint   string `json:"fingerprint"`
}

type DeviceRegistrationResponse struct {
	Device  Device      `json:"device"`
	Relays  []RelayNode `json:"relays"`
	Message string      `json:"message"`
}

type DeviceHealthReport struct {
	Status         DeviceStatus `json:"status"`
	LatencyMS      int          `json:"latencyMs"`
	PeersReachable int          `json:"peersReachable"`
	PeersTotal     int          `json:"peersTotal"`
}

type NetworkPeer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	VirtualIP string `json:"virtualIp"`
	PublicKey string `json:"publicKey"`
}

type NetworkConfig struct {
	Device   Device          `json:"device"`
	Peers    []NetworkPeer   `json:"peers"`
	Policies []NetworkPolicy `json:"policies"`
	Relays   []RelayNode     `json:"relays"`
}

type RelayStatus string

const (
	RelayStatusHealthy   RelayStatus = "healthy"
	RelayStatusDegraded  RelayStatus = "degraded"
	RelayStatusUnhealthy RelayStatus = "unhealthy"
)

type RelayNode struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Region     string      `json:"region"`
	PublicURL  string      `json:"publicUrl"`
	IPv4       string      `json:"ipv4"`
	IPv6       string      `json:"ipv6,omitempty"`
	Status     RelayStatus `json:"status"`
	Load       float64     `json:"load"`
	LastSeenAt time.Time   `json:"lastSeenAt"`
	Registered time.Time   `json:"registered"`
}

type NetworkPolicy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SourceID    string    `json:"sourceId"`
	TargetID    string    `json:"targetId"`
	Ports       []string  `json:"ports"`
	Protocol    string    `json:"protocol"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Invitation struct {
	ID             string    `json:"id"`
	TokenHash      string    `json:"-"`
	UserID         string    `json:"userId"`
	GroupID        string    `json:"groupId"`
	Reusable       bool      `json:"reusable"`
	ExpiresAt      time.Time `json:"expiresAt"`
	UsedAt         time.Time `json:"usedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	PlaintextToken string    `json:"token,omitempty"`
}

type InvitationCreateRequest struct {
	UserID           string `json:"userId"`
	GroupID          string `json:"groupId"`
	Reusable         bool   `json:"reusable"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
}

type APIKey struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Digest       string    `json:"-"`
	Scope        string    `json:"scope"`
	CreatedAt    time.Time `json:"createdAt"`
	LastUsedAt   time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt    time.Time `json:"revokedAt,omitempty"`
	PlaintextKey string    `json:"key,omitempty"`
}

type APIKeyCreateRequest struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type Overview struct {
	OnlineDevices int     `json:"onlineDevices"`
	TotalDevices  int     `json:"totalDevices"`
	RelayNodes    int     `json:"relayNodes"`
	HealthyRelays int     `json:"healthyRelays"`
	PolicyCount   int     `json:"policyCount"`
	NetworkHealth string  `json:"networkHealth"`
	AverageLoad   float64 `json:"averageLoad"`
}
