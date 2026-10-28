package swagger

import (
	"fmt"
	"strings"
)

type Config struct {
	Enabled           bool
	Title             string
	Path              string
	SpecificationPath string
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("swagger title is required")
	}
	if !strings.HasPrefix(c.Path, "/") || !strings.HasSuffix(c.Path, "/") {
		return fmt.Errorf("swagger path must start and end with a slash")
	}
	if !strings.HasPrefix(c.SpecificationPath, "/") {
		return fmt.Errorf("swagger specification path must start with a slash")
	}
	if c.Path == c.SpecificationPath {
		return fmt.Errorf("swagger path and specification path must differ")
	}

	return nil
}
