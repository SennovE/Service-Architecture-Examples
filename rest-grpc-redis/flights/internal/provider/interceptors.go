package provider

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func APIKeyUnaryInterceptor(expectedKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		keys := md.Get("x-api-key")
		if len(keys) == 0 || keys[0] != expectedKey {
			return nil, status.Error(codes.Unauthenticated, "invalid api key")
		}
		return handler(ctx, req)
	}
}

func APIKeyStreamInterceptor(expected string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}
		keys := md.Get("x-api-key")
		if len(keys) == 0 || keys[0] != expected {
			return status.Error(codes.Unauthenticated, "invalid api key")
		}
		return handler(srv, ss)
	}
}
