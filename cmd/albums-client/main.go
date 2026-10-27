// albums-client is the Go client of AlbumsService: get and create (unary), watch (server
// streaming, slide 42).
//
//	go run ./cmd/albums-client get 1
//	go run ./cmd/albums-client create "Blue Train" "Blue Note" 1 2
//	go run ./cmd/albums-client -year 1957 -notes "Coltrane's Blue Note debut" create "Blue Train" "Blue Note" 1 2,6
//	go run ./cmd/albums-client watch                 # until Ctrl+C
//	go run ./cmd/albums-client -max 2 watch          # the server closes the stream after 2 events
//
// Global flags (-addr, -timeout, -max, -year, -notes) go BEFORE the command.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "homework/gen/albums"
)

var (
	addr      = flag.String("addr", "localhost:50051", "address of the gRPC server")
	timeout   = flag.Duration("timeout", 5*time.Second, "timeout of a unary call")
	maxEvents = flag.Int("max", 0, "watch: the server closes the stream after N events (0 = no limit)")
	year      = flag.Int("year", -1, "create: release year (omit for none)")
	notes     = flag.String("notes", "", "create: notes (omit for none)")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	os.Exit(run(flag.Args()))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	// Plaintext is for local development only (slide 33).
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		return 1
	}
	defer conn.Close()
	client := pb.NewAlbumsServiceClient(conn)

	// Ctrl+C cancels the context: that is how the client cancels the stream.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch cmd, rest := args[0], args[1:]; cmd {
	case "get":
		if len(rest) != 1 {
			return usageError("get <id>")
		}
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil {
			return usageError("get <id>: id must be an integer")
		}
		err = cmdGet(ctx, client, id)
		return exitCode(err)
	case "create":
		if len(rest) != 4 {
			return usageError("create <title> <label> <genre_id> <artist_ids>")
		}
		req, err := createRequest(rest)
		if err != nil {
			return usageError(err.Error())
		}
		return exitCode(cmdCreate(ctx, client, req))
	case "watch":
		return exitCode(cmdWatch(ctx, client))
	default:
		usage()
		return 2
	}
}

// --- unary ------------------------------------------------------------------------------

func cmdGet(ctx context.Context, c pb.AlbumsServiceClient, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	album, err := c.GetAlbum(ctx, &pb.GetAlbumRequest{Id: id})
	if err != nil {
		return err
	}
	printMsg(album)
	return nil
}

func cmdCreate(ctx context.Context, c pb.AlbumsServiceClient, req *pb.CreateAlbumRequest) error {
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	album, err := c.CreateAlbum(ctx, req)
	if err != nil {
		return err
	}
	printMsg(album)
	return nil
}

func createRequest(args []string) (*pb.CreateAlbumRequest, error) {
	genreID, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("genre_id must be an integer, got %q", args[2])
	}
	var artistIDs []int64
	for _, part := range strings.Split(args[3], ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("artist_ids must be comma-separated integers, got %q", args[3])
		}
		artistIDs = append(artistIDs, id)
	}
	req := &pb.CreateAlbumRequest{Title: args[0], Label: args[1], GenreId: genreID, ArtistIds: artistIDs}
	if *year >= 0 {
		y := int32(*year) // the server validates the range
		req.ReleaseYear = &y
	}
	if *notes != "" {
		n := *notes
		req.Notes = &n
	}
	return req, nil
}

// --- server streaming (slide 42) ---------------------------------------------------------

func cmdWatch(ctx context.Context, c pb.AlbumsServiceClient) error {
	// No timeout: the stream lives until the server closes it or Ctrl+C cancels it.
	stream, err := c.WatchAlbums(ctx, &pb.WatchAlbumsRequest{MaxEvents: int32(*maxEvents)})
	if err != nil {
		return err
	}
	if *maxEvents > 0 {
		fmt.Printf("watching albums, the server will close the stream after %d events (Ctrl+C to stop)...\n", *maxEvents)
	} else {
		fmt.Println("watching albums (Ctrl+C to stop)...")
	}
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("stream closed by the server (io.EOF)")
			return nil
		}
		if status.Code(err) == codes.Canceled {
			fmt.Println("stream cancelled by the client (Ctrl+C)")
			return nil
		}
		if err != nil {
			return err
		}
		a := ev.GetAlbum()
		fmt.Printf("%s  %s  album %d %q (%s)\n",
			ev.GetAt().AsTime().Local().Format("15:04:05.000"), ev.GetType(), a.GetId(), a.GetTitle(), a.GetLabel())
	}
}

// --- helpers -----------------------------------------------------------------------------

func printMsg(m proto.Message) {
	fmt.Println(protojson.MarshalOptions{Multiline: true, UseProtoNames: true, EmitUnpopulated: true}.Format(m))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if st, ok := status.FromError(err); ok {
		fmt.Fprintf(os.Stderr, "RPC error: code=%s message=%q\n", st.Code(), st.Message())
	} else {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}

func usageError(msg string) int {
	fmt.Fprintln(os.Stderr, "usage: albums-client [flags]", msg)
	return 2
}

func usage() {
	fmt.Fprint(os.Stderr, `albums-client [flags] <command> [arguments]

Commands:
  get <id>                                     unary: one album by id
  create <title> <label> <genre_id> <artist_ids>  unary: create an album (artist_ids: 1 or 1,2)
  watch                                        server streaming: live catalogue events

Flags:
`)
	flag.PrintDefaults()
}
