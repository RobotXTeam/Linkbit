package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/linkbit/linkbit/internal/models"
)

type HTTPRegistrationClient struct {
	controllerURL string
	publicKey     string
	fingerprint   string
	deviceID      string
	deviceToken   string
	client        *http.Client
}

func NewHTTPRegistrationClient(controllerURL string, publicKey string, fingerprint string) *HTTPRegistrationClient {
	return &HTTPRegistrationClient{
		controllerURL: strings.TrimRight(controllerURL, "/"),
		publicKey:     publicKey,
		fingerprint:   fingerprint,
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPRegistrationClient) ReportHealth(ctx context.Context, report models.DeviceHealthReport) (models.Device, error) {
	var out models.Device
	if c.controllerURL == "" || c.deviceID == "" || c.deviceToken == "" {
		return out, errors.New("controller url, device id, and device token are required")
	}
	body, err := json.Marshal(report)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.controllerURL+"/api/v1/devices/"+c.deviceID+"/health", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Linkbit-Device-Token", c.deviceToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return out, errors.New(resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *HTTPRegistrationClient) Register(ctx context.Context, enrollmentKey string, deviceName string) (models.DeviceRegistrationResponse, error) {
	var out models.DeviceRegistrationResponse
	if c.controllerURL == "" {
		return out, errors.New("controller url is required")
	}
	reqBody := models.DeviceRegistrationRequest{
		EnrollmentKey: enrollmentKey,
		Name:          deviceName,
		PublicKey:     c.publicKey,
		Fingerprint:   c.fingerprint,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.controllerURL+"/api/v1/devices/register", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return out, errors.New(resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	c.deviceID = out.Device.ID
	c.deviceToken = out.Device.DeviceToken
	return out, nil
}
