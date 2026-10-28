package c4

import (
	"go/parser"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatType(t *testing.T) {
	expr, err := parser.ParseExpr("[]string")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := FormatType(expr); got != "List<string>" {
		t.Errorf("expected List<string>, got %s", got)
	}

	exprMap, err := parser.ParseExpr("map[string]int")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := FormatType(exprMap); got != "Map<string, int>" {
		t.Errorf("expected Map<string, int>, got %s", got)
	}
}

func TestContainsTypeName(t *testing.T) {
	cases := []struct {
		s        string
		typeName string
		expected bool
	}{
		{"InterviewSummary", "Interview", false},
		{"[]*Interview", "Interview", true},
		{"map[string]Interview", "Interview", true},
		{"Interview", "Interview", true},
		{"OtherType", "Interview", false},
	}

	for _, c := range cases {
		if got := ContainsTypeName(c.s, c.typeName); got != c.expected {
			t.Errorf("ContainsTypeName(%q, %q) = %v; want %v", c.s, c.typeName, got, c.expected)
		}
	}
}

func TestCleanRecvType(t *testing.T) {
	if got := CleanRecvType("*Service"); got != "Service" {
		t.Errorf("CleanRecvType(*Service) = %s; want Service", got)
	}
	if got := CleanRecvType("Service"); got != "Service" {
		t.Errorf("CleanRecvType(Service) = %s; want Service", got)
	}
}

func TestAnalyzeAndGenerate(t *testing.T) {
	// Find root of repo
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "../../../.."))

	cfg := ComponentConfig{
		Name:        "Offer Service",
		Slug:        "hiring-offer-service",
		Container:   "Hiring Service",
		Title:       "Offer Service (Hiring Domain)",
		Description: "Test description",
		UsecasePkg:  filepath.Join(repoRoot, "services/hiring/internal/usecase/offer"),
		DomainPkg:   filepath.Join(repoRoot, "services/hiring/internal/domain/offer"),
		ServiceName: "HiringOfferService",
		Aggregate:   "JobOffer",
	}

	model, err := AnalyzeComponent(cfg)
	if err != nil {
		t.Fatalf("AnalyzeComponent failed: %v", err)
	}

	if model.Service.Name != "HiringOfferService" {
		t.Errorf("expected service name HiringOfferService, got %s", model.Service.Name)
	}
	if model.Aggregate.Name != "JobOffer" {
		t.Errorf("expected aggregate JobOffer, got %s", model.Aggregate.Name)
	}

	puml := GeneratePlantUML(model)
	if len(puml) == 0 {
		t.Errorf("expected non-empty PlantUML string")
	}

	md := GenerateMarkdown(model)
	if len(md) == 0 {
		t.Errorf("expected non-empty Markdown string")
	}
}

func TestTransportComponent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "../../../.."))

	cfg := ComponentConfig{
		Name:           "Identity Registration Transport",
		Slug:           "identity-registration-transport",
		Container:      "Identity Service",
		Kind:           KindTransport,
		Title:          "gRPC Registration Transport",
		Description:    "Inbound gRPC handlers and interceptors",
		TransportPkg:   filepath.Join(repoRoot, "services/identity/internal/transport/grpc/registration"),
		InterceptorPkg: filepath.Join(repoRoot, "packages/go/grpc/interceptor"),
	}

	model, err := AnalyzeComponent(cfg)
	if err != nil {
		t.Fatalf("AnalyzeComponent transport failed: %v", err)
	}

	if len(model.Layers) < 2 {
		t.Fatalf("expected at least 2 layers, got %d", len(model.Layers))
	}

	puml := GeneratePlantUML(model)
	if len(puml) == 0 {
		t.Errorf("expected non-empty PlantUML string")
	}

	md := GenerateMarkdown(model)
	if len(md) == 0 {
		t.Errorf("expected non-empty Markdown string")
	}
}

func TestAdapterComponent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "../../../.."))

	cfg := ComponentConfig{
		Name:        "Identity Postgres Repository",
		Slug:        "identity-postgres-repo",
		Container:   "Identity Service",
		Kind:        KindAdapter,
		Title:       "Postgres Identity Repository",
		Description: "Database adapter for identity persistence",
		AdapterPkg:  filepath.Join(repoRoot, "services/identity/internal/repository/identity"),
	}

	model, err := AnalyzeComponent(cfg)
	if err != nil {
		t.Fatalf("AnalyzeComponent adapter failed: %v", err)
	}

	if len(model.Layers) < 1 {
		t.Fatalf("expected at least 1 layer, got %d", len(model.Layers))
	}

	puml := GeneratePlantUML(model)
	if len(puml) == 0 {
		t.Errorf("expected non-empty PlantUML string")
	}

	md := GenerateMarkdown(model)
	if len(md) == 0 {
		t.Errorf("expected non-empty Markdown string")
	}
}

func TestGenericComponentWithLayers(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "../../../.."))

	cfg := ComponentConfig{
		Name:        "Custom Component",
		Slug:        "custom-component",
		Container:   "Test Container",
		Kind:        KindGeneric,
		Title:       "Custom Multi-Layer Component",
		Description: "Arbitrary layers",
		Layers: []LayerConfig{
			{
				Name:     "Transport Layer",
				Packages: []string{filepath.Join(repoRoot, "services/identity/internal/transport/grpc/registration")},
				Role:     "Handler",
			},
			{
				Name:     "Middlewares",
				Packages: []string{filepath.Join(repoRoot, "packages/go/grpc/interceptor")},
				Role:     "Middleware",
			},
		},
	}

	model, err := AnalyzeComponent(cfg)
	if err != nil {
		t.Fatalf("AnalyzeComponent generic failed: %v", err)
	}

	if len(model.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(model.Layers))
	}

	puml := GeneratePlantUML(model)
	if len(puml) == 0 {
		t.Errorf("expected non-empty PlantUML string")
	}

	md := GenerateMarkdown(model)
	if len(md) == 0 {
		t.Errorf("expected non-empty Markdown string")
	}
}
