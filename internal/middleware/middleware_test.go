package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const method = "/albums.v1.AlbumsService/GetAlbum"

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f fakeStream) Context() context.Context { return f.ctx }

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestUnaryRecoveryTurnsPanicIntoInternal(t *testing.T) {
	handler := func(ctx context.Context, req any) (any, error) { panic("boom") }
	_, err := UnaryRecovery(quiet())(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected codes.Internal, got %v", err)
	}
}

func TestStreamRecoveryTurnsPanicIntoInternal(t *testing.T) {
	handler := func(srv any, ss grpc.ServerStream) error { panic("boom") }
	err := StreamRecovery(quiet())(nil, fakeStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: method}, handler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected codes.Internal, got %v", err)
	}
}

func TestUnaryLoggingRecordsMethodAndCode(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	handler := func(ctx context.Context, req any) (any, error) { return nil, status.Error(codes.NotFound, "nope") }
	_, _ = UnaryLogging(log)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
	for _, want := range []string{"kind=unary", "method=" + method, "code=NotFound", "duration="} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log %q should contain %q", buf.String(), want)
		}
	}
}

func TestStreamLoggingRecordsMethodAndCode(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	handler := func(srv any, ss grpc.ServerStream) error { return nil }
	_ = StreamLogging(log)(nil, fakeStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: method}, handler)
	for _, want := range []string{"kind=stream", "method=" + method, "code=OK"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log %q should contain %q", buf.String(), want)
		}
	}
}
