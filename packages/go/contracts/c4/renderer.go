package c4

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RenderSVG generates an SVG file from a PlantUML (.puml) file using the cached JAR or Docker fallback.
func RenderSVG(pumlPath string) error {
	jarPath := "docs/architecture/c4-plant-uml/.cache/plantuml-1.2026.8.jar"
	if customJar := os.Getenv("PLANTUML_JAR"); customJar != "" {
		jarPath = customJar
	}

	if _, err := os.Stat(jarPath); err == nil {
		cmd := exec.Command("java", "-Djava.awt.headless=true", "-jar", jarPath, "-failfast2", "-tsvg", pumlPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", err, string(output))
		}
		return nil
	}

	cwd, _ := os.Getwd()
	cmd := exec.Command("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/workspace", cwd), "-w", "/workspace", "plantuml/plantuml:1.2026.8", "-failfast2", "-tsvg", pumlPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// GenerateComponent processes a single component configuration, parses the Go code,
// and outputs .puml, .svg, and .md files.
func GenerateComponent(comp ComponentConfig) error {
	fmt.Printf("Processing L4 component: %s (%s)...\n", comp.Name, comp.Slug)

	model, err := AnalyzeComponent(comp)
	if err != nil {
		return fmt.Errorf("analyze %s: %w", comp.Name, err)
	}

	if err := os.MkdirAll(comp.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output dir %s: %w", comp.OutputDir, err)
	}

	pumlContent := GeneratePlantUML(model)
	pumlPath := filepath.Join(comp.OutputDir, comp.Slug+".puml")
	if err := os.WriteFile(pumlPath, []byte(pumlContent), 0644); err != nil {
		return fmt.Errorf("write puml file: %w", err)
	}
	fmt.Printf("  -> Generated %s\n", pumlPath)

	if err := RenderSVG(pumlPath); err != nil {
		fmt.Fprintf(os.Stderr, "  -> Warning: failed to render SVG for %s: %v\n", comp.Slug, err)
	} else {
		svgPath := filepath.Join(comp.OutputDir, comp.Slug+".svg")
		fmt.Printf("  -> Generated %s\n", svgPath)
	}

	mdContent := GenerateMarkdown(model)
	mdPath := filepath.Join(comp.OutputDir, comp.Slug+".md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return fmt.Errorf("write md file: %w", err)
	}
	fmt.Printf("  -> Generated %s\n", mdPath)

	return nil
}

// GenerateFromConfig reads the YAML configuration file and generates C4 L4 artifacts
// for all defined components.
func GenerateFromConfig(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	for _, comp := range cfg.Components {
		if err := GenerateComponent(comp); err != nil {
			return err
		}
	}

	fmt.Println("Done generating C4 L4 specifications!")
	return nil
}

