package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/linkbit/linkbit/internal/auth"
	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
	"github.com/linkbit/linkbit/internal/store"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

type Server struct {
	cfg         config.ControllerConfig
	logger      *slog.Logger
	adminDigest string
	store       store.Store
}

func NewServer(cfg config.ControllerConfig, logger *slog.Logger, bootstrapAPIKey string, storage store.Store) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if storage == nil {
		return nil, errors.New("store is required")
	}
	digest, err := auth.HashAPIKey(bootstrapAPIKey, cfg.APIKeyPepper)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:         cfg,
		logger:      logger,
		adminDigest: digest,
		store:       storage,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.Handle("GET /metrics", s.requireAPIKey(http.HandlerFunc(s.handleMetrics)))
	mux.Handle("GET /api/v1/overview", s.requireAPIKey(http.HandlerFunc(s.handleOverview)))
	mux.Handle("GET /api/v1/settings", s.requireAPIKey(http.HandlerFunc(s.handleSettings)))
	mux.Handle("POST /api/v1/users", s.requireAPIKey(http.HandlerFunc(s.handleUserCreate)))
	mux.Handle("GET /api/v1/users", s.requireAPIKey(http.HandlerFunc(s.handleUserList)))
	mux.Handle("POST /api/v1/groups", s.requireAPIKey(http.HandlerFunc(s.handleGroupCreate)))
	mux.Handle("GET /api/v1/groups", s.requireAPIKey(http.HandlerFunc(s.handleGroupList)))
	mux.Handle("GET /api/v1/derp-map", s.requireAPIKey(http.HandlerFunc(s.handleDERPMap)))
	mux.Handle("POST /api/v1/api-keys", s.requireAPIKey(http.HandlerFunc(s.handleAPIKeyCreate)))
	mux.Handle("GET /api/v1/api-keys", s.requireAPIKey(http.HandlerFunc(s.handleAPIKeyList)))
	mux.Handle("DELETE /api/v1/api-keys/{id}", s.requireAPIKey(http.HandlerFunc(s.handleAPIKeyRevoke)))
	mux.Handle("POST /api/v1/relays/register", s.requireAPIKeyScopes(http.HandlerFunc(s.handleRelayRegister), "admin", "relay"))
	mux.Handle("DELETE /api/v1/relays/{id}", s.requireAPIKey(http.HandlerFunc(s.handleRelayDelete)))
	mux.Handle("POST /api/v1/relays/heartbeat", s.requireAPIKeyScopes(http.HandlerFunc(s.handleRelayHeartbeat), "admin", "relay"))
	mux.Handle("GET /api/v1/relays", s.requireAPIKey(http.HandlerFunc(s.handleRelayList)))
	mux.Handle("GET /api/v1/devices", s.requireAPIKey(http.HandlerFunc(s.handleDeviceList)))
	mux.HandleFunc("POST /api/v1/devices/register", s.handleDeviceRegister)
	mux.HandleFunc("POST /api/v1/devices/{id}/health", s.handleDeviceHealth)
	mux.HandleFunc("GET /api/v1/devices/{id}/network-config", s.handleDeviceNetworkConfig)
	mux.Handle("POST /api/v1/invitations", s.requireAPIKey(http.HandlerFunc(s.handleInvitationCreate)))
	mux.Handle("POST /api/v1/policies", s.requireAPIKey(http.HandlerFunc(s.handlePolicyCreate)))
	mux.Handle("GET /api/v1/policies", s.requireAPIKey(http.HandlerFunc(s.handlePolicyList)))
	mux.Handle("DELETE /api/v1/policies/{id}", s.requireAPIKey(http.HandlerFunc(s.handlePolicyDelete)))
	return securityHeaders(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.store.Overview(r.Context())
	if err != nil {
		s.logger.Error("overview failed", "err", err)
		writeError(w, http.StatusInternalServerError, "overview failed")
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, models.ControllerSettings{
		PublicURL:         s.cfg.PublicURL,
		ListenAddr:        s.cfg.ListenAddr,
		LogLevel:          s.cfg.LogLevel,
		WebConsoleEnabled: s.cfg.WebDir != "",
		DatabaseBackend:   "sqlite",
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	overview, err := s.store.Overview(r.Context())
	if err != nil {
		s.logger.Error("metrics overview failed", "err", err)
		writeError(w, http.StatusInternalServerError, "metrics failed")
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "linkbit_devices_total %d\n", overview.TotalDevices)
	fmt.Fprintf(w, "linkbit_devices_online %d\n", overview.OnlineDevices)
	fmt.Fprintf(w, "linkbit_relays_total %d\n", overview.RelayNodes)
	fmt.Fprintf(w, "linkbit_relays_healthy %d\n", overview.HealthyRelays)
	fmt.Fprintf(w, "linkbit_policies_total %d\n", overview.PolicyCount)
}

func (s *Server) handleDERPMap(w http.ResponseWriter, r *http.Request) {
	relays, err := s.store.ListRelays(r.Context())
	if err != nil {
		s.logger.Error("list relays for derp map failed", "err", err)
		writeError(w, http.StatusInternalServerError, "derp map failed")
		return
	}
	writeJSON(w, http.StatusOK, buildDERPMap(relays))
}

func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	var req models.UserCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid user payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		req.ID = uuid.NewString()
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if req.Role != "admin" && req.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be admin or member")
		return
	}
	user := models.User{
		ID:        strings.TrimSpace(req.ID),
		Name:      strings.TrimSpace(req.Name),
		Email:     strings.TrimSpace(req.Email),
		Role:      req.Role,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateUser(r.Context(), user); err != nil {
		s.logger.Error("create user failed", "err", err)
		writeError(w, http.StatusInternalServerError, "user creation failed")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleUserList(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		s.logger.Error("list users failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list users failed")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleGroupCreate(w http.ResponseWriter, r *http.Request) {
	var req models.DeviceGroupCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid group payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		req.ID = uuid.NewString()
	}
	group := models.DeviceGroup{
		ID:          strings.TrimSpace(req.ID),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.store.CreateGroup(r.Context(), group); err != nil {
		s.logger.Error("create group failed", "err", err)
		writeError(w, http.StatusInternalServerError, "group creation failed")
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (s *Server) handleGroupList(w http.ResponseWriter, r *http.Request) {
	groups, err := s.store.ListGroups(r.Context())
	if err != nil {
		s.logger.Error("list groups failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list groups failed")
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleRelayRegister(w http.ResponseWriter, r *http.Request) {
	var relay models.RelayNode
	if err := decodeJSON(w, r, &relay); err != nil {
		writeError(w, http.StatusBadRequest, "invalid relay payload")
		return
	}
	if strings.TrimSpace(relay.ID) == "" || strings.TrimSpace(relay.PublicURL) == "" {
		writeError(w, http.StatusBadRequest, "relay id and publicUrl are required")
		return
	}

	now := time.Now().UTC()
	relay.Registered = now
	relay.LastSeenAt = now
	relay.Status = models.RelayStatusHealthy

	if err := s.store.UpsertRelay(r.Context(), relay); err != nil {
		s.logger.Error("relay registration failed", "err", err)
		writeError(w, http.StatusInternalServerError, "relay registration failed")
		return
	}

	s.logger.Info("relay registered", "relay_id", relay.ID, "public_url", relay.PublicURL)
	writeJSON(w, http.StatusCreated, relay)
}

func (s *Server) handleRelayDelete(w http.ResponseWriter, r *http.Request) {
	relayID := r.PathValue("id")
	if relayID == "" {
		writeError(w, http.StatusBadRequest, "relay id is required")
		return
	}
	if err := s.store.DeleteRelay(r.Context(), relayID); err != nil {
		s.logger.Error("delete relay failed", "err", err)
		writeError(w, http.StatusInternalServerError, "delete relay failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	var req models.APIKeyCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid api key payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Scope == "" {
		req.Scope = "admin"
	}
	if req.Scope != "admin" && req.Scope != "relay" {
		writeError(w, http.StatusBadRequest, "scope must be admin or relay")
		return
	}

	key, err := auth.NewAPIKey()
	if err != nil {
		s.logger.Error("generate api key failed", "err", err)
		writeError(w, http.StatusInternalServerError, "api key creation failed")
		return
	}
	digest, err := auth.HashAPIKey(key, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "api key creation failed")
		return
	}
	apiKey := models.APIKey{
		ID:           uuid.NewString(),
		Name:         strings.TrimSpace(req.Name),
		Digest:       digest,
		Scope:        req.Scope,
		CreatedAt:    time.Now().UTC(),
		PlaintextKey: key,
	}
	if err := s.store.CreateAPIKey(r.Context(), apiKey); err != nil {
		s.logger.Error("persist api key failed", "err", err)
		writeError(w, http.StatusInternalServerError, "api key creation failed")
		return
	}
	writeJSON(w, http.StatusCreated, apiKey)
}

func (s *Server) handleAPIKeyList(w http.ResponseWriter, r *http.Request) {
	apiKeys, err := s.store.ListAPIKeys(r.Context())
	if err != nil {
		s.logger.Error("list api keys failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list api keys failed")
		return
	}
	writeJSON(w, http.StatusOK, apiKeys)
}

func (s *Server) handleAPIKeyRevoke(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "api key id is required")
		return
	}
	if err := s.store.RevokeAPIKey(r.Context(), keyID); store.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "api key not found")
		return
	} else if err != nil {
		s.logger.Error("revoke api key failed", "err", err)
		writeError(w, http.StatusInternalServerError, "api key revoke failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRelayHeartbeat(w http.ResponseWriter, r *http.Request) {
	var relay models.RelayNode
	if err := decodeJSON(w, r, &relay); err != nil {
		writeError(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}
	if relay.ID == "" {
		writeError(w, http.StatusBadRequest, "relay id is required")
		return
	}

	existing, err := s.store.HeartbeatRelay(r.Context(), relay.ID, relay.Load)
	if store.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "relay is not registered")
		return
	}
	if err != nil {
		s.logger.Error("relay heartbeat failed", "err", err)
		writeError(w, http.StatusInternalServerError, "relay heartbeat failed")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleRelayList(w http.ResponseWriter, r *http.Request) {
	relays, err := s.store.ListRelays(r.Context())
	if err != nil {
		s.logger.Error("list relays failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list relays failed")
		return
	}
	writeJSON(w, http.StatusOK, relays)
}

func (s *Server) handleDeviceList(w http.ResponseWriter, r *http.Request) {
	devices, err := s.store.ListDevices(r.Context())
	if err != nil {
		s.logger.Error("list devices failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list devices failed")
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (s *Server) handleDeviceRegister(w http.ResponseWriter, r *http.Request) {
	var req models.DeviceRegistrationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid registration payload")
		return
	}
	if strings.TrimSpace(req.EnrollmentKey) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.PublicKey) == "" {
		writeError(w, http.StatusBadRequest, "enrollmentKey, name, and publicKey are required")
		return
	}

	tokenHash, err := auth.HashAPIKey(req.EnrollmentKey, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid enrollment key")
		return
	}
	invitation, err := s.store.GetInvitationByTokenHash(r.Context(), tokenHash)
	if store.IsNotFound(err) {
		writeError(w, http.StatusUnauthorized, "invalid enrollment key")
		return
	}
	if err != nil {
		s.logger.Error("lookup invitation failed", "err", err)
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}
	if time.Now().UTC().After(invitation.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "enrollment key expired")
		return
	}
	if !invitation.Reusable && !invitation.UsedAt.IsZero() {
		writeError(w, http.StatusConflict, "enrollment key already used")
		return
	}

	now := time.Now().UTC()
	deviceToken, err := auth.NewAPIKey()
	if err != nil {
		s.logger.Error("generate device token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "device registration failed")
		return
	}
	deviceTokenHash, err := auth.HashAPIKey(deviceToken, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "device registration failed")
		return
	}
	device := models.Device{
		ID:          uuid.NewString(),
		UserID:      invitation.UserID,
		GroupID:     invitation.GroupID,
		Name:        strings.TrimSpace(req.Name),
		VirtualIP:   virtualIPFromUUID(uuid.New()),
		PublicKey:   strings.TrimSpace(req.PublicKey),
		TokenHash:   deviceTokenHash,
		DeviceToken: deviceToken,
		Status:      models.DeviceStatusOnline,
		LastSeenAt:  now,
		CreatedAt:   now,
		Fingerprint: strings.TrimSpace(req.Fingerprint),
	}
	if err := s.store.CreateDevice(r.Context(), device); err != nil {
		s.logger.Error("create device failed", "err", err)
		writeError(w, http.StatusInternalServerError, "device registration failed")
		return
	}
	if !invitation.Reusable {
		if err := s.store.MarkInvitationUsed(r.Context(), invitation.ID); err != nil {
			s.logger.Error("mark invitation used failed", "err", err)
		}
	}
	relays, err := s.store.ListRelays(r.Context())
	if err != nil {
		s.logger.Warn("list relays for registration failed", "err", err)
	}

	writeJSON(w, http.StatusCreated, models.DeviceRegistrationResponse{
		Device:  device,
		Relays:  relays,
		Message: "device registered",
	})
}

func (s *Server) handleDeviceHealth(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	deviceToken := r.Header.Get(linkbitapi.HeaderDeviceToken)
	if deviceID == "" || deviceToken == "" {
		writeError(w, http.StatusUnauthorized, "device id and token are required")
		return
	}
	tokenHash, err := auth.HashAPIKey(deviceToken, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid device token")
		return
	}
	var report models.DeviceHealthReport
	if err := decodeJSON(w, r, &report); err != nil {
		writeError(w, http.StatusBadRequest, "invalid health payload")
		return
	}
	device, err := s.store.UpdateDeviceHealth(r.Context(), deviceID, tokenHash, report)
	if store.IsNotFound(err) {
		writeError(w, http.StatusUnauthorized, "invalid device token")
		return
	}
	if err != nil {
		s.logger.Error("device health update failed", "err", err)
		writeError(w, http.StatusInternalServerError, "health update failed")
		return
	}
	writeJSON(w, http.StatusOK, device)
}

func (s *Server) handleDeviceNetworkConfig(w http.ResponseWriter, r *http.Request) {
	device, ok := s.authenticateDevice(w, r)
	if !ok {
		return
	}
	devices, err := s.store.ListDevices(r.Context())
	if err != nil {
		s.logger.Error("list devices for network config failed", "err", err)
		writeError(w, http.StatusInternalServerError, "network config failed")
		return
	}
	policies, err := s.store.ListPolicies(r.Context())
	if err != nil {
		s.logger.Error("list policies for network config failed", "err", err)
		writeError(w, http.StatusInternalServerError, "network config failed")
		return
	}
	relays, err := s.store.ListRelays(r.Context())
	if err != nil {
		s.logger.Error("list relays for network config failed", "err", err)
		writeError(w, http.StatusInternalServerError, "network config failed")
		return
	}
	writeJSON(w, http.StatusOK, models.NetworkConfig{
		Device:   device,
		Peers:    allowedPeers(device, devices, policies),
		Policies: policies,
		Relays:   relays,
	})
}

func (s *Server) authenticateDevice(w http.ResponseWriter, r *http.Request) (models.Device, bool) {
	deviceID := r.PathValue("id")
	deviceToken := r.Header.Get(linkbitapi.HeaderDeviceToken)
	if deviceID == "" || deviceToken == "" {
		writeError(w, http.StatusUnauthorized, "device id and token are required")
		return models.Device{}, false
	}
	tokenHash, err := auth.HashAPIKey(deviceToken, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid device token")
		return models.Device{}, false
	}
	device, err := s.store.GetDeviceByIDAndTokenHash(r.Context(), deviceID, tokenHash)
	if store.IsNotFound(err) {
		writeError(w, http.StatusUnauthorized, "invalid device token")
		return models.Device{}, false
	}
	if err != nil {
		s.logger.Error("device authentication failed", "err", err)
		writeError(w, http.StatusInternalServerError, "device authentication failed")
		return models.Device{}, false
	}
	return device, true
}

func (s *Server) handleInvitationCreate(w http.ResponseWriter, r *http.Request) {
	var req models.InvitationCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid invitation payload")
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	if strings.TrimSpace(req.GroupID) == "" {
		req.GroupID = "default"
	}
	if _, err := s.store.GetUser(r.Context(), strings.TrimSpace(req.UserID)); err != nil {
		writeError(w, http.StatusBadRequest, "userId does not exist")
		return
	}
	if _, err := s.store.GetGroup(r.Context(), strings.TrimSpace(req.GroupID)); err != nil {
		writeError(w, http.StatusBadRequest, "groupId does not exist")
		return
	}
	expiresIn := time.Duration(req.ExpiresInSeconds) * time.Second
	if expiresIn <= 0 {
		expiresIn = 24 * time.Hour
	}
	if expiresIn > 30*24*time.Hour {
		writeError(w, http.StatusBadRequest, "expiration must be 30 days or less")
		return
	}

	token, err := auth.NewAPIKey()
	if err != nil {
		s.logger.Error("generate invitation failed", "err", err)
		writeError(w, http.StatusInternalServerError, "invitation creation failed")
		return
	}
	tokenHash, err := auth.HashAPIKey(token, s.cfg.APIKeyPepper)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invitation creation failed")
		return
	}
	now := time.Now().UTC()
	invitation := models.Invitation{
		ID:             uuid.NewString(),
		TokenHash:      tokenHash,
		UserID:         strings.TrimSpace(req.UserID),
		GroupID:        strings.TrimSpace(req.GroupID),
		Reusable:       req.Reusable,
		ExpiresAt:      now.Add(expiresIn),
		CreatedAt:      now,
		PlaintextToken: token,
	}
	if err := s.store.CreateInvitation(r.Context(), invitation); err != nil {
		s.logger.Error("persist invitation failed", "err", err)
		writeError(w, http.StatusInternalServerError, "invitation creation failed")
		return
	}
	writeJSON(w, http.StatusCreated, invitation)
}

func (s *Server) handlePolicyCreate(w http.ResponseWriter, r *http.Request) {
	var policy models.NetworkPolicy
	if err := decodeJSON(w, r, &policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy payload")
		return
	}
	if policy.ID == "" || policy.SourceID == "" || policy.TargetID == "" {
		writeError(w, http.StatusBadRequest, "policy id, sourceId, and targetId are required")
		return
	}
	now := time.Now().UTC()
	if policy.Name == "" {
		policy.Name = policy.SourceID + " to " + policy.TargetID
	}
	if policy.Protocol == "" {
		policy.Protocol = "tcp"
	}
	if policy.Ports == nil {
		policy.Ports = []string{"*"}
	}
	policy.CreatedAt = now
	policy.UpdatedAt = now

	if err := s.store.CreatePolicy(r.Context(), policy); err != nil {
		s.logger.Error("create policy failed", "err", err)
		writeError(w, http.StatusInternalServerError, "policy creation failed")
		return
	}
	writeJSON(w, http.StatusCreated, policy)
}

func (s *Server) handlePolicyList(w http.ResponseWriter, r *http.Request) {
	policies, err := s.store.ListPolicies(r.Context())
	if err != nil {
		s.logger.Error("list policies failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list policies failed")
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (s *Server) handlePolicyDelete(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policy id is required")
		return
	}
	if err := s.store.DeletePolicy(r.Context(), policyID); store.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	} else if err != nil {
		s.logger.Error("delete policy failed", "err", err)
		writeError(w, http.StatusInternalServerError, "policy deletion failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requireAPIKey(next http.Handler) http.Handler {
	return s.requireAPIKeyScopes(next, "admin")
}

func (s *Server) requireAPIKeyScopes(next http.Handler, scopes ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(linkbitapi.HeaderAPIKey)
		if !s.verifyAPIKeyScope(r, apiKey, scopes...) {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) verifyAPIKeyScope(r *http.Request, apiKey string, scopes ...string) bool {
	if scopeAllowed("admin", scopes) && auth.VerifyAPIKey(apiKey, s.adminDigest, s.cfg.APIKeyPepper) {
		return true
	}
	digest, err := auth.HashAPIKey(apiKey, s.cfg.APIKeyPepper)
	if err != nil {
		return false
	}
	stored, err := s.store.GetAPIKeyByDigest(r.Context(), digest)
	if err != nil {
		return false
	}
	if !scopeAllowed(stored.Scope, scopes) {
		return false
	}
	if err := s.store.TouchAPIKey(r.Context(), stored.ID); err != nil {
		s.logger.Warn("failed to touch api key", "err", err)
	}
	return true
}

func scopeAllowed(actual string, allowed []string) bool {
	for _, scope := range allowed {
		if actual == scope {
			return true
		}
	}
	return false
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func allowedPeers(device models.Device, devices []models.Device, policies []models.NetworkPolicy) []models.NetworkPeer {
	allowed := make(map[string]bool)
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		if policyAllows(policy.SourceID, device) {
			allowed[policy.TargetID] = true
		}
	}

	peers := make([]models.NetworkPeer, 0)
	for _, candidate := range devices {
		if candidate.ID == device.ID {
			continue
		}
		if allowed[candidate.ID] || allowed[candidate.GroupID] || allowed["*"] {
			peers = append(peers, models.NetworkPeer{
				ID:        candidate.ID,
				Name:      candidate.Name,
				VirtualIP: candidate.VirtualIP,
				PublicKey: candidate.PublicKey,
			})
		}
	}
	return peers
}

func policyAllows(selector string, device models.Device) bool {
	return selector == "*" || selector == device.ID || selector == device.GroupID
}

func virtualIPFromUUID(id uuid.UUID) string {
	bytes := id
	return fmt.Sprintf("100.96.%d.%d", bytes[0], bytes[1])
}
