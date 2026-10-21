package generator

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func decodePolicy(options protoreflect.ProtoMessage, descriptors policyDescriptors) (string, error) {
	if options == nil {
		return "", errors.New("authentication policy is required")
	}
	known := false
	var knownValue protoreflect.EnumNumber
	options.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, current protoreflect.Value) bool {
		if field.IsExtension() && field.FullName() == descriptors.extension.FullName() {
			knownValue = current.Enum()
			known = true
		}
		return true
	})
	value, found, err := enumExtensionValue(options.ProtoReflect().GetUnknown(), protowire.Number(descriptors.extension.Number()))
	if err != nil {
		return "", err
	}
	if known && found {
		return "", errors.New("authentication policy is duplicated")
	}
	if known {
		value, found = knownValue, true
	}
	if !found {
		return "", errors.New("authentication policy is required")
	}
	enumValue := descriptors.enum.Values().ByNumber(value)
	if enumValue == nil {
		return "", fmt.Errorf("authentication policy value %d is unknown", value)
	}
	name := string(enumValue.Name())
	switch name {
	case "AUTHENTICATION_POLICY_PUBLIC", "AUTHENTICATION_POLICY_REQUIRED":
		return name, nil
	default:
		return "", fmt.Errorf("authentication policy %s is not allowed", name)
	}
}

func enumExtensionValue(unknown []byte, extensionNumber protowire.Number) (protoreflect.EnumNumber, bool, error) {
	var result protoreflect.EnumNumber
	found := false
	for len(unknown) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(unknown)
		if tagLength < 0 {
			return 0, false, protowire.ParseError(tagLength)
		}
		unknown = unknown[tagLength:]
		if number == extensionNumber {
			if found {
				return 0, false, errors.New("authentication policy is duplicated")
			}
			if wireType != protowire.VarintType {
				return 0, false, fmt.Errorf("authentication policy has unexpected wire type %d", wireType)
			}
			value, length := protowire.ConsumeVarint(unknown)
			if length < 0 {
				return 0, false, protowire.ParseError(length)
			}
			result = protoreflect.EnumNumber(value)
			found = true
			unknown = unknown[length:]
			continue
		}
		length := protowire.ConsumeFieldValue(number, wireType, unknown)
		if length < 0 {
			return 0, false, protowire.ParseError(length)
		}
		unknown = unknown[length:]
	}
	return result, found, nil
}
