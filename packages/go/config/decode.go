package config

import (
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func validateFileConfig(v *viper.Viper, root reflect.Type) error {
	target := reflect.New(root).Interface()

	return v.UnmarshalExact(target, decoderOption)
}

func decode(input map[string]any, target any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           target,
		TagName:          "__config_disabled__",
		ErrorUnused:      true,
		WeaklyTypedInput: false,
		ZeroFields:       true,
		MapFieldName:     snakeCase,
		MatchName:        matchFieldName,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToBasicTypeHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	})
	if err != nil {
		return fmt.Errorf("create decoder: %w", err)
	}

	return decoder.Decode(input)
}

func decoderOption(config *mapstructure.DecoderConfig) {
	config.TagName = "__config_disabled__"
	config.ErrorUnused = true
	config.WeaklyTypedInput = false
	config.ZeroFields = true
	config.MapFieldName = snakeCase
	config.MatchName = matchFieldName
	config.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.StringToBasicTypeHookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
	)
}

func matchFieldName(mapKey, fieldName string) bool {
	return mapKey == snakeCase(fieldName)
}
