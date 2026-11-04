package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Component struct {
	metrics *Metrics
	server  *Server
}

func New(cfg Config) (*Component, error) {
	m, err := NewMetrics(cfg)
	if err != nil {
		return nil, fmt.Errorf("init metrics: %w", err)
	}

	server, err := NewServer(cfg, m)
	if err != nil {
		return nil, fmt.Errorf("init metrics server: %w", err)
	}

	return &Component{
		metrics: m,
		server:  server,
	}, nil
}

func (c *Component) Name() string {
	return "metrics-server"
}

func (c *Component) Start() error {
	if c == nil || c.server == nil {
		return nil
	}

	if err := c.server.Start(); err != nil {
		return fmt.Errorf("start metrics server: %w", err)
	}

	return nil
}

func (c *Component) Shutdown(ctx context.Context) error {
	if c == nil || c.server == nil {
		return nil
	}

	if err := c.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown metrics server: %w", err)
	}

	return nil
}

func (c *Component) Registry() *prometheus.Registry {
	if c == nil || c.metrics == nil {
		return nil
	}

	return c.metrics.Registry()
}

func (c *Component) Namespace() string {
	if c == nil || c.metrics == nil {
		return ""
	}

	return c.metrics.Namespace()
}

func (c *Component) Enabled() bool {
	if c == nil || c.metrics == nil {
		return false
	}

	return c.metrics.Enabled()
}

func (c *Component) Register(collector prometheus.Collector) error {
	if c == nil || c.metrics == nil {
		return nil
	}

	if err := c.metrics.Register(collector); err != nil {
		return fmt.Errorf("register metrics collector: %w", err)
	}

	return nil
}

func (c *Component) MustRegister(collectors ...prometheus.Collector) {
	if c == nil || c.metrics == nil {
		return
	}

	c.metrics.MustRegister(collectors...)
}

func (c *Component) Unregister(collector prometheus.Collector) bool {
	if c == nil || c.metrics == nil {
		return false
	}

	return c.metrics.Unregister(collector)
}
