package preset

import (
	"strings"

	"github.com/krackl1n/careeria/grpc/client/interceptor"
	"google.golang.org/grpc"
)

type Deps struct {
	Metrics        *interceptor.Metrics
	MethodTimeout  *interceptor.MethodTimeoutDeps
	Authentication *interceptor.AuthenticationDeps
}

type Preset struct {
	Unary  []grpc.UnaryClientInterceptor
	Stream []grpc.StreamClientInterceptor
}

func New(peerService string, deps Deps) Preset {
	if deps.Metrics == nil {
		panic("grpc client preset: nil metrics")
	}
	if strings.TrimSpace(peerService) == "" {
		panic("grpc client preset: empty peer service")
	}
	unary := make([]grpc.UnaryClientInterceptor, 0, 5)
	stream := make([]grpc.StreamClientInterceptor, 0, 5)
	unary = append(unary, interceptor.UnaryRequestID())
	stream = append(stream, interceptor.StreamRequestID())
	unary = append(unary, deps.Metrics.UnaryClientInterceptor(peerService))
	stream = append(stream, deps.Metrics.StreamClientInterceptor(peerService))
	if deps.MethodTimeout != nil {
		unary = append(unary, interceptor.UnaryMethodTimeout(*deps.MethodTimeout))
		stream = append(stream, interceptor.StreamMethodTimeout(*deps.MethodTimeout))
	}
	unary = append(unary, interceptor.UnaryValidation())
	stream = append(stream, interceptor.StreamValidation())
	if deps.Authentication != nil {
		unary = append(unary, interceptor.UnaryAuthentication(*deps.Authentication))
		stream = append(stream, interceptor.StreamAuthentication(*deps.Authentication))
	}
	return Preset{Unary: unary, Stream: stream}
}
