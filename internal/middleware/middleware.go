// Package middleware holds the server interceptors (slide 33): recovery turns a panic in a
// handler into codes.Internal instead of a crash, logging records every call with its
// method, gRPC code and duration. Each comes in a unary and a stream flavour, because the
// streaming call would otherwise bypass them.
package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryRecovery converts a panic in a unary handler into codes.Internal.
func UnaryRecovery(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return h(ctx, req)
	}
}

// StreamRecovery converts a panic in a streaming handler into codes.Internal.
func StreamRecovery(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, h grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return h(srv, ss)
	}
}

// UnaryLogging logs method, code, duration and the client address of a unary call.
func UnaryLogging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := h(ctx, req)
		logCall(log, ctx, "unary", info.FullMethod, start, err)
		return resp, err
	}
}

// StreamLogging logs a streaming call once it has finished.
func StreamLogging(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, h grpc.StreamHandler) error {
		start := time.Now()
		err := h(srv, ss)
		logCall(log, ss.Context(), "stream", info.FullMethod, start, err)
		return err
	}
}

func logCall(log *slog.Logger, ctx context.Context, kind, method string, start time.Time, err error) {
	remote := ""
	if p, ok := peer.FromContext(ctx); ok {
		remote = p.Addr.String()
	}
	log.Info("rpc",
		"kind", kind,
		"method", method,
		"code", status.Code(err).String(),
		"duration", time.Since(start).Round(time.Microsecond).String(),
		"remote", remote,
	)
}
