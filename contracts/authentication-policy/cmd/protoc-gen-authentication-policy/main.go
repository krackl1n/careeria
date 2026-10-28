package main

import (
	"flag"

	"github.com/krackl1n/protoc-gen-authentication-policy/internal/generator"
	"google.golang.org/protobuf/compiler/protogen"
)

var excludePackageSegment = flag.String("exclude_package_segment", "", "omit packages containing this dot-separated segment")

func main() {
	protogen.Options{ParamFunc: flag.CommandLine.Set}.Run(func(plugin *protogen.Plugin) error {
		return generator.Generate(plugin, generator.Config{ExcludePackageSegment: *excludePackageSegment})
	})
}
