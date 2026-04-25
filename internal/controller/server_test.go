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
	handler := server.Handler()
	body := `{"id":"relay-1","name":"Relay 1","region":"ap-east","publicUrl":"https://relay.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/relays/register", bytes.NewBufferString(body))
	req.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	derpReq := httptest.NewRequest(http.MethodGet, "/api/v1/derp-map", nil)
	derpReq.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	derpRec := httptest.NewRecorder()
	handler.ServeHTTP(derpRec, derpReq)
	if derpRec.Code != http.StatusOK {
		t.Fatalf("derp map status = %d, want %d; body=%s", derpRec.Code, http.StatusOK, derpRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/relays/relay-1", nil)
	deleteReq.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body=%s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
}

func TestPersistentAPIKeyAuthenticates(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewBufferString(`{"name":"ops","scope":"admin"}`))
	createReq.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create key status = %d, want %d; body=%s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	var apiKey models.APIKey
	if err := json.NewDecoder(createRec.Body).Decode(&apiKey); err != nil {
		t.Fatalf("decode api key: %v", err)
	}
	if apiKey.PlaintextKey == "" || apiKey.Digest != "" {
		t.Fatalf("api key leaked wrong fields: %+v", apiKey)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/relays", nil)
	listReq.Header.Set(linkbitapi.HeaderAPIKey, apiKey.PlaintextKey)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
}

func TestInvitationRegistersDeviceOnce(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	createUser(t, handler, "user-1")
	createGroup(t, handler, "ops")

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
	var registration models.DeviceRegistrationResponse
	if err := json.NewDecoder(registerRec.Body).Decode(&registration); err != nil {
		t.Fatalf("decode registration: %v", err)
	}
	if registration.Device.DeviceToken == "" || registration.Device.TokenHash != "" {
		t.Fatalf("device leaked wrong token fields: %+v", registration.Device)
	}

	healthReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+registration.Device.ID+"/health", bytes.NewBufferString(`{"status":"online","latencyMs":8,"peersReachable":1,"peersTotal":1}`))
	healthReq.Header.Set(linkbitapi.HeaderDeviceToken, registration.Device.DeviceToken)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d; body=%s", healthRec.Code, http.StatusOK, healthRec.Body.String())
	}

	configReq := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+registration.Device.ID+"/network-config", nil)
	configReq.Header.Set(linkbitapi.HeaderDeviceToken, registration.Device.DeviceToken)
	configRec := httptest.NewRecorder()
	handler.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusOK {
		t.Fatalf("network config status = %d, want %d; body=%s", configRec.Code, http.StatusOK, configRec.Body.String())
	}

	registerAgainReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(registerBody))
	registerAgainRec := httptest.NewRecorder()
	handler.ServeHTTP(registerAgainRec, registerAgainReq)
	if registerAgainRec.Code != http.StatusConflict {
		t.Fatalf("second register status = %d, want %d; body=%s", registerAgainRec.Code, http.StatusConflict, registerAgainRec.Body.String())
	}
}

func TestInvitationRequiresKnownUserAndGroup(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invitations", bytes.NewBufferString(`{"userId":"missing","groupId":"missing","expiresInSeconds":3600}`))
	req.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func createUser(t *testing.T, handler http.Handler, id string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"id":"`+id+`","name":"Test User","role":"member"}`))
	req.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create user status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func createGroup(t *testing.T, handler http.Handler, id string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewBufferString(`{"id":"`+id+`","name":"Test Group"}`))
	req.Header.Set(linkbitapi.HeaderAPIKey, "test-admin-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
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
