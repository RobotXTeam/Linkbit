package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/linkbit/linkbit/internal/models"
)

type StateStore interface {
	Load() (models.DeviceRegistrationResponse, error)
	Save(models.DeviceRegistrationResponse) error
}

type FileStateStore struct {
	path string
}

func NewFileStateStore(path string) *FileStateStore {
	return &FileStateStore{path: path}
}

func (s *FileStateStore) Load() (models.DeviceRegistrationResponse, error) {
	var state models.DeviceRegistrationResponse
	if s == nil || s.path == "" {
		return state, errors.New("state path is required")
	}
	file, err := os.Open(s.path)
	if err != nil {
		return state, err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return state, err
	}
	if state.Device.ID == "" || state.Device.DeviceToken == "" {
		return state, errors.New("agent state is missing device credentials")
	}
	return state, nil
}

func (s *FileStateStore) Save(state models.DeviceRegistrationResponse) error {
	if s == nil || s.path == "" {
		return errors.New("state path is required")
	}
	if state.Device.ID == "" || state.Device.DeviceToken == "" {
		return errors.New("device id and token are required")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	state.Device.TokenHash = ""
	state.Message = "device state"
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".agent-state-*")
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
	if err := encoder.Encode(struct {
		models.DeviceRegistrationResponse
		SavedAt time.Time `json:"savedAt"`
	}{
		DeviceRegistrationResponse: state,
		SavedAt:                    time.Now().UTC(),
	}); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}
