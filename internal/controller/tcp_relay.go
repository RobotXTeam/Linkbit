package controller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkbit/linkbit/internal/models"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

const tcpRelaySessionTTL = 2 * time.Minute

type tcpRelaySessionState struct {
	session models.TCPRelaySession
	client  net.Conn
	agent   net.Conn
}

type TCPRelayBroker struct {
	mu       sync.Mutex
	sessions map[string]*tcpRelaySessionState
	wake     chan struct{}
	logger   *slog.Logger
}

func NewTCPRelayBroker(logger *slog.Logger) *TCPRelayBroker {
	if logger == nil {
		logger = slog.Default()
	}
	return &TCPRelayBroker{sessions: make(map[string]*tcpRelaySessionState), wake: make(chan struct{}), logger: logger}
}

func (b *TCPRelayBroker) Create(source models.Device, target models.Device, port int) models.TCPRelaySession {
	now := time.Now().UTC()
	session := models.TCPRelaySession{
		ID:              uuid.NewString(),
		SourceDeviceID:  source.ID,
		TargetDeviceID:  target.ID,
		TargetName:      target.Name,
		TargetVirtualIP: target.VirtualIP,
		TargetPort:      port,
		Protocol:        "tcp",
		ExpiresAt:       now.Add(tcpRelaySessionTTL),
	}
	b.mu.Lock()
	b.sessions[session.ID] = &tcpRelaySessionState{session: session}
	b.cleanupLocked(now)
	b.signalLocked()
	b.mu.Unlock()
	return session
}

func (b *TCPRelayBroker) Pending(targetDeviceID string) []models.TCPRelaySession {
	now := time.Now().UTC()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupLocked(now)
	out := make([]models.TCPRelaySession, 0)
	for _, state := range b.sessions {
		if state.session.TargetDeviceID == targetDeviceID && state.agent == nil {
			out = append(out, state.session)
		}
	}
	return out
}

func (b *TCPRelayBroker) PendingWait(ctx context.Context, targetDeviceID string, wait time.Duration) []models.TCPRelaySession {
	pending := b.Pending(targetDeviceID)
	if len(pending) > 0 || wait <= 0 {
		return pending
	}
	b.mu.Lock()
	wake := b.wake
	b.mu.Unlock()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	case <-wake:
	}
	return b.Pending(targetDeviceID)
}

func (b *TCPRelayBroker) ValidateClient(sessionID string, sourceDeviceID string) error {
	return b.validate(sessionID, sourceDeviceID, true)
}

func (b *TCPRelayBroker) ValidateAgent(sessionID string, targetDeviceID string) error {
	return b.validate(sessionID, targetDeviceID, false)
}

func (b *TCPRelayBroker) AttachClient(sessionID string, sourceDeviceID string, conn net.Conn) error {
	return b.attach(sessionID, sourceDeviceID, conn, true)
}

func (b *TCPRelayBroker) AttachAgent(sessionID string, targetDeviceID string, conn net.Conn) error {
	return b.attach(sessionID, targetDeviceID, conn, false)
}

func (b *TCPRelayBroker) validate(sessionID string, deviceID string, client bool) error {
	now := time.Now().UTC()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupLocked(now)
	state, ok := b.sessions[sessionID]
	if !ok {
		return errors.New("relay session not found")
	}
	if client && state.session.SourceDeviceID != deviceID {
		return errors.New("relay session source mismatch")
	}
	if !client && state.session.TargetDeviceID != deviceID {
		return errors.New("relay session target mismatch")
	}
	if client && state.client != nil {
		return errors.New("relay client stream already attached")
	}
	if !client && state.agent != nil {
		return errors.New("relay agent stream already attached")
	}
	return nil
}

func (b *TCPRelayBroker) attach(sessionID string, deviceID string, conn net.Conn, client bool) error {
	var clientConn net.Conn
	var agentConn net.Conn

	b.mu.Lock()
	state, ok := b.sessions[sessionID]
	if !ok {
		b.mu.Unlock()
		_ = conn.Close()
		return errors.New("relay session not found")
	}
	if time.Now().UTC().After(state.session.ExpiresAt) {
		delete(b.sessions, sessionID)
		b.mu.Unlock()
		_ = conn.Close()
		return errors.New("relay session expired")
	}
	if client {
		if state.session.SourceDeviceID != deviceID || state.client != nil {
			b.mu.Unlock()
			_ = conn.Close()
			return errors.New("relay client stream rejected")
		}
		state.client = conn
	} else {
		if state.session.TargetDeviceID != deviceID || state.agent != nil {
			b.mu.Unlock()
			_ = conn.Close()
			return errors.New("relay agent stream rejected")
		}
		state.agent = conn
	}
	if state.client != nil && state.agent != nil {
		clientConn = state.client
		agentConn = state.agent
		delete(b.sessions, sessionID)
	}
	b.mu.Unlock()

	if clientConn != nil && agentConn != nil {
		go proxyTCPRelay(clientConn, agentConn)
	}
	return nil
}

func (b *TCPRelayBroker) cleanupLocked(now time.Time) {
	for id, state := range b.sessions {
		if now.After(state.session.ExpiresAt) {
			if state.client != nil {
				_ = state.client.Close()
			}
			if state.agent != nil {
				_ = state.agent.Close()
			}
			delete(b.sessions, id)
		}
	}
}

func (b *TCPRelayBroker) signalLocked() {
	close(b.wake)
	b.wake = make(chan struct{})
}

func proxyTCPRelay(a net.Conn, b net.Conn) {
	var once sync.Once
	closeBoth := func() {
		_ = a.Close()
		_ = b.Close()
	}
	go func() {
		_, _ = io.Copy(a, b)
		once.Do(closeBoth)
	}()
	go func() {
		_, _ = io.Copy(b, a)
		once.Do(closeBoth)
	}()
}

func (s *Server) handleTCPRelaySessionCreate(w http.ResponseWriter, r *http.Request) {
	var req models.TCPRelaySessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid relay session payload")
		return
	}
	source, ok := s.authenticateDeviceCredentials(w, r, strings.TrimSpace(req.SourceDeviceID))
	if !ok {
		return
	}
	if req.Protocol != "" && req.Protocol != "tcp" {
		writeError(w, http.StatusBadRequest, "only tcp relay sessions are supported")
		return
	}
	if req.TargetPort < 1 || req.TargetPort > 65535 {
		writeError(w, http.StatusBadRequest, "targetPort must be between 1 and 65535")
		return
	}
	target, allowed, err := s.resolveRelayTarget(r.Context(), source, strings.TrimSpace(req.Target))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "network policy does not allow this target")
		return
	}
	session := s.relayBroker.Create(source, target, req.TargetPort)
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) handleTCPRelayPending(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get(linkbitapi.HeaderDeviceID))
	}
	if _, ok := s.authenticateDeviceCredentials(w, r, deviceID); !ok {
		return
	}
	wait := 0 * time.Second
	if r.URL.Query().Get("wait") != "" {
		wait = 25 * time.Second
	}
	writeJSON(w, http.StatusOK, s.relayBroker.PendingWait(r.Context(), deviceID, wait))
}

func (s *Server) handleTCPRelayClientStream(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimSpace(r.URL.Query().Get("sourceDeviceId"))
	if sourceID == "" {
		sourceID = strings.TrimSpace(r.Header.Get(linkbitapi.HeaderDeviceID))
	}
	if _, ok := s.authenticateDeviceCredentials(w, r, sourceID); !ok {
		return
	}
	sessionID := r.PathValue("id")
	if err := s.relayBroker.ValidateClient(sessionID, sourceID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	conn, ok := hijackRelayStream(w)
	if !ok {
		return
	}
	_ = s.relayBroker.AttachClient(sessionID, sourceID, conn)
}

func (s *Server) handleTCPRelayAgentStream(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get(linkbitapi.HeaderDeviceID))
	}
	if _, ok := s.authenticateDeviceCredentials(w, r, deviceID); !ok {
		return
	}
	sessionID := r.PathValue("id")
	if err := s.relayBroker.ValidateAgent(sessionID, deviceID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	conn, ok := hijackRelayStream(w)
	if !ok {
		return
	}
	_ = s.relayBroker.AttachAgent(sessionID, deviceID, conn)
}

func (s *Server) resolveRelayTarget(ctx context.Context, source models.Device, targetValue string) (models.Device, bool, error) {
	if targetValue == "" {
		return models.Device{}, false, errors.New("target is required")
	}
	devices, err := s.store.ListDevices(ctx)
	if err != nil {
		return models.Device{}, false, errors.New("list devices failed")
	}
	policies, err := s.store.ListPolicies(ctx)
	if err != nil {
		return models.Device{}, false, errors.New("list policies failed")
	}
	var target models.Device
	for _, device := range devices {
		if device.ID == targetValue || device.VirtualIP == targetValue || strings.EqualFold(device.Name, targetValue) {
			target = device
			break
		}
	}
	if target.ID == "" {
		return models.Device{}, false, errors.New("target device not found")
	}
	for _, peer := range allowedPeers(source, devices, policies) {
		if peer.ID == target.ID {
			return target, true, nil
		}
	}
	return target, false, nil
}

func hijackRelayStream(w http.ResponseWriter) (net.Conn, bool) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return nil, false
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, false
	}
	// A 101 response keeps the HTTP authentication envelope but hands the socket
	// to the relay broker for raw TCP bytes after this point.
	if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: linkbit-relay\r\n\r\n"); err != nil {
		_ = conn.Close()
		return nil, false
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, false
	}
	return conn, true
}
