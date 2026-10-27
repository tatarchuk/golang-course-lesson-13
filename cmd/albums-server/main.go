// albums-server is the gRPC AlbumsService of Lesson 13 (slide 41: listens on :50051).
//
//	go run ./cmd/albums-server                 # plaintext, local development only
//	go run ./cmd/albums-server -addr :50061
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "homework/gen/albums"
	"homework/internal/middleware"
	"homework/internal/server"
	"homework/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "albums-server:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", ":50051", "address the gRPC server listens on")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Interceptor chain (slide 33): recovery outermost so it also covers logging.
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.UnaryRecovery(log), middleware.UnaryLogging(log)),
		grpc.ChainStreamInterceptor(middleware.StreamRecovery(log), middleware.StreamLogging(log)),
	)
	st := store.New()
	pb.RegisterAlbumsServiceServer(srv, server.New(st, st, log))
	reflection.Register(srv) // lets grpcurl discover the service without the .proto file

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("albums server started", "addr", lis.Addr().String(), "service", "albums.v1.AlbumsService")
	log.Warn("TLS off: plaintext is for local development only (slide 33)")

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(lis) }()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// GracefulStop waits for open calls; an open stream never ends by itself, so after
	// 5 seconds the server stops it.
	log.Info("shutting down...")
	done := make(chan struct{})
	go func() { srv.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Warn("graceful stop timed out, forcing")
		srv.Stop()
	}
	log.Info("bye")
	return nil
}
