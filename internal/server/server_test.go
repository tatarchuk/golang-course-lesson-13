package server_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "homework/gen/albums"
	"homework/internal/middleware"
	"homework/internal/server"
	"homework/internal/store"
)

// env is a real gRPC server on a loopback port with the interceptors of the binary,
// and a client connected to it. Each test gets a fresh store with the seed data and
// the server log captured, so a test can assert what the handlers reported.
type env struct {
	client pb.AlbumsServiceClient
	store  *store.Store
	log    *syncBuffer
}

// syncBuffer collects the server log from the server goroutines without a data race.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func start(t *testing.T) env {
	t.Helper()
	st := store.New()
	logs := &syncBuffer{}
	log := slog.New(slog.NewTextHandler(logs, nil))
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.UnaryRecovery(log), middleware.UnaryLogging(log)),
		grpc.ChainStreamInterceptor(middleware.StreamRecovery(log), middleware.StreamLogging(log)),
	)
	pb.RegisterAlbumsServiceServer(srv, server.New(st, st, log))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		srv.Stop()
	})
	return env{client: pb.NewAlbumsServiceClient(conn), store: st, log: logs}
}

func callCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func artistNames(artists []*pb.Artist) string {
	names := make([]string, 0, len(artists))
	for _, a := range artists {
		names = append(names, a.GetName())
	}
	return strings.Join(names, ",")
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func newAlbum(title, label string) *pb.CreateAlbumRequest {
	year := int32(1957)
	return &pb.CreateAlbumRequest{Title: title, Label: label, GenreId: 1, ArtistIds: []int64{2}, ReleaseYear: &year}
}

// --- GetAlbum (unary) ---------------------------------------------------------------

func TestGetAlbumReturnsKindOfBlueWithRelationsAndTimestamps(t *testing.T) {
	e := start(t)
	a, err := e.client.GetAlbum(callCtx(t), &pb.GetAlbumRequest{Id: 1})
	if err != nil {
		t.Fatal(err)
	}
	if a.GetId() != 1 || a.GetTitle() != "Kind of Blue" || a.GetLabel() != "Columbia" {
		t.Errorf("got id %d title %q label %q", a.GetId(), a.GetTitle(), a.GetLabel())
	}
	if a.GetGenre().GetId() != 1 || a.GetGenre().GetName() != "jazz" {
		t.Errorf("got genre %v", a.GetGenre())
	}
	if got := artistNames(a.GetArtists()); got != "Miles Davis,John Coltrane,Bill Evans" {
		t.Errorf("got artists %s", got)
	}
	if a.ReleaseYear == nil || a.GetReleaseYear() != 1959 {
		t.Errorf("got release year %v", a.ReleaseYear)
	}
	if a.Notes == nil || !strings.Contains(a.GetNotes(), "Coltrane") {
		t.Errorf("got notes %v", a.Notes)
	}
	if !a.GetCreatedAt().IsValid() || a.GetCreatedAt().AsTime().IsZero() || !a.GetUpdatedAt().IsValid() {
		t.Errorf("timestamps not set: %v %v", a.GetCreatedAt(), a.GetUpdatedAt())
	}
}

func TestGetAlbumUnknownIDIsNotFound(t *testing.T) {
	e := start(t)
	_, err := e.client.GetAlbum(callCtx(t), &pb.GetAlbumRequest{Id: 999})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestGetAlbumZeroIDIsInvalidArgument(t *testing.T) {
	e := start(t)
	_, err := e.client.GetAlbum(callCtx(t), &pb.GetAlbumRequest{Id: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetAlbumTenHasNoReleaseYear(t *testing.T) {
	e := start(t)
	a, err := e.client.GetAlbum(callCtx(t), &pb.GetAlbumRequest{Id: 10})
	if err != nil {
		t.Fatal(err)
	}
	if a.GetTitle() != "Waltz for Debby" || a.ReleaseYear != nil {
		t.Errorf("got title %q release year %v, want Waltz for Debby and unset", a.GetTitle(), a.ReleaseYear)
	}
}

// --- CreateAlbum (unary) ------------------------------------------------------------

func TestCreateAlbumReturnsIDElevenAndGetFindsIt(t *testing.T) {
	e := start(t)
	ctx := callCtx(t)
	created, err := e.client.CreateAlbum(ctx, newAlbum("Blue Train", "Blue Note"))
	if err != nil {
		t.Fatal(err)
	}
	if created.GetId() != 11 || !created.GetCreatedAt().IsValid() {
		t.Errorf("got %v", created)
	}
	got, err := e.client.GetAlbum(ctx, &pb.GetAlbumRequest{Id: 11})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetTitle() != "Blue Train" || got.GetGenre().GetName() != "jazz" || artistNames(got.GetArtists()) != "John Coltrane" {
		t.Errorf("stored album %v", got)
	}
	if got.ReleaseYear == nil || got.GetReleaseYear() != 1957 {
		t.Errorf("got release year %v", got.ReleaseYear)
	}
}

func TestCreateAlbumValidationErrorsAreInvalidArgument(t *testing.T) {
	e := start(t)
	cases := map[string]struct {
		req  *pb.CreateAlbumRequest
		want string // part of the Validate() message the status must carry
	}{
		"missing title":   {&pb.CreateAlbumRequest{Label: "Blue Note", GenreId: 1, ArtistIds: []int64{2}}, "title"},
		"missing label":   {&pb.CreateAlbumRequest{Title: "Blue Train", GenreId: 1, ArtistIds: []int64{2}}, "label"},
		"no artists":      {&pb.CreateAlbumRequest{Title: "Blue Train", Label: "Blue Note", GenreId: 1}, "artist_ids"},
		"genre id zero":   {&pb.CreateAlbumRequest{Title: "Blue Train", Label: "Blue Note", ArtistIds: []int64{2}}, "genre_id"},
		"repeated artist": {&pb.CreateAlbumRequest{Title: "Blue Train", Label: "Blue Note", GenreId: 1, ArtistIds: []int64{2, 2}}, "artist_ids"},
	}
	for name, tc := range cases {
		_, err := e.client.CreateAlbum(callCtx(t), tc.req)
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument || !strings.Contains(st.Message(), tc.want) {
			t.Errorf("%s: expected InvalidArgument naming %q, got %v", name, tc.want, err)
		}
	}
}

func TestCreateAlbumUnknownGenreIsFailedPrecondition(t *testing.T) {
	e := start(t)
	req := newAlbum("Blue Train", "Blue Note")
	req.GenreId = 99
	_, err := e.client.CreateAlbum(callCtx(t), req)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func TestCreateAlbumDuplicateIsAlreadyExists(t *testing.T) {
	e := start(t)
	_, err := e.client.CreateAlbum(callCtx(t), newAlbum("Kind of Blue", "Columbia"))
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
}

// --- WatchAlbums (server streaming) ---------------------------------------------------

func TestWatchAlbumsSendsCreatedEventsInOrderThenEOFAfterMaxEvents(t *testing.T) {
	e := start(t)
	ctx := callCtx(t)
	stream, err := e.client.WatchAlbums(ctx, &pb.WatchAlbumsRequest{MaxEvents: 2})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the handler to subscribe", func() bool { return e.store.SubscriberCount() == 1 })
	for _, req := range []*pb.CreateAlbumRequest{newAlbum("Blue Train", "Blue Note"), newAlbum("Giant Steps", "Atlantic")} {
		if _, err := e.client.CreateAlbum(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range []string{"Blue Train", "Giant Steps"} {
		ev, err := stream.Recv()
		if err != nil {
			t.Fatalf("event %d: %v", i+1, err)
		}
		if ev.GetType() != pb.AlbumEvent_CREATED || ev.GetAlbum().GetTitle() != want || !ev.GetAt().IsValid() {
			t.Errorf("event %d: got %v, want CREATED %q", i+1, ev, want)
		}
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("after max_events expected io.EOF, got %v", err)
	}
}

func TestWatchAlbumsClientCancelIsCanceledAndReleasesTheSubscription(t *testing.T) {
	e := start(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := e.client.WatchAlbums(ctx, &pb.WatchAlbumsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the handler to subscribe", func() bool { return e.store.SubscriberCount() == 1 })
	cancel()
	if _, err := stream.Recv(); status.Code(err) != codes.Canceled {
		t.Fatalf("expected Canceled on the client, got %v", err)
	}
	// The proof of slide 46 is on the server side: the handler itself saw the cancellation,
	// returned Canceled through the interceptor chain, and its deferred cancel released the
	// subscription. Only the handler can release it, so the count is checked after its exit.
	waitFor(t, "the handler to finish with the cancellation reason", func() bool {
		out := e.log.String()
		return strings.Contains(out, `reason="client gone / cancelled"`) && strings.Contains(out, "code=Canceled")
	})
	if n := e.store.SubscriberCount(); n != 0 {
		t.Fatalf("subscriber count after the handler returned: %d, want 0 (the handler must release its subscription)", n)
	}
}

// --- slide 46: errors are gRPC statuses, never plain error values ----------------------

func TestServerHandlersUseStatusErrorsOnly(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"errors.New(", "fmt.Errorf("} {
		if strings.Contains(string(src), forbidden) {
			t.Errorf("server.go must not use %s; return status errors with the right code", forbidden)
		}
	}
}
