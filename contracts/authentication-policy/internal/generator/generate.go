package generator

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func Generate(plugin *protogen.Plugin, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	descriptors, err := findPolicyDescriptors(plugin)
	if err != nil {
		return err
	}
	entries, err := collectPolicies(plugin, descriptors, config)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("no RPC authentication policies found")
	}
	file := generatedAuthenticationFile(plugin)
	if file == nil {
		return fmt.Errorf("protobuf file %q is not present", authenticationProtoPath)
	}
	renderMap(plugin, file, entries)
	return nil
}
