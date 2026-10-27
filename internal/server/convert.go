package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "homework/gen/albums"
	"homework/internal/store"
)

// toProto maps a stored album to the wire message; unset optional fields stay unset.
func toProto(a store.Album) *pb.Album {
	out := &pb.Album{
		Id:        a.ID,
		Title:     a.Title,
		Label:     a.Label,
		Genre:     &pb.Genre{Id: a.Genre.ID, Name: a.Genre.Name},
		Artists:   make([]*pb.Artist, 0, len(a.Artists)),
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
	for _, ar := range a.Artists {
		out.Artists = append(out.Artists, &pb.Artist{Id: ar.ID, Name: ar.Name})
	}
	if a.ReleaseYear != nil {
		year := int32(*a.ReleaseYear) // validated to 0..9999, so the conversion cannot overflow
		out.ReleaseYear = &year
	}
	if a.Notes != nil {
		notes := *a.Notes
		out.Notes = &notes
	}
	return out
}

// toInput maps the request to the store's input; presence of the optional fields is kept.
func toInput(req *pb.CreateAlbumRequest) store.AlbumInput {
	in := store.AlbumInput{
		Title:     req.GetTitle(),
		Label:     req.GetLabel(),
		GenreID:   req.GetGenreId(),
		ArtistIDs: append([]int64(nil), req.GetArtistIds()...),
	}
	if req.ReleaseYear != nil {
		year := int(req.GetReleaseYear())
		in.ReleaseYear = &year
	}
	if req.Notes != nil {
		notes := req.GetNotes()
		in.Notes = &notes
	}
	return in
}

func toEvent(ev store.Event) *pb.AlbumEvent {
	t := pb.AlbumEvent_TYPE_UNSPECIFIED
	if ev.Type == store.EventCreated {
		t = pb.AlbumEvent_CREATED
	}
	return &pb.AlbumEvent{Type: t, Album: toProto(ev.Album), At: timestamppb.New(ev.At)}
}
