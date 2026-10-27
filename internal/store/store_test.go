package store

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// validInput is a new album that references seeded relations without duplicating a
// seeded (title, label) pair.
func validInput() AlbumInput {
	year := 1957
	notes := "Coltrane's Blue Note debut."
	return AlbumInput{
		Title:       "Blue Train",
		Label:       "Blue Note",
		GenreID:     1,
		ArtistIDs:   []int64{2},
		ReleaseYear: &year,
		Notes:       &notes,
	}
}

func artistNames(artists []Artist) []string {
	names := make([]string, 0, len(artists))
	for _, a := range artists {
		names = append(names, a.Name)
	}
	return names
}

// --- seed ---------------------------------------------------------------------

func TestSeedHasTenAlbumsEachWithArtistsAndTimestamps(t *testing.T) {
	s := New()
	for id := int64(1); id <= 10; id++ {
		a, err := s.Get(id)
		if err != nil {
			t.Fatalf("album %d: %v", id, err)
		}
		if len(a.Artists) == 0 {
			t.Errorf("album %d has no artists", id)
		}
		if a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() {
			t.Errorf("album %d: timestamps not set", id)
		}
	}
	if _, err := s.Get(11); !errors.Is(err, ErrNotFound) {
		t.Errorf("album 11: expected ErrNotFound, got %v", err)
	}
}

func TestGetReturnsKindOfBlueWithGenreAndArtistsInSeedOrder(t *testing.T) {
	a, err := New().Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Kind of Blue" || a.Label != "Columbia" {
		t.Errorf("got title %q label %q", a.Title, a.Label)
	}
	if a.Genre != (Genre{ID: 1, Name: "jazz"}) {
		t.Errorf("got genre %+v", a.Genre)
	}
	if got, want := strings.Join(artistNames(a.Artists), ","), "Miles Davis,John Coltrane,Bill Evans"; got != want {
		t.Errorf("got artists %s, want %s", got, want)
	}
	if a.ReleaseYear == nil || *a.ReleaseYear != 1959 {
		t.Errorf("got release year %v", a.ReleaseYear)
	}
	if a.Notes == nil || *a.Notes != "Recorded in 1959 with Coltrane and Evans." {
		t.Errorf("got notes %v", a.Notes)
	}
}

func TestGetAlbumTenHasNoReleaseYear(t *testing.T) {
	a, err := New().Get(10)
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Waltz for Debby" || a.ReleaseYear != nil {
		t.Errorf("got title %q release year %v, want Waltz for Debby and nil", a.Title, a.ReleaseYear)
	}
}

func TestGetUnknownIDIsNotFound(t *testing.T) {
	s := New()
	for _, id := range []int64{0, -1, 999} {
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("id %d: expected ErrNotFound, got %v", id, err)
		}
	}
}

// --- validation (the Lesson 12 rules) --------------------------------------------

func TestValidateAcceptsCompleteInput(t *testing.T) {
	if err := validInput().Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateAcceptsOmittedOptionalFields(t *testing.T) {
	in := validInput()
	in.ReleaseYear, in.Notes = nil, nil
	if err := in.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateRejectsBlankRequiredFields(t *testing.T) {
	in := validInput()
	in.Title, in.Label = "", "   "
	err := in.Validate()
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	for _, field := range []string{"title", "label"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error %q should name %q", err, field)
		}
	}
}

func TestValidateRejectsNonPositiveGenreID(t *testing.T) {
	for _, id := range []int64{0, -1} {
		in := validInput()
		in.GenreID = id
		err := in.Validate()
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "genre_id") {
			t.Errorf("genre_id=%d: expected ErrValidation naming genre_id, got %v", id, err)
		}
	}
}

func TestValidateRejectsEmptyOrInvalidArtistIDs(t *testing.T) {
	cases := map[string][]int64{
		"nil":       nil,
		"empty":     {},
		"zero":      {0},
		"negative":  {1, -2},
		"duplicate": {1, 1},
	}
	for name, ids := range cases {
		in := validInput()
		in.ArtistIDs = ids
		err := in.Validate()
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "artist_ids") {
			t.Errorf("%s: expected ErrValidation naming artist_ids, got %v", name, err)
		}
	}
}

func TestValidateRejectsReleaseYearOutOfRange(t *testing.T) {
	for _, year := range []int{-5, 10000} {
		in := validInput()
		in.ReleaseYear = &year
		err := in.Validate()
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "release_year") {
			t.Errorf("release_year=%d: expected ErrValidation naming release_year, got %v", year, err)
		}
	}
}

// --- create ---------------------------------------------------------------------

func TestCreateAssignsNextIDResolvesRelationsAndSetsTimestamps(t *testing.T) {
	s := New()
	before := time.Now()
	a, err := s.Create(validInput())
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != 11 {
		t.Errorf("got id %d, want 11", a.ID)
	}
	if a.CreatedAt.Before(before) || !a.CreatedAt.Equal(a.UpdatedAt) {
		t.Errorf("timestamps: created %v updated %v", a.CreatedAt, a.UpdatedAt)
	}
	got, err := s.Get(11)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Blue Train" || got.Genre.Name != "jazz" || strings.Join(artistNames(got.Artists), ",") != "John Coltrane" {
		t.Errorf("stored album %+v", got)
	}
	if got.ReleaseYear == nil || *got.ReleaseYear != 1957 || got.Notes == nil {
		t.Errorf("optional fields not stored: %v %v", got.ReleaseYear, got.Notes)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	in := validInput()
	in.Title = ""
	_, err := New().Create(in)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateRejectsUnknownGenre(t *testing.T) {
	in := validInput()
	in.GenreID = 99
	_, err := New().Create(in)
	if !errors.Is(err, ErrInvalidReference) || !strings.Contains(err.Error(), "genre") {
		t.Fatalf("expected ErrInvalidReference naming the genre, got %v", err)
	}
}

func TestCreateRejectsUnknownArtist(t *testing.T) {
	in := validInput()
	in.ArtistIDs = []int64{1, 99}
	_, err := New().Create(in)
	if !errors.Is(err, ErrInvalidReference) || !strings.Contains(err.Error(), "artist") {
		t.Fatalf("expected ErrInvalidReference naming the artist, got %v", err)
	}
}

func TestCreateRejectsDuplicateTitleAndLabel(t *testing.T) {
	in := validInput()
	in.Title, in.Label = "Kind of Blue", "Columbia" // seeded album 1
	_, err := New().Create(in)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreateAllowsSameTitleOnAnotherLabel(t *testing.T) {
	in := validInput()
	in.Title, in.Label = "Kind of Blue", "Legacy"
	a, err := New().Create(in)
	if err != nil || a.ID != 11 {
		t.Fatalf("expected album 11, got %+v, %v", a, err)
	}
}

// --- subscriptions ----------------------------------------------------------------

func TestSubscribeReceivesCreatedEvents(t *testing.T) {
	s := New()
	events, cancel := s.Subscribe()
	defer cancel()
	a, err := s.Create(validInput())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		if ev.Type != EventCreated || ev.Album.ID != a.ID || ev.Album.Title != "Blue Train" || ev.At.IsZero() {
			t.Fatalf("unexpected event %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("no event received")
	}
}

func TestCancelIsIdempotentAndClosesTheChannel(t *testing.T) {
	s := New()
	events, cancel := s.Subscribe()
	if n := s.SubscriberCount(); n != 1 {
		t.Fatalf("subscriber count %d, want 1", n)
	}
	cancel()
	cancel()
	if n := s.SubscriberCount(); n != 0 {
		t.Errorf("subscriber count after cancel %d, want 0", n)
	}
	if _, ok := <-events; ok {
		t.Error("the events channel should be closed after cancel")
	}
}

func TestSlowSubscriberDoesNotBlockCreate(t *testing.T) {
	s := New()
	_, cancel := s.Subscribe() // never reads
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			in := validInput()
			in.Title = fmt.Sprintf("Album %d", i)
			if _, err := s.Create(in); err != nil {
				t.Errorf("create %d: %v", i, err)
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Create blocked on a subscriber that never reads")
	}
}
