package agent

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/linkbit/linkbit/internal/models"
	"golang.org/x/crypto/curve25519"
)

type StateStore interface {
	Load() (models.DeviceRegistrationResponse, error)
	Save(models.DeviceRegistrationResponse) error
}

var ErrNoDeviceCredentials = errors.New("agent state is missing device credentials")

type FileStateStore struct {
	path string
}

type Identity struct {
	PrivateKey  string
	PublicKey   string
	Fingerprint string
}

type agentState struct {
	models.DeviceRegistrationResponse
	WireGuardPrivateKey string    `json:"wireGuardPrivateKey,omitempty"`
	WireGuardPublicKey  string    `json:"wireGuardPublicKey,omitempty"`
	Fingerprint         string    `json:"fingerprint,omitempty"`
	SavedAt             time.Time `json:"savedAt"`
}

func NewFileStateStore(path string) *FileStateStore {
	return &FileStateStore{path: path}
}

func (s *FileStateStore) Load() (models.DeviceRegistrationResponse, error) {
	var state agentState
	if s == nil || s.path == "" {
		return state.DeviceRegistrationResponse, errors.New("state path is required")
	}
	state, err := readAgentState(s.path)
	if err != nil {
		return state.DeviceRegistrationResponse, err
	}
	if state.Device.ID == "" || state.Device.DeviceToken == "" {
		return state.DeviceRegistrationResponse, ErrNoDeviceCredentials
	}
	return state.DeviceRegistrationResponse, nil
}

func (s *FileStateStore) Save(state models.DeviceRegistrationResponse) error {
	if s == nil || s.path == "" {
		return errors.New("state path is required")
	}
	if state.Device.ID == "" || state.Device.DeviceToken == "" {
		return errors.New("device id and token are required")
	}
	existing, err := readAgentState(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	existing.DeviceRegistrationResponse = state
	existing.Device.TokenHash = ""
	existing.Message = "device state"
	return writeAgentState(s.path, existing)
}

func EnsureIdentity(path string, privateKey string, publicKey string, fingerprint string) (Identity, error) {
	var state agentState
	var err error
	if path != "" {
		state, err = readAgentState(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Identity{}, err
		}
	}
	identity := Identity{
		PrivateKey:  firstNonEmpty(privateKey, state.WireGuardPrivateKey),
		PublicKey:   firstNonEmpty(publicKey, state.WireGuardPublicKey),
		Fingerprint: firstNonEmpty(fingerprint, state.Fingerprint),
	}
	if identity.PrivateKey == "" {
		identity.PrivateKey, err = GenerateWireGuardPrivateKey()
		if err != nil {
			return Identity{}, err
		}
	}
	if identity.PublicKey == "" {
		identity.PublicKey, err = WireGuardPublicKey(identity.PrivateKey)
		if err != nil {
			return Identity{}, err
		}
	}
	if identity.Fingerprint == "" {
		identity.Fingerprint = fingerprintFromPublicKey(identity.PublicKey)
	}
	if path != "" {
		state.WireGuardPrivateKey = identity.PrivateKey
		state.WireGuardPublicKey = identity.PublicKey
		state.Fingerprint = identity.Fingerprint
		if err := writeAgentState(path, state); err != nil {
			return Identity{}, err
		}
	}
	return identity, nil
}

func GenerateWireGuardPrivateKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func WireGuardPublicKey(privateKey string) (string, error) {
	privateBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}
	if len(privateBytes) != 32 {
		return "", fmt.Errorf("wireguard private key must decode to 32 bytes, got %d", len(privateBytes))
	}
	publicBytes, err := curve25519.X25519(privateBytes, curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(publicBytes), nil
}

func readAgentState(path string) (agentState, error) {
	var state agentState
	file, err := os.Open(path)
	if err != nil {
		return state, err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return state, err
	}
	return state, nil
}

func writeAgentState(path string, state agentState) error {
	if path == "" {
		return errors.New("state path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	state.SavedAt = time.Now().UTC()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-state-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fingerprintFromPublicKey(publicKey string) string {
	sum := sha256.Sum256([]byte(publicKey))
	return "lb-" + base64.RawURLEncoding.EncodeToString(sum[:6])
}
