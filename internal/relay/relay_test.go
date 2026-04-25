package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linkbit/linkbit/internal/config"
	"github.com/linkbit/linkbit/internal/models"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

func TestNodeRegisterAndHeartbeat(t *testing.T) {
	seenRegister := false
	seenHeartbeat := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(linkbitapi.HeaderAPIKey) != "relay-key" {
			t.Fatalf("missing relay api key")
		}
		var relay models.RelayNode
		if err := json.NewDecoder(r.Body).Decode(&relay); err != nil {
			t.Fatalf("decode relay: %v", err)
		}
		switch r.URL.Path {
		case "/api/v1/relays/register":
			seenRegister = relay.ID == "relay-1"
			w.WriteHeader(http.StatusCreated)
		case "/api/v1/relays/heartbeat":
			seenHeartbeat = relay.ID == "relay-1"
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	node, err := NewNode(config.RelayConfig{
		ControllerURL: server.URL,
		APIKey:        "relay-key",
		RelayID:       "relay-1",
		Name:          "Relay 1",
		PublicURL:     "https://relay.example.com",
		Region:        "test",
		Heartbeat:     time.Hour,
	}, nil, nil)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}

	if err := node.register(t.Context()); err != nil {
		t.Fatalf("register() error = %v", err)
	}
	if err := node.heartbeat(t.Context()); err != nil {
		t.Fatalf("heartbeat() error = %v", err)
	}
	if !seenRegister || !seenHeartbeat {
		t.Fatalf("seenRegister=%v seenHeartbeat=%v", seenRegister, seenHeartbeat)
	}
}
