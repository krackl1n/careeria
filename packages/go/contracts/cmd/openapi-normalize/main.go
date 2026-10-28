package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	validate "github.com/envoyproxy/protoc-gen-validate/validate"
	openapiv2 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	_ "github.com/krackl1n/careeria/contracts/gen/careeria/api/v1"
	_ "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	errorResponseRef = "#/responses/Error"
)

var schemaPackages = map[protoreflect.FullName]struct{}{
	"careeria.api.v1":      {},
	"careeria.identity.v1": {},
}

var publicOperationIDs = map[string]struct{}{
	"Authorize":                 {},
	"ContinueAuthorization":     {},
	"ExchangeToken":             {},
	"RevokeToken":               {},
	"GetAuthenticationMethods":  {},
	"GetAuthenticationContext":  {},
	"AuthenticateWithPassword":  {},
	"VerifyEmail":               {},
	"VerifyPhone":               {},
	"SkipMFAEnrollment":         {},
	"GrantAuthorizationConsent": {},
	"DenyAuthorizationConsent":  {},
	"GetOpenIDConfiguration":    {},
	"GetJSONWebKeySet":          {},
	"RegisterIdentity":          {},
	"ResolveInvitationToken":    {},
	"ResolveInvitationCode":     {},
}

var formOperationIDs = map[string]struct{}{
	"ExchangeToken": {},
	"RevokeToken":   {},
}

func main() {
	file := flag.String("file", "openapi/careeria.swagger.json", "OpenAPI v2 specification to normalize")
	flag.Parse()

	if err := run(*file); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read OpenAPI specification: %w", err)
	}

	var specification map[string]any
	if err := json.Unmarshal(data, &specification); err != nil {
		return fmt.Errorf("decode OpenAPI specification: %w", err)
	}

	definitions, err := object(specification, "definitions")
	if err != nil {
		return err
	}
	if _, ok := definitions["v1ErrorResponse"]; !ok {
		return fmt.Errorf("ErrorResponse definition is missing")
	}
	responses := map[string]any{
		"Error": map[string]any{
			"description": "Ошибка API Gateway.",
			"schema": map[string]any{
				"$ref": "#/definitions/v1ErrorResponse",
			},
		},
	}
	specification["responses"] = responses

	messages, methods := contractDescriptors()
	for _, message := range messages {
		if definition, ok := definitions["v1"+string(message.Name())].(map[string]any); ok {
			applyMessageSchema(definition, message)
		}
	}

	paths, err := object(specification, "paths")
	if err != nil {
		return err
	}
	for path, rawPath := range paths {
		pathItem, ok := rawPath.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q is not an object", path)
		}
		for method, rawOperation := range pathItem {
			if method == "parameters" {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				return fmt.Errorf("operation %s %s is not an object", method, path)
			}
			operationResponses, ok := operation["responses"].(map[string]any)
			if !ok {
				return fmt.Errorf("responses are missing for %s %s", method, path)
			}
			delete(operationResponses, "Error")
			operationResponses["default"] = map[string]any{"$ref": errorResponseRef}

			operationID, _ := operation["operationId"].(string)
			if _, public := publicOperationIDs[operationID]; public {
				operation["security"] = []any{}
			}
			methodDescriptor := methods[operationID]
			if methodDescriptor == nil {
				continue
			}
			applyOperationParameters(operation, definitions, methodDescriptor.Input())
			if _, form := formOperationIDs[operationID]; form {
				normalizeFormParameters(operation, definitions, methodDescriptor.Input())
			}
		}
	}

	result, err := json.MarshalIndent(specification, "", "  ")
	if err != nil {
		return fmt.Errorf("encode OpenAPI specification: %w", err)
	}
	result = append(result, '\n')
	if err := os.WriteFile(file, result, 0o644); err != nil {
		return fmt.Errorf("write OpenAPI specification: %w", err)
	}
	return nil
}

func normalizeFormParameters(operation, definitions map[string]any, input protoreflect.MessageDescriptor) {
	parameters, _ := operation["parameters"].([]any)
	var bodySchema map[string]any
	var remaining []any
	for _, rawParameter := range parameters {
		parameter, ok := rawParameter.(map[string]any)
		if !ok || parameter["in"] != "body" {
			remaining = append(remaining, rawParameter)
			continue
		}
		bodySchema, _ = parameter["schema"].(map[string]any)
	}
	if bodySchema == nil {
		return
	}
	if reference, _ := bodySchema["$ref"].(string); reference != "" {
		bodySchema, _ = definitions[strings.TrimPrefix(reference, "#/definitions/")].(map[string]any)
	}
	if bodySchema == nil {
		return
	}

	properties, _ := bodySchema["properties"].(map[string]any)
	requiredNames := make(map[string]struct{})
	if required, ok := bodySchema["required"].([]any); ok {
		for _, rawName := range required {
			if name, ok := rawName.(string); ok {
				requiredNames[name] = struct{}{}
			}
		}
	}

	formParameters := make([]any, 0, input.Fields().Len()+len(remaining))
	fields := input.Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		name := field.JSONName()
		property, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		parameter := make(map[string]any, len(property)+3)
		for key, value := range property {
			parameter[key] = value
		}
		parameter["in"] = "formData"
		parameter["name"] = name
		_, required := requiredNames[name]
		parameter["required"] = required
		formParameters = append(formParameters, parameter)
	}
	operation["parameters"] = append(formParameters, remaining...)
	if oneof, ok := bodySchema["x-oneof"]; ok {
		operation["x-oneof"] = oneof
	}
}

func contractDescriptors() ([]protoreflect.MessageDescriptor, map[string]protoreflect.MethodDescriptor) {
	var messages []protoreflect.MessageDescriptor
	methods := make(map[string]protoreflect.MethodDescriptor)

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if _, ok := schemaPackages[file.Package()]; !ok {
			return true
		}
		collectMessages(file.Messages(), &messages)
		services := file.Services()
		for index := 0; index < services.Len(); index++ {
			serviceMethods := services.Get(index).Methods()
			for methodIndex := 0; methodIndex < serviceMethods.Len(); methodIndex++ {
				method := serviceMethods.Get(methodIndex)
				operationID := string(method.Name())
				options, _ := method.Options().(*descriptorpb.MethodOptions)
				if options != nil && proto.HasExtension(options, openapiv2.E_Openapiv2Operation) {
					if operation, ok := proto.GetExtension(options, openapiv2.E_Openapiv2Operation).(*openapiv2.Operation); ok && operation.GetOperationId() != "" {
						operationID = operation.GetOperationId()
					}
				}
				methods[operationID] = method
			}
		}
		return true
	})

	return messages, methods
}

func collectMessages(source protoreflect.MessageDescriptors, target *[]protoreflect.MessageDescriptor) {
	for index := 0; index < source.Len(); index++ {
		message := source.Get(index)
		*target = append(*target, message)
		collectMessages(message.Messages(), target)
	}
}

func applyOperationParameters(operation, definitions map[string]any, input protoreflect.MessageDescriptor) {
	parameters, _ := operation["parameters"].([]any)
	for _, rawParameter := range parameters {
		parameter, ok := rawParameter.(map[string]any)
		if !ok {
			continue
		}
		name, _ := parameter["name"].(string)
		if name == "body" {
			schema, _ := parameter["schema"].(map[string]any)
			if schema == nil {
				continue
			}
			if reference, _ := schema["$ref"].(string); reference != "" {
				definitionName := strings.TrimPrefix(reference, "#/definitions/")
				if definition, ok := definitions[definitionName].(map[string]any); ok {
					applyMessageSchema(definition, input)
				}
				continue
			}
			applyMessageSchema(schema, input)
			continue
		}

		field := fieldByJSONName(input.Fields(), name)
		if field != nil {
			applyFieldRules(parameter, field)
		}
	}
}

func applyMessageSchema(schema map[string]any, message protoreflect.MessageDescriptor) {
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		return
	}
	fields := message.Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		property, ok := properties[field.JSONName()].(map[string]any)
		if !ok {
			continue
		}
		applyFieldRules(property, field)
		normalizeDefault(property)
	}

	var oneofGroups []any
	oneofs := message.Oneofs()
	for index := 0; index < oneofs.Len(); index++ {
		oneof := oneofs.Get(index)
		if oneof.IsSynthetic() {
			continue
		}
		options, _ := oneof.Options().(*descriptorpb.OneofOptions)
		required := false
		if options != nil && proto.HasExtension(options, validate.E_Required) {
			required = extensionBool(proto.GetExtension(options, validate.E_Required))
		}
		var names []string
		oneofFields := oneof.Fields()
		for fieldIndex := 0; fieldIndex < oneofFields.Len(); fieldIndex++ {
			name := oneofFields.Get(fieldIndex).JSONName()
			if _, exists := properties[name]; exists {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			oneofGroups = append(oneofGroups, map[string]any{
				"fields":   names,
				"required": required,
			})
		}
	}
	if len(oneofGroups) > 0 {
		schema["x-oneof"] = oneofGroups
	}
}

func normalizeDefault(schema map[string]any) {
	defaultValue, ok := schema["default"].(string)
	if !ok {
		return
	}

	switch schema["type"] {
	case "boolean":
		switch defaultValue {
		case "true":
			schema["default"] = true
		case "false":
			schema["default"] = false
		}
	}
}

func extensionBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case *bool:
		return value != nil && *value
	default:
		return false
	}
}

func applyFieldRules(schema map[string]any, field protoreflect.FieldDescriptor) {
	options, _ := field.Options().(*descriptorpb.FieldOptions)
	if options == nil || !proto.HasExtension(options, validate.E_Rules) {
		return
	}
	rules, ok := proto.GetExtension(options, validate.E_Rules).(*validate.FieldRules)
	if !ok || rules == nil {
		return
	}

	applyStringRules(schema, rules.GetString_())

	if bytesRules := rules.GetBytes(); bytesRules != nil {
		if bytesRules.MinLen != nil {
			schema["x-min-bytes"] = *bytesRules.MinLen
		}
		if bytesRules.MaxLen != nil {
			schema["x-max-bytes"] = *bytesRules.MaxLen
		}
	}

	if repeatedRules := rules.GetRepeated(); repeatedRules != nil {
		if repeatedRules.MinItems != nil {
			schema["minItems"] = *repeatedRules.MinItems
		}
		if repeatedRules.MaxItems != nil {
			schema["maxItems"] = *repeatedRules.MaxItems
		}
		if repeatedRules.Unique != nil {
			schema["uniqueItems"] = *repeatedRules.Unique
		}
		if items, ok := schema["items"].(map[string]any); ok && repeatedRules.Items != nil {
			applyRules(items, repeatedRules.Items)
			applyEnumRules(items, field, repeatedRules.Items.GetEnum())
		}
	}

	applyEnumRules(schema, field, rules.GetEnum())
	if boolRules := rules.GetBool(); boolRules != nil && boolRules.Const != nil {
		schema["enum"] = []bool{*boolRules.Const}
	}

	applyNumericRules(schema, rules)
}

func applyEnumRules(schema map[string]any, field protoreflect.FieldDescriptor, rules *validate.EnumRules) {
	if rules == nil || field.Kind() != protoreflect.EnumKind {
		return
	}

	included := make(map[int32]bool, len(rules.In))
	for _, number := range rules.In {
		included[number] = true
	}
	excluded := make(map[int32]bool, len(rules.NotIn))
	for _, number := range rules.NotIn {
		excluded[number] = true
	}

	values := field.Enum().Values()
	allowed := make([]string, 0, values.Len())
	for index := 0; index < values.Len(); index++ {
		value := values.Get(index)
		number := int32(value.Number())
		if len(included) > 0 && !included[number] {
			continue
		}
		if excluded[number] {
			continue
		}
		allowed = append(allowed, string(value.Name()))
	}
	if len(allowed) == 0 {
		return
	}

	delete(schema, "$ref")
	schema["type"] = "string"
	schema["enum"] = allowed
	if defaultValue, ok := schema["default"].(string); ok {
		validDefault := false
		for _, value := range allowed {
			if value == defaultValue {
				validDefault = true
				break
			}
		}
		if !validDefault {
			delete(schema, "default")
		}
	}
}

func applyRules(schema map[string]any, rules *validate.FieldRules) {
	if rules == nil {
		return
	}
	applyStringRules(schema, rules.GetString_())
}

func applyStringRules(schema map[string]any, rules *validate.StringRules) {
	if rules == nil {
		return
	}
	if rules.Len != nil {
		schema["minLength"] = *rules.Len
		schema["maxLength"] = *rules.Len
	}
	if rules.MinLen != nil {
		schema["minLength"] = *rules.MinLen
	}
	if rules.MaxLen != nil {
		schema["maxLength"] = *rules.MaxLen
	}
	if rules.MinBytes != nil {
		schema["x-min-bytes"] = *rules.MinBytes
	}
	if rules.MaxBytes != nil {
		schema["x-max-bytes"] = *rules.MaxBytes
	}
	if rules.Pattern != nil {
		schema["pattern"] = *rules.Pattern
	}
	if rules.GetUuid() {
		schema["format"] = "uuid"
	}
	if rules.GetEmail() {
		schema["format"] = "email"
	}
	if rules.GetUri() {
		schema["format"] = "uri"
	}
	if rules.GetIp() {
		schema["format"] = "ip"
	}
	if rules.Const != nil {
		schema["enum"] = []string{*rules.Const}
	} else if len(rules.In) > 0 {
		schema["enum"] = rules.In
	}
}

func applyNumericRules(schema map[string]any, rules *validate.FieldRules) {
	setMinimum := func(value any, exclusive bool) {
		if schema["type"] == "string" {
			schema["x-minimum"] = value
			if exclusive {
				schema["x-exclusive-minimum"] = true
			}
			return
		}
		schema["minimum"] = value
		if exclusive {
			schema["exclusiveMinimum"] = true
		}
	}
	setMaximum := func(value any, exclusive bool) {
		if schema["type"] == "string" {
			schema["x-maximum"] = value
			if exclusive {
				schema["x-exclusive-maximum"] = true
			}
			return
		}
		schema["maximum"] = value
		if exclusive {
			schema["exclusiveMaximum"] = true
		}
	}

	if numeric := rules.GetInt32(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
	if numeric := rules.GetInt64(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
	if numeric := rules.GetUint32(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
	if numeric := rules.GetUint64(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
	if numeric := rules.GetFloat(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
	if numeric := rules.GetDouble(); numeric != nil {
		if numeric.Gte != nil {
			setMinimum(*numeric.Gte, false)
		}
		if numeric.Gt != nil {
			setMinimum(*numeric.Gt, true)
		}
		if numeric.Lte != nil {
			setMaximum(*numeric.Lte, false)
		}
		if numeric.Lt != nil {
			setMaximum(*numeric.Lt, true)
		}
	}
}

func fieldByJSONName(fields protoreflect.FieldDescriptors, name string) protoreflect.FieldDescriptor {
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if field.JSONName() == name || string(field.Name()) == name {
			return field
		}
	}
	return nil
}

func object(parent map[string]any, key string) (map[string]any, error) {
	result, ok := parent[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is missing or is not an object", key)
	}
	return result, nil
}
