package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

type Validatable interface {
	Validate() error
}

type Options struct {
	File        string
	EnvPrefix   string
	EnvOnly     []string
	RequiredEnv []string
}

func Load(options Options, target Validatable) error {
	root, err := rootType(target)
	if err != nil {
		return err
	}

	file, err := validateFile(options.File)
	if err != nil {
		return err
	}

	prefix, err := normalizeEnvPrefix(options.EnvPrefix)
	if err != nil {
		return err
	}

	schema, err := newSchema(root, prefix)
	if err != nil {
		return fmt.Errorf("build config schema: %w", err)
	}

	policy, err := newPolicy(schema, options.EnvOnly, options.RequiredEnv)
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigFile(file)
	v.SetConfigType("yaml")
	v.AllowEmptyEnv(true)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file %q: %w", file, err)
	}

	if err := validateFileConfig(v, root); err != nil {
		return fmt.Errorf("validate config file %q: %w", file, err)
	}

	if err := validateEnvOnly(v, policy); err != nil {
		return err
	}

	if err := bindEnv(v, schema.fields); err != nil {
		return err
	}

	if err := validateRequiredEnv(policy); err != nil {
		return err
	}

	settings := effectiveSettings(v, schema.fields)

	if err := decode(settings, target); err != nil {
		return fmt.Errorf("decode effective config: %w", err)
	}

	if err := target.Validate(); err != nil {
		return fmt.Errorf("validate effective config: %w", err)
	}

	return nil
}

func rootType(target Validatable) (reflect.Type, error) {
	value := reflect.ValueOf(target)

	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, fmt.Errorf("config target must be a non-nil pointer to a struct")
	}

	typ := value.Elem().Type()

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config target must point to a struct, got %s", typ)
	}

	return typ, nil
}

func validateFile(value string) (string, error) {
	file := strings.TrimSpace(value)

	if file == "" {
		return "", fmt.Errorf("config file is required")
	}

	switch strings.ToLower(filepath.Ext(file)) {
	case ".yaml", ".yml":
		return file, nil
	default:
		return "", fmt.Errorf("config file %q must have .yaml or .yml extension", file)
	}
}

func normalizeEnvPrefix(value string) (string, error) {
	prefix := strings.ToUpper(strings.Trim(strings.TrimSpace(value), "_"))

	if prefix == "" {
		return "", fmt.Errorf("environment prefix is required")
	}

	for _, char := range prefix {
		switch {
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '_':
		default:
			return "", fmt.Errorf(
				"environment prefix %q contains invalid character %q",
				prefix,
				char,
			)
		}
	}

	return prefix, nil
}

func bindEnv(v *viper.Viper, fields []field) error {
	for _, field := range fields {
		if !field.envSupported {
			continue
		}

		if err := v.BindEnv(field.key, field.env); err != nil {
			return fmt.Errorf(
				"bind environment variable %q to %q: %w",
				field.env,
				field.key,
				err,
			)
		}
	}

	return nil
}

func effectiveSettings(v *viper.Viper, fields []field) map[string]any {
	result := make(map[string]any)

	for _, field := range fields {
		if !v.IsSet(field.key) {
			continue
		}

		setValue(result, field.path, v.Get(field.key))
	}

	return result
}

func setValue(target map[string]any, path []string, value any) {
	current := target

	for _, part := range path[:len(path)-1] {
		next, exists := current[part]
		if !exists {
			child := make(map[string]any)
			current[part] = child
			current = child
			continue
		}

		current = next.(map[string]any)
	}

	current[path[len(path)-1]] = value
}

func validateEnvOnly(v *viper.Viper, policy policy) error {
	for _, field := range policy.envOnly {
		if v.InConfig(field.key) {
			return fmt.Errorf(
				"config key %q must not be present in YAML",
				field.key,
			)
		}
	}

	return nil
}

func validateRequiredEnv(policy policy) error {
	for _, field := range policy.requiredEnv {
		value, exists := os.LookupEnv(field.env)

		if !exists || value == "" {
			return fmt.Errorf(
				"required environment variable %q for config key %q is not set",
				field.env,
				field.key,
			)
		}
	}

	return nil
}
