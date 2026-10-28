package authn

import (
	"context"

	policyv1 "github.com/krackl1n/careeria/contracts/gen/authentication/policy/v1"
	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryPolicy() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		request any,
		reply any,
		connection *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		options ...grpc.CallOption,
	) error {
		if err := requirePrincipal(ctx, method); err != nil {
			return err
		}
		return invoker(ctx, method, request, reply, connection, options...)
	}
}

func StreamPolicy() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		description *grpc.StreamDesc,
		connection *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		options ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if err := requirePrincipal(ctx, method); err != nil {
			return nil, err
		}
		return streamer(ctx, description, connection, method, options...)
	}
}

func requirePrincipal(ctx context.Context, method string) error {
	policy, ok := policyv1.MethodAuthenticationPolicies[method]
	if ok && policy == policyv1.AuthenticationPolicy_AUTHENTICATION_POLICY_PUBLIC {
		return nil
	}

	principal, ok := ctxmeta.PrincipalFromContext(ctx)
	if !ok || principal.SubjectID == "" {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	return nil
}
