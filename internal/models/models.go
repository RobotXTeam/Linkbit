package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserCreateRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type DeviceGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DeviceGroupCreateRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

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
	Endpoint    string       `json:"endpoint,omitempty"`
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
	Endpoint      string `json:"endpoint"`
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
	Endpoint       string       `json:"endpoint,omitempty"`
}

type NetworkPeer struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	VirtualIP  string   `json:"virtualIp"`
	PublicKey  string   `json:"publicKey"`
	Endpoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowedIps,omitempty"`
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

type ControllerSettings struct {
	PublicURL         string `json:"publicUrl"`
	ListenAddr        string `json:"listenAddr"`
	LogLevel          string `json:"logLevel"`
	WebConsoleEnabled bool   `json:"webConsoleEnabled"`
	DatabaseBackend   string `json:"databaseBackend"`
}

type TCPRelaySessionRequest struct {
	SourceDeviceID string `json:"sourceDeviceId"`
	Target         string `json:"target"`
	TargetPort     int    `json:"targetPort"`
	Protocol       string `json:"protocol"`
}

type TCPRelaySession struct {
	ID              string    `json:"id"`
	SourceDeviceID  string    `json:"sourceDeviceId"`
	TargetDeviceID  string    `json:"targetDeviceId"`
	TargetName      string    `json:"targetName"`
	TargetVirtualIP string    `json:"targetVirtualIp"`
	TargetPort      int       `json:"targetPort"`
	Protocol        string    `json:"protocol"`
	ExpiresAt       time.Time `json:"expiresAt"`
}
