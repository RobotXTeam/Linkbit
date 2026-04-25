package relay

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDERPHTTPServiceHealthAndProbe(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	service, err := NewDERPHTTPService(addr, nil)
	if err != nil {
		t.Fatalf("NewDERPHTTPService() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- service.Start(ctx) }()

	client := http.Client{Timeout: 3 * time.Second}
	waitForHTTP(t, &client, "http://"+addr+"/healthz")
	probeReq, err := http.NewRequest(http.MethodHead, "http://"+addr+"/derp/probe", nil)
	if err != nil {
		t.Fatalf("new probe request: %v", err)
	}
	probeResp, err := client.Do(probeReq)
	if err != nil {
		t.Fatalf("probe request: %v", err)
	}
	defer probeResp.Body.Close()
	if probeResp.StatusCode != http.StatusOK {
		t.Fatalf("probe status = %d, want %d", probeResp.StatusCode, http.StatusOK)
	}
	cancel()
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("DERP service did not stop")
	}
}

func waitForHTTP(t *testing.T, client *http.Client, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
}
