package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linkbit/linkbit/internal/models"
)

func TestHTTPRegistrationClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/devices/register" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var req models.DeviceRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.EnrollmentKey != "invite" || req.Name != "device-1" || req.PublicKey != "pub" {
			t.Fatalf("unexpected request: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(models.DeviceRegistrationResponse{
			Device: models.Device{ID: "device-id", VirtualIP: "100.96.1.2"},
		})
	}))
	defer server.Close()

	client := NewHTTPRegistrationClient(server.URL, "pub", "fp")
	resp, err := client.Register(t.Context(), "invite", "device-1")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.Device.ID != "device-id" {
		t.Fatalf("device id = %s", resp.Device.ID)
	}
}
