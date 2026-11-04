package config

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
)

var textUnmarshalerType = reflect.TypeOf(
	(*encoding.TextUnmarshaler)(nil),
).Elem()

type schema struct {
	fields []field
	byKey  map[string]field
}

type field struct {
	key          string
	path         []string
	env          string
	typ          reflect.Type
	envSupported bool
}

func newSchema(root reflect.Type, envPrefix string) (schema, error) {
	fields, err := collectFields(root, nil, envPrefix)
	if err != nil {
		return schema{}, err
	}

	byKey := make(map[string]field, len(fields))

	for _, field := range fields {
		previous, exists := byKey[field.key]
		if exists {
			return schema{}, fmt.Errorf(
				"config key collision: %q maps both %s and %s",
				field.key,
				strings.Join(previous.path, "."),
				strings.Join(field.path, "."),
			)
		}

		byKey[field.key] = field
	}

	return schema{
		fields: fields,
		byKey:  byKey,
	}, nil
}

func collectFields(
	typ reflect.Type,
	prefix []string,
	envPrefix string,
) ([]field, error) {
	var result []field

	for index := 0; index < typ.NumField(); index++ {
		structField := typ.Field(index)

		if structField.PkgPath != "" {
			return nil, fmt.Errorf(
				"config field %s.%s must be exported",
				typ,
				structField.Name,
			)
		}

		if structField.Anonymous {
			return nil, fmt.Errorf(
				"embedded config field %s.%s is not supported",
				typ,
				structField.Name,
			)
		}

		fieldType := structField.Type
		path := append(
			append([]string(nil), prefix...),
			snakeCase(structField.Name),
		)

		if unsupportedType(fieldType) {
			return nil, fmt.Errorf(
				"config field %q has unsupported type %s",
				strings.Join(path, "."),
				fieldType,
			)
		}

		if fieldType.Kind() == reflect.Struct &&
			!implementsTextUnmarshaler(fieldType) {
			children, err := collectFields(fieldType, path, envPrefix)
			if err != nil {
				return nil, err
			}

			result = append(result, children...)
			continue
		}

		key := strings.Join(path, ".")

		result = append(result, field{
			key:          key,
			path:         path,
			env:          envPrefix + "_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_")),
			typ:          fieldType,
			envSupported: supportsEnv(fieldType),
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("config struct %s has no exported fields", typ)
	}

	return result, nil
}

func unsupportedType(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Ptr,
		reflect.Interface,
		reflect.Func,
		reflect.Chan,
		reflect.UnsafePointer:
		return true
	default:
		return false
	}
}

func supportsEnv(typ reflect.Type) bool {
	if implementsTextUnmarshaler(typ) {
		return true
	}
	if typ.Kind() == reflect.Slice {
		return typ.Elem().Kind() == reflect.String
	}

	switch typ.Kind() {
	case reflect.Bool,
		reflect.String,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
		return true
	default:
		return false
	}
}

func implementsTextUnmarshaler(typ reflect.Type) bool {
	if typ.Implements(textUnmarshalerType) {
		return true
	}

	return reflect.PtrTo(typ).Implements(textUnmarshalerType)
}

type policy struct {
	envOnly     []field
	requiredEnv []field
}

func newPolicy(
	schema schema,
	envOnlyKeys []string,
	requiredEnvKeys []string,
) (policy, error) {
	result := policy{}

	envOnly := make(map[string]struct{}, len(envOnlyKeys))

	for _, rawKey := range envOnlyKeys {
		field, err := policyField(schema, rawKey, "env-only")
		if err != nil {
			return policy{}, err
		}

		if _, exists := envOnly[field.key]; exists {
			return policy{}, fmt.Errorf(
				"env-only config key %q is declared more than once",
				field.key,
			)
		}

		envOnly[field.key] = struct{}{}
		result.envOnly = append(result.envOnly, field)
	}

	required := make(map[string]struct{}, len(requiredEnvKeys))

	for _, rawKey := range requiredEnvKeys {
		field, err := policyField(schema, rawKey, "required environment")
		if err != nil {
			return policy{}, err
		}

		if _, exists := required[field.key]; exists {
			return policy{}, fmt.Errorf(
				"required environment config key %q is declared more than once",
				field.key,
			)
		}

		if _, exists := envOnly[field.key]; !exists {
			return policy{}, fmt.Errorf(
				"required environment config key %q must also be env-only",
				field.key,
			)
		}

		required[field.key] = struct{}{}
		result.requiredEnv = append(result.requiredEnv, field)
	}

	return result, nil
}

func policyField(
	schema schema,
	rawKey string,
	kind string,
) (field, error) {
	key := strings.TrimSpace(rawKey)

	if key == "" {
		return field{}, fmt.Errorf("%s config key is empty", kind)
	}

	value, exists := schema.byKey[key]
	if !exists {
		return field{}, fmt.Errorf(
			"%s config key %q does not exist in config schema",
			kind,
			key,
		)
	}

	if !value.envSupported {
		return field{}, fmt.Errorf(
			"%s config key %q has unsupported environment type %s",
			kind,
			key,
			value.typ,
		)
	}

	return value, nil
}
