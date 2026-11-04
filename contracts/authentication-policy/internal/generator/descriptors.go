package generator

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type policyDescriptors struct {
	extension protoreflect.ExtensionDescriptor
	enum      protoreflect.EnumDescriptor
}

func findPolicyDescriptors(plugin *protogen.Plugin) (policyDescriptors, error) {
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: plugin.Request.ProtoFile})
	if err != nil {
		return policyDescriptors{}, fmt.Errorf("build protobuf descriptor registry: %w", err)
	}
	extensionDescriptor, err := files.FindDescriptorByName(authenticationExtensionName)
	if err != nil {
		return policyDescriptors{}, fmt.Errorf("find authentication extension: %w", err)
	}
	extension, ok := extensionDescriptor.(protoreflect.ExtensionDescriptor)
	if !ok || extension.Kind() != protoreflect.EnumKind || extension.ContainingMessage().FullName() != "google.protobuf.MethodOptions" {
		return policyDescriptors{}, errors.New("authentication extension must be an enum method option")
	}
	enumDescriptor, err := files.FindDescriptorByName(authenticationEnumName)
	if err != nil {
		return policyDescriptors{}, fmt.Errorf("find authentication policy enum: %w", err)
	}
	enum, ok := enumDescriptor.(protoreflect.EnumDescriptor)
	if !ok || extension.Enum().FullName() != enum.FullName() {
		return policyDescriptors{}, errors.New("authentication extension does not use the policy enum")
	}
	return policyDescriptors{extension: extension, enum: enum}, nil
}

func generatedAuthenticationFile(plugin *protogen.Plugin) *protogen.File {
	for _, file := range plugin.Files {
		if file.Desc.Path() == authenticationProtoPath {
			return file
		}
	}
	return nil
}
