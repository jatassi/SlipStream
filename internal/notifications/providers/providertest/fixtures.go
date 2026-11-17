// Package providertest checks what a provider builds against golden files,
// and holds the fixture messages and posters it builds them from.
package providertest

import (
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// At is when every fixture message is sent.
var At = time.Date(2026, time.November, 16, 14, 2, 0, 0, time.UTC)

// Fixture is one kind's message as its row renders it from fixture facts.
type Fixture struct {
	Kind    entry.Kind
	Message entry.Message
	// Poster: the message carries a Work's poster, so the provider builds it
	// with and without one.
	Poster bool
}

var heat = &entry.BodyWork{ID: "w1", Title: "Heat (1995)", ContentType: "fixture"}

// Fixtures are one message per kind, in the words the spec's table gives.
var Fixtures = []Fixture{
	{Kind: "grabbed", Poster: true, Message: entry.Message{
		Title: "Grabbed Heat (1995)",
		Lines: []string{
			"WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB",
			"Automatic search found it on TorrentLeech.",
			"It scores 1250 for the 1080p Target and 900 for the 720p Target.",
		},
		Link: entry.Link{Path: "https://slipstream.example/works/w1"},
		Tone: entry.Info,
		Body: entry.Body{Work: heat},
	}},
	{Kind: "grab_failed", Poster: true, Message: entry.Message{
		Title: "The Grab for Heat (1995) failed",
		Lines: []string{
			"qBittorrent refused the torrent because it already has it.",
			"SlipStream searches again automatically.",
		},
		Link: entry.Link{Path: "https://slipstream.example/works/w1"},
		Tone: entry.Fail,
		Body: entry.Body{Work: heat},
	}},
	{Kind: "imported", Poster: true, Message: entry.Message{
		Title: "Imported 10 episodes of Severance",
		Lines: []string{
			"Season 2, episodes 1 to 10.",
			"WEBDL-2160p · Dolby Vision · DDP 5.1 · NTb · 48.1 GB",
			"1 episode waits for you on the series page.",
		},
		Link: entry.Link{Path: "https://slipstream.example/works/w2"},
		Tone: entry.Good,
		Body: entry.Body{
			Work:  &entry.BodyWork{ID: "w2", Title: "Severance", ContentType: "fixture"},
			Files: []entry.BodyFile{{Items: []string{"S02E01", "S02E02"}, Quality: "WEBDL-2160p", PlacedBy: "hardlink"}},
		},
	}},
	{Kind: "upgraded", Poster: true, Message: entry.Message{
		Title: "Upgraded Heat (1995) to Remux-2160p",
		Lines: []string{"It replaced WEBDL-1080p."},
		Link:  entry.Link{Path: "https://slipstream.example/works/w1"},
		Tone:  entry.Good,
		Body: entry.Body{
			Work:  heat,
			Files: []entry.BodyFile{{Items: []string{"Heat (1995)"}, Quality: "Remux-2160p", Replaced: "WEBDL-1080p", PlacedBy: "hardlink"}},
		},
	}},
	{Kind: "works_added", Poster: true, Message: entry.Message{
		Title: "Added Heat (1995) to Movies",
		Lines: []string{"Sam added it."},
		Link:  entry.Link{Path: "https://slipstream.example/works/w1"},
		Body:  entry.Body{Work: heat},
	}},
	{Kind: "works_added_by_scan", Message: entry.Message{
		Title: "A Library scan of the Movies library root on Den added 12 movies",
		Lines: []string{"Alien (1979)", "Arrival (2016)", "Blade Runner (1982)", "9 more were added."},
	}},
	{Kind: "works_deleted", Message: entry.Message{
		Title: "Deleted 12 movies",
		Lines: []string{"Alien (1979)", "Arrival (2016)", "Blade Runner (1982)", "9 more were deleted."},
	}},
	{Kind: "new_health_item", Message: entry.Message{
		Title: "Den's disk is almost full",
		Lines: []string{"Movies has 4 GB free of 2 TB.", "It has been on Health since 13:57."},
		Link:  entry.Link{Path: "https://slipstream.example/health#health-storage-den"},
		Tone:  entry.Warn,
		Body:  entry.Body{Health: []entry.BodyItem{{Category: "storage", Key: "den", Level: "warning", Subject: "Den's disk is almost full", Text: "Movies has 4 GB free of 2 TB."}}},
	}},
	{Kind: "health_item_cleared", Message: entry.Message{
		Title: "Cleared: Den's disk is almost full",
		Lines: []string{"It was on Health from 13:57 to 14:02."},
		Tone:  entry.Good,
		Body:  entry.Body{Health: []entry.BodyItem{{Category: "storage", Key: "den", Level: "warning", Subject: "Den's disk is almost full", Text: "Movies has 4 GB free of 2 TB."}}},
	}},
	{Kind: "test", Message: entry.Message{
		Title: "Test from SlipStream",
		Lines: []string{"This connection works.", "Den sent this message at 14:02."},
	}},
}

// FixtureOf is the fixture of kind k.
func FixtureOf(k entry.Kind) Fixture {
	for _, f := range Fixtures {
		if f.Kind == k {
			return f
		}
	}
	panic("providertest: no fixture of kind " + string(k))
}

// PosterBytes stand in for a poster's JPEG.
var PosterBytes = []byte("\xff\xd8\xff\xe0 fixture poster \xff\xd9")

// PosterFor is the fixture poster in the way p takes one, nil for none.
func PosterFor(p providers.Provider) *providers.Poster {
	switch p.Image {
	case providers.Bytes:
		return &providers.Poster{Bytes: PosterBytes, Type: "image/jpeg", Name: "poster.jpg"}
	case providers.PublicURL, providers.BodyURL:
		return &providers.Poster{URL: "https://images.example/t/p/w500/heat.jpg"}
	case providers.NoImage:
	}
	return nil
}
