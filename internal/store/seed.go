package store

// The demo data of the earlier lessons (variant "albums") as Go literals: three genres,
// six artists, ten albums. Album 1 has three artists, artist 1 appears on albums 1 and 2,
// every album has at least one artist, album 10 has no release year.

type seedAlbum struct {
	id        int64
	title     string
	label     string
	genreID   int64
	year      *int
	notes     *string
	artistIDs []int64
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func seedGenres() []Genre {
	return []Genre{{1, "jazz"}, {2, "rock"}, {3, "electronic"}}
}

func seedArtists() []Artist {
	return []Artist{
		{1, "Miles Davis"}, {2, "John Coltrane"}, {3, "Nirvana"},
		{4, "Radiohead"}, {5, "Daft Punk"}, {6, "Bill Evans"},
	}
}

func seedAlbums() []seedAlbum {
	return []seedAlbum{
		{1, "Kind of Blue", "Columbia", 1, intPtr(1959), strPtr("Recorded in 1959 with Coltrane and Evans."), []int64{1, 2, 6}},
		{2, "Sketches of Spain", "Columbia", 1, intPtr(1960), nil, []int64{1}},
		{3, "A Love Supreme", "Impulse!", 1, intPtr(1965), strPtr("Four-part suite."), []int64{2}},
		{4, "Nevermind", "DGC", 2, intPtr(1991), nil, []int64{3}},
		{5, "In Utero", "DGC", 2, intPtr(1993), nil, []int64{3}},
		{6, "OK Computer", "Parlophone", 2, intPtr(1997), strPtr("Recorded at St Catherine's Court."), []int64{4}},
		{7, "Kid A", "Parlophone", 2, intPtr(2000), nil, []int64{4}},
		{8, "Discovery", "Virgin", 3, intPtr(2001), nil, []int64{5}},
		{9, "Random Access Memories", "Columbia", 3, intPtr(2013), strPtr("Grammy Album of the Year 2014."), []int64{5}},
		{10, "Waltz for Debby", "Riverside", 1, nil, strPtr("Live at the Village Vanguard."), []int64{6}},
	}
}
