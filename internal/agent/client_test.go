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
		switch r.URL.Path {
		case "/api/v1/devices/register":
			var req models.DeviceRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.EnrollmentKey != "invite" || req.Name != "device-1" || req.PublicKey != "pub" {
				t.Fatalf("unexpected request: %+v", req)
			}
			if req.Endpoint != "198.51.100.10:41641" {
				t.Fatalf("endpoint = %q, want %q", req.Endpoint, "198.51.100.10:41641")
			}
			_ = json.NewEncoder(w).Encode(models.DeviceRegistrationResponse{
				Device: models.Device{ID: "device-id", VirtualIP: "10.88.1.2", DeviceToken: "device-token"},
			})
		case "/api/v1/devices/device-id/health":
			if r.Header.Get("X-Linkbit-Device-Token") != "device-token" {
				t.Fatalf("missing device token")
			}
			_ = json.NewEncoder(w).Encode(models.Device{ID: "device-id", Status: models.DeviceStatusOnline})
		case "/api/v1/devices/device-id/network-config":
			if r.Header.Get("X-Linkbit-Device-Token") != "device-token" {
				t.Fatalf("missing device token")
			}
			_ = json.NewEncoder(w).Encode(models.NetworkConfig{
				Device: models.Device{ID: "device-id"},
				Peers:  []models.NetworkPeer{{ID: "peer-1", VirtualIP: "10.88.1.3"}},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHTTPRegistrationClient(server.URL, "pub", "fp", "198.51.100.10:41641")
	resp, err := client.Register(t.Context(), "invite", "device-1")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.Device.ID != "device-id" {
		t.Fatalf("device id = %s", resp.Device.ID)
	}
	health, err := client.ReportHealth(t.Context(), models.DeviceHealthReport{Status: models.DeviceStatusOnline})
	if err != nil {
		t.Fatalf("ReportHealth() error = %v", err)
	}
	if health.Status != models.DeviceStatusOnline {
		t.Fatalf("health status = %s", health.Status)
	}
	cfg, err := client.GetNetworkConfig(t.Context())
	if err != nil {
		t.Fatalf("GetNetworkConfig() error = %v", err)
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("peers = %d, want 1", len(cfg.Peers))
	}
}
