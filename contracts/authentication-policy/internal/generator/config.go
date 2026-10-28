package generator

import (
	"errors"
	"strings"
)

const (
	authenticationExtensionName = "authentication.policy.v1.authentication"
	authenticationEnumName      = "authentication.policy.v1.AuthenticationPolicy"
	authenticationProtoPath     = "authentication/policy/v1/policy.proto"
	outputPath                  = "authentication/policy/v1/policy_map.pb.go"
)

var errInvalidPackageSegment = errors.New("exclude_package_segment must be one package segment")

type Config struct {
	ExcludePackageSegment string
}

func (c Config) Validate() error {
	segment := c.ExcludePackageSegment
	if segment != "" && (strings.Contains(segment, ".") || strings.TrimSpace(segment) != segment) {
		return errInvalidPackageSegment
	}
	return nil
}

func hasPackageSegment(packageName, segment string) bool {
	if segment == "" {
		return false
	}
	for _, part := range strings.Split(packageName, ".") {
		if part == segment {
			return true
		}
	}
	return false
}
