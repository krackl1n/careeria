package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Server struct {
	server *http.Server
}

func (s *Server) Name() string {
	return "http"
}

func New(cfg Config, handler http.Handler) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate http config: %w", err)
	}

	if handler == nil {
		return nil, errors.New("http handler is required")
	}

	return &Server{
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
	}, nil
}

func (s *Server) HTTP() *http.Server {
	return s.server
}

func (s *Server) Start() error {
	if err := s.server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http: %w", err)
	}

	return nil
}
