// Package server implements AlbumsServiceServer (slides 16, 19, 20).
//
// Rules kept here (slides 45 and 46): every error leaves a handler as a gRPC status with
// the right code, never as a plain error value; the streaming handler works with
// stream.Context(), so a client that goes away is noticed and its subscription released;
// a stream that ends by design returns nil, which the client sees as a clean io.EOF.
package server

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "homework/gen/albums"
	"homework/internal/store"
)

// Catalog is what the unary calls need from the store.
type Catalog interface {
	Get(id int64) (store.Album, error)
	Create(in store.AlbumInput) (store.Album, error)
}

// Events is what the streaming call needs from the store.
type Events interface {
	Subscribe() (<-chan store.Event, func())
}

// Server is the AlbumsService. Embedding Unimplemented... keeps it forward compatible
// (slide 16): a new rpc in the contract does not break the build before it is implemented.
type Server struct {
	pb.UnimplementedAlbumsServiceServer

	catalog Catalog
	events  Events
	log     *slog.Logger
}

// New wires the service to its catalogue and event source (both satisfied by *store.Store).
func New(catalog Catalog, events Events, log *slog.Logger) *Server {
	return &Server{catalog: catalog, events: events, log: log}
}

// GetAlbum is the unary call that mirrors GET /albums/{id}.
func (s *Server) GetAlbum(_ context.Context, req *pb.GetAlbumRequest) (*pb.Album, error) {
	if req.GetId() < 1 {
		return nil, status.Error(codes.InvalidArgument, "id must be a positive integer")
	}
	a, err := s.catalog.Get(req.GetId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "album %d not found", req.GetId())
	}
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toProto(a), nil
}

// CreateAlbum is the unary call that mirrors POST /albums.
func (s *Server) CreateAlbum(_ context.Context, req *pb.CreateAlbumRequest) (*pb.Album, error) {
	a, err := s.catalog.Create(toInput(req))
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toProto(a), nil
}

// WatchAlbums is the server-streaming call: it subscribes first (no event is missed),
// forwards catalogue events, ends after max_events when that is set, and returns the
// context's status when the client cancels.
func (s *Server) WatchAlbums(req *pb.WatchAlbumsRequest, stream pb.AlbumsService_WatchAlbumsServer) error {
	ctx := stream.Context()
	events, cancel := s.events.Subscribe()
	defer cancel() // the handler owns the subscription: released on every exit, so nothing leaks

	limit := req.GetMaxEvents()
	s.log.Info("watch started", "max_events", limit)
	var sent int32
	for {
		select {
		case <-ctx.Done():
			s.log.Info("watch finished", "reason", "client gone / cancelled", "sent", sent)
			return status.FromContextError(ctx.Err()).Err()
		case ev, ok := <-events:
			if !ok {
				// Only this handler's cancel closes the channel, after the loop; a closed
				// channel here would be a store bug, never a normal end of the stream.
				s.log.Error("watch finished", "reason", "subscription closed unexpectedly", "sent", sent)
				return status.Error(codes.Internal, "event subscription closed")
			}
			if err := stream.Send(toEvent(ev)); err != nil {
				s.log.Info("watch finished", "reason", "send failed", "err", err, "sent", sent)
				return err
			}
			sent++
			if limit > 0 && sent >= limit {
				s.log.Info("watch finished", "reason", "max_events reached", "sent", sent)
				return nil // the client receives a clean io.EOF
			}
		}
	}
}

// toStatus maps the store's sentinel errors to gRPC codes, the same way the HTTP handlers
// of the earlier lessons mapped them to 422, 409 and 422. Anything else is Internal and
// its details stay in the server log.
func (s *Server) toStatus(err error) error {
	switch {
	case errors.Is(err, store.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, store.ErrInvalidReference):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, store.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		s.log.Error("internal error", "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}
