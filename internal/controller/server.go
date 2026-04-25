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
	mux.Handle("GET /api/v1/overview", s.requireAPIKey(http.HandlerFunc(s.handleOverview)))
	mux.Handle("POST /api/v1/relays/register", s.requireAPIKey(http.HandlerFunc(s.handleRelayRegister)))
	mux.Handle("POST /api/v1/relays/heartbeat", s.requireAPIKey(http.HandlerFunc(s.handleRelayHeartbeat)))
	mux.Handle("GET /api/v1/relays", s.requireAPIKey(http.HandlerFunc(s.handleRelayList)))
	mux.Handle("GET /api/v1/devices", s.requireAPIKey(http.HandlerFunc(s.handleDeviceList)))
	mux.HandleFunc("POST /api/v1/devices/register", s.handleDeviceRegister)
	mux.Handle("POST /api/v1/invitations", s.requireAPIKey(http.HandlerFunc(s.handleInvitationCreate)))
	mux.Handle("POST /api/v1/policies", s.requireAPIKey(http.HandlerFunc(s.handlePolicyCreate)))
	mux.Handle("GET /api/v1/policies", s.requireAPIKey(http.HandlerFunc(s.handlePolicyList)))
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
	device := models.Device{
		ID:          uuid.NewString(),
		UserID:      invitation.UserID,
		GroupID:     invitation.GroupID,
		Name:        strings.TrimSpace(req.Name),
		VirtualIP:   virtualIPFromUUID(uuid.New()),
		PublicKey:   strings.TrimSpace(req.PublicKey),
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

func (s *Server) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(linkbitapi.HeaderAPIKey)
		if !auth.VerifyAPIKey(apiKey, s.adminDigest, s.cfg.APIKeyPepper) {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		next.ServeHTTP(w, r)
	})
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

func virtualIPFromUUID(id uuid.UUID) string {
	bytes := id
	return fmt.Sprintf("100.96.%d.%d", bytes[0], bytes[1])
}
