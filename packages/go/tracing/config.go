package tracing

import (
	"errors"
	"fmt"
	"time"
)

type ExporterType string

const (
	ExporterNone ExporterType = "none"
	ExporterGRPC ExporterType = "grpc"
	ExporterHTTP ExporterType = "http"
)

type Config struct {
	Enabled            bool
	ServiceName        string
	ServiceVersion     string
	Environment        string
	Exporter           ExporterType
	Endpoint           string
	Insecure           bool
	SamplingRatio      float64
	ExportTimeout      time.Duration
	ShutdownTimeout    time.Duration
	BatchTimeout       time.Duration
	MaxQueueSize       int
	MaxExportBatchSize int
}

func DefaultConfig(serviceName string) Config {
	return Config{
		Enabled:            true,
		ServiceName:        serviceName,
		ServiceVersion:     "unknown",
		Environment:        "local",
		Exporter:           ExporterGRPC,
		Endpoint:           "localhost:4317",
		Insecure:           true,
		SamplingRatio:      1,
		ExportTimeout:      5 * time.Second,
		ShutdownTimeout:    5 * time.Second,
		BatchTimeout:       5 * time.Second,
		MaxQueueSize:       2048,
		MaxExportBatchSize: 512,
	}
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ServiceName == "" {
		return errors.New("service name is required")
	}

	if c.Exporter == "" {
		return errors.New("exporter is required")
	}

	if c.Exporter != ExporterNone && c.Exporter != ExporterGRPC && c.Exporter != ExporterHTTP {
		return fmt.Errorf("unsupported exporter: %s", c.Exporter)
	}

	if c.Exporter != ExporterNone && c.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if c.SamplingRatio < 0 || c.SamplingRatio > 1 {
		return errors.New("sampling ratio must be between 0 and 1")
	}

	if c.ExportTimeout <= 0 {
		return errors.New("export timeout must be positive")
	}

	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}

	if c.BatchTimeout <= 0 {
		return errors.New("batch timeout must be positive")
	}

	if c.MaxQueueSize <= 0 {
		return errors.New("max queue size must be positive")
	}

	if c.MaxExportBatchSize <= 0 {
		return errors.New("max export batch size must be positive")
	}

	if c.MaxExportBatchSize > c.MaxQueueSize {
		return errors.New("max export batch size must not be greater than max queue size")
	}

	return nil
}
