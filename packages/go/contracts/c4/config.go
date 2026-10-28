package c4

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type ComponentKind string

const (
	KindDomain    ComponentKind = "domain"
	KindTransport ComponentKind = "transport"
	KindAdapter   ComponentKind = "adapter"
	KindGeneric   ComponentKind = "generic"
)

type Config struct {
	Components []ComponentConfig `yaml:"components"`
}

type LayerConfig struct {
	Name     string   `yaml:"name"`
	Packages []string `yaml:"packages"`
	Role     string   `yaml:"role"`
}

type ComponentConfig struct {
	Name        string        `yaml:"name"`
	Slug        string        `yaml:"slug"`
	Container   string        `yaml:"container"`
	Title       string        `yaml:"title"`
	Description string        `yaml:"description"`
	Kind        ComponentKind `yaml:"kind"`
	OutputDir   string        `yaml:"output_dir"`

	// Domain / Application Core fields
	UsecasePkg     string   `yaml:"usecase_pkg"`
	DomainPkg      string   `yaml:"domain_pkg"`
	PolicyPkg      string   `yaml:"policy_pkg"`
	Subpackages    []string `yaml:"subpackages"`
	CrossDomainPkg string   `yaml:"cross_domain_pkg"`
	ServiceName    string   `yaml:"service_name"`
	Aggregate      string   `yaml:"aggregate"`

	// Transport layer fields (gRPC/HTTP handlers, servers, interceptors)
	TransportPkg   string `yaml:"transport_pkg"`
	InterceptorPkg string `yaml:"interceptor_pkg"`
	PortPkg        string `yaml:"port_pkg"`

	// Adapter / Infrastructure fields (repositories, clients, DB pools)
	AdapterPkg string `yaml:"adapter_pkg"`
	InfraPkg   string `yaml:"infra_pkg"`

	// Generic layer definitions for custom components
	Layers []LayerConfig `yaml:"layers"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	return &cfg, nil
}

