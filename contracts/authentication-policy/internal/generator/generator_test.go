package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateProducesSortedMapAndExcludesAdmin(t *testing.T) {
	request := fixtureRequest()
	plugin := newPlugin(t, request)
	if err := Generate(plugin, Config{ExcludePackageSegment: "admin"}); err != nil {
		t.Fatal(err)
	}
	response := plugin.Response()
	if response.GetError() != "" || len(response.File) != 1 {
		t.Fatalf("unexpected plugin response: %+v", response)
	}
	if got := response.File[0].GetName(); got != outputPath {
		t.Fatalf("output path = %q", got)
	}
	content := response.File[0].GetContent()
	alpha := strings.Index(content, `"/sample.api.v1.ThingService/Alpha"`)
	zeta := strings.Index(content, `"/sample.api.v1.ThingService/Zeta"`)
	if alpha < 0 || zeta < 0 || alpha >= zeta {
		t.Fatalf("map is incomplete or not sorted:\n%s", content)
	}
	if strings.Contains(content, "sample.admin.v1") {
		t.Fatalf("admin RPC leaked into map:\n%s", content)
	}
	if !strings.Contains(content, "AuthenticationPolicy_AUTHENTICATION_POLICY_PUBLIC") || !strings.Contains(content, "AuthenticationPolicy_AUTHENTICATION_POLICY_REQUIRED") {
		t.Fatalf("policy enum constants missing:\n%s", content)
	}
}

func TestGenerateRejectsMissingAndInvalidPolicy(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  []byte
		want string
	}{
		{name: "missing", want: "authentication policy is required"},
		{name: "unspecified", raw: policyWire(0), want: "not allowed"},
		{name: "unknown", raw: policyWire(99), want: "unknown"},
		{name: "duplicate", raw: append(policyWire(1), policyWire(2)...), want: "duplicated"},
		{name: "wrong wire type", raw: protowire.AppendString(protowire.AppendTag(nil, 50001, protowire.BytesType), "bad"), want: "unexpected wire type"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := fixtureRequest()
			request.ProtoFile[2].Service[0].Method[0].Options.ProtoReflect().SetUnknown(test.raw)
			plugin := newPlugin(t, request)
			err := Generate(plugin, Config{ExcludePackageSegment: "admin"})
			if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), "/sample.api.v1.ThingService/Zeta") {
				t.Fatalf("error = %v, want RPC and %q", err, test.want)
			}
		})
	}
}

func TestEnumExtensionValueRejectsMalformedWire(t *testing.T) {
	if _, _, err := enumExtensionValue([]byte{0xff}, 50001); err == nil {
		t.Fatal("malformed protobuf wire data was accepted")
	}
	if _, _, err := enumExtensionValue(protowire.AppendTag(nil, 50001, protowire.VarintType), 50001); err == nil {
		t.Fatal("truncated enum value was accepted")
	}
	if _, _, err := enumExtensionValue(protowire.AppendTag(nil, 42, protowire.BytesType), 50001); err == nil {
		t.Fatal("truncated unrelated field was accepted")
	}
	raw := protowire.AppendString(protowire.AppendTag(nil, 42, protowire.BytesType), "other")
	raw = append(raw, policyWire(2)...)
	value, found, err := enumExtensionValue(raw, 50001)
	if err != nil || !found || value != 2 {
		t.Fatalf("unrelated field affected policy: value=%d found=%v err=%v", value, found, err)
	}
}

func TestDecodePolicySupportsKnownExtension(t *testing.T) {
	plugin := newPlugin(t, fixtureRequest())
	descriptors, err := findPolicyDescriptors(plugin)
	if err != nil {
		t.Fatal(err)
	}
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, dynamicpb.NewExtensionType(descriptors.extension), protoreflect.EnumNumber(2))
	if got, err := decodePolicy(options, descriptors); err != nil || got != "AUTHENTICATION_POLICY_PUBLIC" {
		t.Fatalf("known extension: got=%q err=%v", got, err)
	}
	options.ProtoReflect().SetUnknown(policyWire(1))
	if _, err := decodePolicy(options, descriptors); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("known and unknown duplicate error = %v", err)
	}
	if _, err := decodePolicy(nil, descriptors); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("nil options error = %v", err)
	}
}

func TestDescriptorValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*pluginpb.CodeGeneratorRequest)
		want   string
	}{
		{name: "missing extension", mutate: func(request *pluginpb.CodeGeneratorRequest) {
			request.ProtoFile[1].Extension = nil
		}, want: "find authentication extension"},
		{name: "wrong extension kind", mutate: func(request *pluginpb.CodeGeneratorRequest) {
			request.ProtoFile[1].Extension[0].Type = descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()
			request.ProtoFile[1].Extension[0].TypeName = nil
		}, want: "must be an enum method option"},
		{name: "missing enum", mutate: func(request *pluginpb.CodeGeneratorRequest) {
			request.ProtoFile[1].EnumType = nil
		}, want: "build protobuf descriptor registry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plugin := newPlugin(t, fixtureRequest())
			test.mutate(plugin.Request)
			if _, err := findPolicyDescriptors(plugin); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("descriptor error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCollectRejectsDuplicateRPC(t *testing.T) {
	plugin := newPlugin(t, fixtureRequest())
	descriptors, err := findPolicyDescriptors(plugin)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range plugin.Files {
		if file.Desc.Path() == "sample/api/v1/service.proto" {
			plugin.Files = append(plugin.Files, file)
			break
		}
	}
	if _, err := collectPolicies(plugin, descriptors, Config{ExcludePackageSegment: "admin"}); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate RPC error = %v", err)
	}
}

func TestGenerateRejectsUnexcludedAdminAndEmptyInput(t *testing.T) {
	request := fixtureRequest()
	plugin := newPlugin(t, request)
	if err := Generate(plugin, Config{}); err == nil || !strings.Contains(err.Error(), "sample.admin.v1.AdminService") {
		t.Fatalf("unexcluded admin error = %v", err)
	}
	request.FileToGenerate = []string{authenticationProtoPath}
	plugin = newPlugin(t, request)
	if err := Generate(plugin, Config{}); err == nil || !strings.Contains(err.Error(), "no RPC authentication policies") {
		t.Fatalf("empty input error = %v", err)
	}
}

func TestGenerateRejectsInvalidConfigAndMissingSchema(t *testing.T) {
	plugin := newPlugin(t, fixtureRequest())
	if err := Generate(plugin, Config{ExcludePackageSegment: "admin.v1"}); err != errInvalidPackageSegment {
		t.Fatalf("config error = %v", err)
	}
	request := fixtureRequest()
	request.ProtoFile[1].Name = proto.String("other/policy.proto")
	request.ProtoFile[2].Dependency[0] = "other/policy.proto"
	request.ProtoFile[3].Dependency[0] = "other/policy.proto"
	plugin = newPlugin(t, request)
	if err := Generate(plugin, Config{ExcludePackageSegment: "admin"}); err == nil || !strings.Contains(err.Error(), "is not present") {
		t.Fatalf("missing schema error = %v", err)
	}
}

func TestHasPackageSegment(t *testing.T) {
	for _, test := range []struct {
		pkg, segment string
		want         bool
	}{
		{"sample.admin.v1", "admin", true},
		{"sample.administrator.v1", "admin", false},
		{"sample.admin.v1", "", false},
	} {
		if got := hasPackageSegment(test.pkg, test.segment); got != test.want {
			t.Errorf("hasPackageSegment(%q, %q) = %v", test.pkg, test.segment, got)
		}
	}
}

func newPlugin(t *testing.T, request *pluginpb.CodeGeneratorRequest) *protogen.Plugin {
	t.Helper()
	plugin, err := protogen.Options{}.New(request)
	if err != nil {
		t.Fatal(err)
	}
	return plugin
}

func policyWire(value uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(nil, 50001, protowire.VarintType), value)
}

func fixtureRequest() *pluginpb.CodeGeneratorRequest {
	policy := &descriptorpb.FileDescriptorProto{
		Name: proto.String(authenticationProtoPath), Package: proto.String("authentication.policy.v1"), Syntax: proto.String("proto3"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		Options:    &descriptorpb.FileOptions{GoPackage: proto.String("example.com/contracts/gen/authentication/policy/v1;policyv1")},
		EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("AuthenticationPolicy"), Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: proto.String("AUTHENTICATION_POLICY_UNSPECIFIED"), Number: proto.Int32(0)},
			{Name: proto.String("AUTHENTICATION_POLICY_REQUIRED"), Number: proto.Int32(1)},
			{Name: proto.String("AUTHENTICATION_POLICY_PUBLIC"), Number: proto.Int32(2)},
		}}},
		Extension: []*descriptorpb.FieldDescriptorProto{{
			Name: proto.String("authentication"), Number: proto.Int32(50001), Extendee: proto.String(".google.protobuf.MethodOptions"),
			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
			TypeName: proto.String(".authentication.policy.v1.AuthenticationPolicy"),
		}},
	}
	serviceFile := func(path, pkg, service string, methods ...*descriptorpb.MethodDescriptorProto) *descriptorpb.FileDescriptorProto {
		return &descriptorpb.FileDescriptorProto{
			Name: proto.String(path), Package: proto.String(pkg), Syntax: proto.String("proto3"),
			Dependency:  []string{authenticationProtoPath},
			Options:     &descriptorpb.FileOptions{GoPackage: proto.String("example.com/contracts/gen/" + strings.ReplaceAll(pkg, ".", "/") + ";" + strings.ReplaceAll(pkg, ".", ""))},
			MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Request")}, {Name: proto.String("Response")}},
			Service:     []*descriptorpb.ServiceDescriptorProto{{Name: proto.String(service), Method: methods}},
		}
	}
	method := func(name, pkg string, value uint64) *descriptorpb.MethodDescriptorProto {
		options := &descriptorpb.MethodOptions{}
		options.ProtoReflect().SetUnknown(policyWire(value))
		return &descriptorpb.MethodDescriptorProto{
			Name: proto.String(name), InputType: proto.String("." + pkg + ".Request"), OutputType: proto.String("." + pkg + ".Response"), Options: options,
		}
	}
	adminMethod := method("Internal", "sample.admin.v1", 1)
	adminMethod.Options.ProtoReflect().SetUnknown(nil)
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"sample/api/v1/service.proto", "sample/admin/v1/service.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto), policy,
			serviceFile("sample/api/v1/service.proto", "sample.api.v1", "ThingService", method("Zeta", "sample.api.v1", 1), method("Alpha", "sample.api.v1", 2)),
			serviceFile("sample/admin/v1/service.proto", "sample.admin.v1", "AdminService", adminMethod),
		},
	}
}
