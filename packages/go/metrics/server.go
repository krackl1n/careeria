package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Server struct {
	enabled bool
	server  *http.Server
}

func NewServer(cfg Config, m *Metrics) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate metrics server config: %w", err)
	}

	if !cfg.Enabled {
		return &Server{
			enabled: false,
		}, nil
	}

	if m == nil {
		return nil, errors.New("metrics is required")
	}

	mux := http.NewServeMux()

	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/healthz", healthzHandler)

	return &Server{
		enabled: true,
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           mux,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		},
	}, nil
}

func (s *Server) Name() string {
	return "metrics-server"
}

func (s *Server) Start() error {
	if s == nil || !s.enabled || s.server == nil {
		return nil
	}

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve metrics server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || !s.enabled || s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown metrics server: %w", err)
	}

	return nil
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte("ok"))
}
