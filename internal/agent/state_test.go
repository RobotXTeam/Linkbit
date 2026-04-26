package agent

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/linkbit/linkbit/internal/models"
)

func TestEnsureIdentityGeneratesAndPersistsWireGuardKeys(t *testing.T) {
	statePath := t.TempDir() + "/agent-state.json"

	first, err := EnsureIdentity(statePath, "", "", "")
	if err != nil {
		t.Fatalf("EnsureIdentity() error = %v", err)
	}
	if first.PrivateKey == "" || first.PublicKey == "" || first.Fingerprint == "" {
		t.Fatalf("identity has empty fields: %+v", first)
	}
	for name, key := range map[string]string{"private": first.PrivateKey, "public": first.PublicKey} {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatalf("%s key is not base64: %v", name, err)
		}
		if len(decoded) != 32 {
			t.Fatalf("%s key length = %d, want 32", name, len(decoded))
		}
	}

	second, err := EnsureIdentity(statePath, "", "", "")
	if err != nil {
		t.Fatalf("EnsureIdentity() second error = %v", err)
	}
	if second != first {
		t.Fatalf("identity was not reused: first=%+v second=%+v", first, second)
	}
}

func TestFileStateStorePreservesIdentityWhenSavingDevice(t *testing.T) {
	statePath := t.TempDir() + "/agent-state.json"
	identity, err := EnsureIdentity(statePath, "", "", "")
	if err != nil {
		t.Fatalf("EnsureIdentity() error = %v", err)
	}
	store := NewFileStateStore(statePath)
	if err := store.Save(models.DeviceRegistrationResponse{
		Device: models.Device{ID: "device-id", VirtualIP: "100.96.1.2", DeviceToken: "device-token", TokenHash: "must-not-persist"},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(raw) == 0 || strings.Contains(string(raw), "must-not-persist") {
		t.Fatalf("unexpected raw state: %s", string(raw))
	}
	loadedIdentity, err := EnsureIdentity(statePath, "", "", "")
	if err != nil {
		t.Fatalf("EnsureIdentity() reload error = %v", err)
	}
	if loadedIdentity != identity {
		t.Fatalf("identity changed after save: before=%+v after=%+v", identity, loadedIdentity)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Device.ID != "device-id" || loaded.Device.DeviceToken != "device-token" {
		t.Fatalf("loaded device = %+v", loaded.Device)
	}
}
