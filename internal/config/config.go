package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

type ControllerConfig struct {
	ListenAddr   string
	PublicURL    string
	DatabasePath string
	APIKeyPepper []byte
}

type RelayConfig struct {
	ControllerURL string
	APIKey        string
	RelayID       string
	Name          string
	PublicURL     string
	Region        string
	ListenAddr    string
	Heartbeat     time.Duration
}

type AgentConfig struct {
	ControllerURL string
	EnrollmentKey string
	DeviceName    string
	HealthEvery   time.Duration
}

func LoadController() (ControllerConfig, error) {
	cfg := ControllerConfig{
		ListenAddr:   getenv("LINKBIT_LISTEN_ADDR", linkbitapi.DefaultListenAddr),
		PublicURL:    os.Getenv("LINKBIT_PUBLIC_URL"),
		DatabasePath: getenv("LINKBIT_DATABASE_PATH", "linkbit.db"),
		APIKeyPepper: []byte(os.Getenv("LINKBIT_API_KEY_PEPPER")),
	}
	if len(cfg.APIKeyPepper) == 0 {
		return cfg, errors.New("LINKBIT_API_KEY_PEPPER is required")
	}
	return cfg, nil
}

func LoadRelay() RelayConfig {
	return RelayConfig{
		ControllerURL: os.Getenv("LINKBIT_CONTROLLER_URL"),
		APIKey:        os.Getenv("LINKBIT_API_KEY"),
		RelayID:       os.Getenv("LINKBIT_RELAY_ID"),
		Name:          getenv("LINKBIT_RELAY_NAME", "linkbit-relay"),
		PublicURL:     os.Getenv("LINKBIT_RELAY_PUBLIC_URL"),
		Region:        getenv("LINKBIT_RELAY_REGION", "default"),
		ListenAddr:    getenv("LINKBIT_LISTEN_ADDR", ":3478"),
		Heartbeat:     getenvDuration("LINKBIT_HEARTBEAT_SECONDS", 30*time.Second),
	}
}

func LoadAgent() AgentConfig {
	return AgentConfig{
		ControllerURL: os.Getenv("LINKBIT_CONTROLLER_URL"),
		EnrollmentKey: os.Getenv("LINKBIT_ENROLLMENT_KEY"),
		DeviceName:    getenv("LINKBIT_DEVICE_NAME", hostname()),
		HealthEvery:   getenvDuration("LINKBIT_HEALTH_SECONDS", 30*time.Second),
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "linkbit-device"
	}
	return name
}
