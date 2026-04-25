package relay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"tailscale.com/derp/derpserver"
	"tailscale.com/types/key"
)

type DERPHTTPService struct {
	addr   string
	logger *slog.Logger
	server *derpserver.Server
}

func NewDERPHTTPService(addr string, logger *slog.Logger) (*DERPHTTPService, error) {
	if addr == "" {
		return nil, errors.New("listen address is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DERPHTTPService{
		addr:   addr,
		logger: logger,
		server: derpserver.New(key.NewNode(), func(format string, args ...any) {
			logger.Debug("derp", "msg", logMessage(format, args...))
		}),
	}, nil
}

func (s *DERPHTTPService) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/derp", derpserver.Handler(s.server))
	mux.HandleFunc("/derp/probe", derpserver.ProbeHandler)
	mux.HandleFunc("/derp/latency-check", derpserver.ProbeHandler)
	mux.HandleFunc("/generate_204", derpserver.ServeNoContent)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	httpServer := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Warn("derp http shutdown failed", "err", err)
		}
	}()

	s.logger.Info("starting derp http service", "addr", s.addr)
	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return ctx.Err()
	}
	return err
}

func logMessage(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
