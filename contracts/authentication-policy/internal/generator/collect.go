package generator

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/compiler/protogen"
)

type policyEntry struct {
	fullMethod   string
	constantName string
}

func collectPolicies(plugin *protogen.Plugin, descriptors policyDescriptors, config Config) ([]policyEntry, error) {
	var entries []policyEntry
	seen := make(map[string]struct{})
	for _, file := range plugin.Files {
		if !file.Generate || hasPackageSegment(string(file.Desc.Package()), config.ExcludePackageSegment) {
			continue
		}
		for _, service := range file.Services {
			for _, method := range service.Methods {
				fullMethod := "/" + string(service.Desc.FullName()) + "/" + string(method.Desc.Name())
				policy, err := decodePolicy(method.Desc.Options(), descriptors)
				if err != nil {
					return nil, fmt.Errorf("RPC %s: %w", fullMethod, err)
				}
				if _, exists := seen[fullMethod]; exists {
					return nil, fmt.Errorf("RPC %s is duplicated", fullMethod)
				}
				seen[fullMethod] = struct{}{}
				entries = append(entries, policyEntry{fullMethod: fullMethod, constantName: policy})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].fullMethod < entries[j].fullMethod })
	return entries, nil
}
