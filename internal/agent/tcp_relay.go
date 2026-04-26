package agent

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/linkbit/linkbit/internal/models"
	"github.com/linkbit/linkbit/pkg/linkbitapi"
)

type TCPRelayClient interface {
	CreateRelaySession(ctx context.Context, req models.TCPRelaySessionRequest) (models.TCPRelaySession, error)
	PendingRelaySessions(ctx context.Context) ([]models.TCPRelaySession, error)
	AttachRelayClient(ctx context.Context, sessionID string) (net.Conn, error)
	AttachRelayAgent(ctx context.Context, sessionID string) (net.Conn, error)
}

type TCPRelayTarget struct {
	client TCPRelayClient
	poll   time.Duration
	logger *slog.Logger
	active map[string]struct{}
	mu     sync.Mutex
}

type TCPForwarder struct {
	client TCPRelayClient
	logger *slog.Logger
}

func NewTCPRelayTarget(client TCPRelayClient, poll time.Duration, logger *slog.Logger) *TCPRelayTarget {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &TCPRelayTarget{client: client, poll: poll, logger: logger, active: make(map[string]struct{})}
}

func NewTCPForwarder(client TCPRelayClient, logger *slog.Logger) *TCPForwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &TCPForwarder{client: client, logger: logger}
}

func (t *TCPRelayTarget) Run(ctx context.Context) {
	for {
		if err := t.check(ctx); err != nil {
			t.logger.Warn("tcp relay poll failed", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(t.poll):
			}
			continue
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (t *TCPRelayTarget) check(ctx context.Context) error {
	sessions, err := t.client.PendingRelaySessions(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if !t.markActive(session.ID) {
			continue
		}
		go func(session models.TCPRelaySession) {
			defer t.clearActive(session.ID)
			if err := t.handle(ctx, session); err != nil {
				t.logger.Warn("tcp relay session failed", "session_id", session.ID, "target_port", session.TargetPort, "err", err)
			}
		}(session)
	}
	return nil
}

func (t *TCPRelayTarget) handle(ctx context.Context, session models.TCPRelaySession) error {
	if session.TargetPort < 1 || session.TargetPort > 65535 {
		return errors.New("invalid relay target port")
	}
	local, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(session.TargetPort)))
	if err != nil {
		return err
	}
	defer local.Close()

	relay, err := t.client.AttachRelayAgent(ctx, session.ID)
	if err != nil {
		return err
	}
	defer relay.Close()
	proxyConns(local, relay)
	return nil
}

func (t *TCPRelayTarget) markActive(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.active[id]; ok {
		return false
	}
	t.active[id] = struct{}{}
	return true
}

func (t *TCPRelayTarget) clearActive(id string) {
	t.mu.Lock()
	delete(t.active, id)
	t.mu.Unlock()
}

func (f *TCPForwarder) ListenAndServe(ctx context.Context, listenAddr string, target string, targetPort int, sourceDeviceID string) error {
	if listenAddr == "" {
		return errors.New("listen address is required")
	}
	if target == "" || targetPort < 1 || targetPort > 65535 {
		return errors.New("target and target port are required")
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	f.logger.Info("tcp relay forwarder listening", "listen", listener.Addr().String(), "target", target, "target_port", targetPort)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		go f.handle(ctx, conn, target, targetPort, sourceDeviceID)
	}
}

func (f *TCPForwarder) handle(ctx context.Context, local net.Conn, target string, targetPort int, sourceDeviceID string) {
	defer local.Close()
	session, err := f.client.CreateRelaySession(ctx, models.TCPRelaySessionRequest{
		SourceDeviceID: sourceDeviceID,
		Target:         target,
		TargetPort:     targetPort,
		Protocol:       "tcp",
	})
	if err != nil {
		f.logger.Warn("create tcp relay session failed", "err", err)
		return
	}
	relay, err := f.client.AttachRelayClient(ctx, session.ID)
	if err != nil {
		f.logger.Warn("attach tcp relay client failed", "session_id", session.ID, "err", err)
		return
	}
	defer relay.Close()
	proxyConns(local, relay)
}

func proxyConns(a net.Conn, b net.Conn) {
	var once sync.Once
	closeBoth := func() {
		_ = a.Close()
		_ = b.Close()
	}
	go func() {
		_, _ = io.Copy(a, b)
		once.Do(closeBoth)
	}()
	_, _ = io.Copy(b, a)
	once.Do(closeBoth)
}

func (c *HTTPRegistrationClient) CreateRelaySession(ctx context.Context, reqBody models.TCPRelaySessionRequest) (models.TCPRelaySession, error) {
	var out models.TCPRelaySession
	if reqBody.SourceDeviceID == "" {
		reqBody.SourceDeviceID = c.deviceID
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.controllerURL+"/api/v1/relay/sessions", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(linkbitapi.HeaderDeviceToken, c.deviceToken)
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

func (c *HTTPRegistrationClient) PendingRelaySessions(ctx context.Context) ([]models.TCPRelaySession, error) {
	var out []models.TCPRelaySession
	if c.deviceID == "" || c.deviceToken == "" {
		return out, errors.New("device credentials are required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.controllerURL+"/api/v1/relay/sessions/pending?wait=25&deviceId="+url.QueryEscape(c.deviceID), nil)
	if err != nil {
		return out, err
	}
	req.Header.Set(linkbitapi.HeaderDeviceToken, c.deviceToken)
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

func (c *HTTPRegistrationClient) AttachRelayClient(ctx context.Context, sessionID string) (net.Conn, error) {
	return c.dialRelayUpgrade(ctx, "/api/v1/relay/client/"+url.PathEscape(sessionID)+"?sourceDeviceId="+url.QueryEscape(c.deviceID))
}

func (c *HTTPRegistrationClient) AttachRelayAgent(ctx context.Context, sessionID string) (net.Conn, error) {
	return c.dialRelayUpgrade(ctx, "/api/v1/relay/agent/"+url.PathEscape(sessionID)+"?deviceId="+url.QueryEscape(c.deviceID))
}

func (c *HTTPRegistrationClient) dialRelayUpgrade(ctx context.Context, requestPath string) (net.Conn, error) {
	if c.controllerURL == "" || c.deviceID == "" || c.deviceToken == "" {
		return nil, errors.New("controller url and device credentials are required")
	}
	base, err := url.Parse(c.controllerURL)
	if err != nil {
		return nil, err
	}
	address := base.Host
	if !strings.Contains(address, ":") {
		if base.Scheme == "https" {
			address = net.JoinHostPort(address, "443")
		} else {
			address = net.JoinHostPort(address, "80")
		}
	}
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	if base.Scheme == "https" {
		raw, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		tlsConn := tls.Client(raw, &tls.Config{ServerName: base.Hostname(), MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = raw.Close()
			return nil, err
		}
		conn = tlsConn
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		defer conn.SetDeadline(time.Time{})
	}
	host := base.Host
	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: linkbit-relay\r\n%s: %s\r\n%s: %s\r\n\r\n",
		requestPath, host, linkbitapi.HeaderDeviceID, c.deviceID, linkbitapi.HeaderDeviceToken, c.deviceToken)
	if _, err := io.WriteString(conn, request); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !strings.Contains(statusLine, "101") {
		_ = conn.Close()
		return nil, fmt.Errorf("relay upgrade failed: %s", strings.TrimSpace(statusLine))
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if line == "\r\n" {
			break
		}
	}
	return &bufferedConn{Conn: conn, reader: reader}, nil
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader.Buffered() > 0 {
		return c.reader.Read(p)
	}
	return c.Conn.Read(p)
}

func ParseRelayTarget(value string) (string, int, error) {
	host, portValue, err := net.SplitHostPort(value)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, errors.New("target port must be between 1 and 65535")
	}
	if host == "" {
		return "", 0, errors.New("target host is required")
	}
	return host, port, nil
}
