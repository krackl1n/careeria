package contracts_test

import (
	"strings"
	"testing"

	policyv1 "github.com/krackl1n/careeria/contracts/gen/authentication/policy/v1"
	_ "github.com/krackl1n/careeria/contracts/gen/careeria/identity/admin/v1"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestAdminRPCsAreNotExposedByPublicGateway(t *testing.T) {
	t.Parallel()

	found := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.Contains(string(file.Package()), ".admin.") {
			return true
		}

		services := file.Services()
		for serviceIndex := 0; serviceIndex < services.Len(); serviceIndex++ {
			methods := services.Get(serviceIndex).Methods()
			for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
				method := methods.Get(methodIndex)
				options := method.Options().(*descriptorpb.MethodOptions)
				if proto.HasExtension(options, annotations.E_Http) {
					t.Errorf("admin RPC %s must not declare an HTTP route", method.FullName())
				}

				fullMethod := "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
				if _, exists := policyv1.MethodAuthenticationPolicies[fullMethod]; exists {
					t.Errorf("admin RPC %s must not be present in public authentication policies", fullMethod)
				}
				found++
			}
		}
		return true
	})

	if found != 2 {
		t.Fatalf("found %d admin RPCs, want 2", found)
	}
}
