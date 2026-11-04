package grpcserver

import "google.golang.org/grpc"

type Option func(*options)

type options struct {
	serverOptions      []grpc.ServerOption
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(o *options) {
		o.unaryInterceptors = append(o.unaryInterceptors, interceptors...)
	}
}

func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(o *options) {
		o.streamInterceptors = append(o.streamInterceptors, interceptors...)
	}
}

func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(o *options) {
		o.serverOptions = append(o.serverOptions, opts...)
	}
}
