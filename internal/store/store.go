// Package store keeps the album catalogue in memory (slide 41: a map first, a real
// database later) together with the subscriptions that feed the server-streaming call.
package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType says what happened to the catalogue.
type EventType int

const (
	EventUnspecified EventType = iota
	EventCreated
)

// Event is one catalogue change delivered to subscribers.
type Event struct {
	Type  EventType
	Album Album
	At    time.Time
}

// subscriberBuffer is how many events a subscriber may lag behind before events are
// dropped for it; a slow subscriber never blocks Create.
const subscriberBuffer = 16

type subscription struct {
	ch   chan Event
	done chan struct{}
	once sync.Once
}

// Store is safe for concurrent use.
type Store struct {
	mu      sync.RWMutex
	genres  map[int64]Genre
	artists map[int64]Artist
	albums  map[int64]*Album
	nextID  int64
	subs    map[*subscription]struct{}
}

// New returns a store loaded with the seed data; the next created album gets id 11.
func New() *Store {
	s := &Store{
		genres:  make(map[int64]Genre),
		artists: make(map[int64]Artist),
		albums:  make(map[int64]*Album),
		subs:    make(map[*subscription]struct{}),
	}
	for _, g := range seedGenres() {
		s.genres[g.ID] = g
	}
	for _, a := range seedArtists() {
		s.artists[a.ID] = a
	}
	loaded := time.Now()
	for _, sa := range seedAlbums() {
		artists := make([]Artist, 0, len(sa.artistIDs))
		for _, id := range sa.artistIDs {
			artists = append(artists, s.artists[id])
		}
		s.albums[sa.id] = &Album{
			ID: sa.id, Title: sa.title, Label: sa.label, Genre: s.genres[sa.genreID], Artists: artists,
			ReleaseYear: sa.year, Notes: sa.notes, CreatedAt: loaded, UpdatedAt: loaded,
		}
		if sa.id >= s.nextID {
			s.nextID = sa.id + 1
		}
	}
	return s
}

// Get returns a copy of the album, so callers never share mutable state.
func (s *Store) Get(id int64) (Album, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.albums[id]
	if !ok {
		return Album{}, ErrNotFound
	}
	return clone(*a), nil
}

// Create validates the input, resolves the references, rejects a duplicate (title, label),
// stores the album with the next id and both timestamps, and publishes a CREATED event.
func (s *Store) Create(in AlbumInput) (Album, error) {
	if err := in.Validate(); err != nil {
		return Album{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	genre, ok := s.genres[in.GenreID]
	if !ok {
		return Album{}, fmt.Errorf("%w: genre %d", ErrInvalidReference, in.GenreID)
	}
	artists := make([]Artist, 0, len(in.ArtistIDs))
	for _, id := range in.ArtistIDs {
		a, ok := s.artists[id]
		if !ok {
			return Album{}, fmt.Errorf("%w: artist %d", ErrInvalidReference, id)
		}
		artists = append(artists, a)
	}
	for _, a := range s.albums {
		if a.Title == in.Title && a.Label == in.Label {
			return Album{}, fmt.Errorf("%w: %q / %q", ErrConflict, in.Title, in.Label)
		}
	}
	now := time.Now()
	a := &Album{
		ID: s.nextID, Title: in.Title, Label: in.Label, Genre: genre, Artists: artists,
		ReleaseYear: copyInt(in.ReleaseYear), Notes: copyString(in.Notes), CreatedAt: now, UpdatedAt: now,
	}
	s.nextID++
	s.albums[a.ID] = a
	s.publish(Event{Type: EventCreated, Album: clone(*a), At: now})
	return clone(*a), nil
}

// Subscribe registers a subscriber for catalogue events. The returned cancel removes the
// subscription and closes the channel; it is safe to call more than once. The subscription
// is also removed when ctx is done, so a handler that passes its stream context cannot leak.
func (s *Store) Subscribe(ctx context.Context) (<-chan Event, func()) {
	sub := &subscription{ch: make(chan Event, subscriberBuffer), done: make(chan struct{})}
	s.mu.Lock()
	s.subs[sub] = struct{}{}
	s.mu.Unlock()
	cancel := func() {
		sub.once.Do(func() {
			s.mu.Lock()
			delete(s.subs, sub)
			close(sub.ch) // under the lock: publish sends under the same lock, so never on a closed channel
			s.mu.Unlock()
			close(sub.done)
		})
	}
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-sub.done:
		}
	}()
	return sub.ch, cancel
}

// SubscriberCount is how many subscriptions are active (the leak check of slide 46).
func (s *Store) SubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subs)
}

// publish delivers the event to every subscriber without blocking: a subscriber whose
// buffer is full misses the event. The caller holds s.mu.
func (s *Store) publish(ev Event) {
	for sub := range s.subs {
		select {
		case sub.ch <- ev:
		default:
		}
	}
}

func clone(a Album) Album {
	a.Artists = append([]Artist(nil), a.Artists...)
	a.ReleaseYear = copyInt(a.ReleaseYear)
	a.Notes = copyString(a.Notes)
	return a
}

func copyInt(v *int) *int {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

func copyString(v *string) *string {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
