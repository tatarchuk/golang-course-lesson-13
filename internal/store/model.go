package store

import (
	"fmt"
	"strings"
	"time"
)

// Bounds for release_year: a year outside this range is a typo, not a release.
const (
	minReleaseYear = 0
	maxReleaseYear = 9999
)

// Genre is a music genre; every album belongs to exactly one.
type Genre struct {
	ID   int64
	Name string
}

// Artist is a performer; an album has one or more artists.
type Artist struct {
	ID   int64
	Name string
}

// Album is the stored resource. Optional fields are pointers so that "not provided" stays
// distinguishable from zero and empty, as null did in the REST API of the earlier lessons.
type Album struct {
	ID          int64
	Title       string
	Label       string
	Genre       Genre
	Artists     []Artist
	ReleaseYear *int
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AlbumInput is the album to create. Relations are given by id, so a client can only
// reference genres and artists that already exist; the store checks that they do.
type AlbumInput struct {
	Title       string
	Label       string
	GenreID     int64
	ArtistIDs   []int64
	ReleaseYear *int
	Notes       *string
}

// Validate checks the required fields and the shape of the references with the rules and
// messages of POST /albums from the earlier lessons. Whether the referenced genre and
// artists exist is the store's job (Create).
func (in AlbumInput) Validate() error {
	var problems []string
	var missing []string
	if strings.TrimSpace(in.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(in.Label) == "" {
		missing = append(missing, "label")
	}
	if len(missing) > 0 {
		problems = append(problems, "required fields missing or empty: "+strings.Join(missing, ", "))
	}
	if in.GenreID < 1 {
		problems = append(problems, "genre_id must be a positive integer")
	}
	if in.ReleaseYear != nil && (*in.ReleaseYear < minReleaseYear || *in.ReleaseYear > maxReleaseYear) {
		problems = append(problems, fmt.Sprintf("release_year must be between %d and %d", minReleaseYear, maxReleaseYear))
	}
	switch {
	case len(in.ArtistIDs) == 0:
		problems = append(problems, "artist_ids must contain at least one artist id")
	default:
		seen := make(map[int64]bool, len(in.ArtistIDs))
		for _, id := range in.ArtistIDs {
			if id < 1 {
				problems = append(problems, "artist_ids must contain positive integers")
				break
			}
			if seen[id] {
				problems = append(problems, "artist_ids must not repeat an id")
				break
			}
			seen[id] = true
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
}
