package controller

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
	sqlitestore "github.com/linkbit/linkbit/internal/store/sqlite"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

func TestRelayRegistrationRequiresAPIKey(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/relays/register", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRelayRegistration(t *testing.T) {
	server := newTestServer(t)
	body := `{"id":"relay-1","name":"Relay 1","region":"ap-east","publicUrl":"https://relay.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/relays/register", bytes.NewBufferString(body))
	req.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestInvitationRegistersDeviceOnce(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	inviteReq := httptest.NewRequest(http.MethodPost, "/api/v1/invitations", bytes.NewBufferString(`{"userId":"user-1","groupId":"ops","expiresInSeconds":3600}`))
	inviteReq.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	inviteRec := httptest.NewRecorder()
	handler.ServeHTTP(inviteRec, inviteReq)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("invite status = %d, want %d; body=%s", inviteRec.Code, http.StatusCreated, inviteRec.Body.String())
	}

	var invitation models.Invitation
	if err := json.NewDecoder(inviteRec.Body).Decode(&invitation); err != nil {
		t.Fatalf("decode invitation: %v", err)
	}
	if invitation.PlaintextToken == "" || invitation.TokenHash != "" {
		t.Fatalf("invitation leaked wrong fields: %+v", invitation)
	}

	registerBody := `{"enrollmentKey":"` + invitation.PlaintextToken + `","name":"laptop","publicKey":"wg-public-key","fingerprint":"fp-1"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(registerBody))
	registerRec := httptest.NewRecorder()
	handler.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body=%s", registerRec.Code, http.StatusCreated, registerRec.Body.String())
	}

	registerAgainReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(registerBody))
	registerAgainRec := httptest.NewRecorder()
	handler.ServeHTTP(registerAgainRec, registerAgainReq)
	if registerAgainRec.Code != http.StatusConflict {
		t.Fatalf("second register status = %d, want %d; body=%s", registerAgainRec.Code, http.StatusConflict, registerAgainRec.Body.String())
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	storage, err := sqlitestore.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New() error = %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	if err := storage.Migrate(t.Context()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	server, err := NewServer(config.ControllerConfig{
		ListenAddr:   ":0",
		DatabasePath: ":memory:",
		APIKeyPepper: []byte("test-pepper"),
	}, slog.Default(), "test-admin-key", storage)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}
