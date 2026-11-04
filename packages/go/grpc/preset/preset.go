package preset

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

type ServerPreset struct {
	Unary        []grpc.UnaryServerInterceptor
	Stream       []grpc.StreamServerInterceptor
	StatsHandler stats.Handler
}
