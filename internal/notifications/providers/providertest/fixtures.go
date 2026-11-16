// Package providertest checks what a provider builds against golden files,
// and holds the fixture messages and posters it builds them from.
package providertest

import (
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// At is when every fixture message is sent.
var At = time.Date(2026, time.November, 16, 14, 2, 0, 0, time.UTC)

// Fixture is one kind's message as its row renders it from fixture facts.
type Fixture struct {
	Kind    notice.Kind
	Message notice.Message
	// Poster: the message carries a Work's poster, so the provider builds it
	// with and without one.
	Poster bool
}

var heat = &notice.BodyWork{ID: "w1", Title: "Heat (1995)", ContentType: "fixture"}

// Fixtures are one message per kind, in the words the spec's table gives.
var Fixtures = []Fixture{
	{Kind: "grabbed", Poster: true, Message: notice.Message{
		Title: "Grabbed Heat (1995)",
		Lines: []string{
			"WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB",
			"Automatic search found it on TorrentLeech.",
			"It scores 1250 for the 1080p Target and 900 for the 720p Target.",
		},
		Link: notice.Link{Path: "https://slipstream.example/works/w1"},
		Tone: notice.Info,
		Body: notice.Body{Work: heat},
	}},
	{Kind: "grab_failed", Poster: true, Message: notice.Message{
		Title: "The Grab for Heat (1995) failed",
		Lines: []string{
			"qBittorrent refused the torrent because it already has it.",
			"SlipStream searches again automatically.",
		},
		Link: notice.Link{Path: "https://slipstream.example/works/w1"},
		Tone: notice.Fail,
		Body: notice.Body{Work: heat},
	}},
	{Kind: "imported", Poster: true, Message: notice.Message{
		Title: "Imported 10 episodes of Severance",
		Lines: []string{
			"Season 2, episodes 1 to 10.",
			"WEBDL-2160p · Dolby Vision · DDP 5.1 · NTb · 48.1 GB",
			"1 episode waits for you on the Work page.",
		},
		Link: notice.Link{Path: "https://slipstream.example/works/w2"},
		Tone: notice.Good,
		Body: notice.Body{
			Work:  &notice.BodyWork{ID: "w2", Title: "Severance", ContentType: "fixture"},
			Files: []notice.BodyFile{{Items: []string{"S02E01", "S02E02"}, Quality: "WEBDL-2160p", PlacedBy: "hardlink"}},
		},
	}},
	{Kind: "upgraded", Poster: true, Message: notice.Message{
		Title: "Upgraded Heat (1995) to Remux-2160p",
		Lines: []string{"It replaced WEBDL-1080p."},
		Link:  notice.Link{Path: "https://slipstream.example/works/w1"},
		Tone:  notice.Good,
		Body: notice.Body{
			Work:  heat,
			Files: []notice.BodyFile{{Items: []string{"Heat (1995)"}, Quality: "Remux-2160p", Replaced: "WEBDL-1080p", PlacedBy: "hardlink"}},
		},
	}},
	{Kind: "works_added", Poster: true, Message: notice.Message{
		Title: "Added Heat (1995) to Movies",
		Lines: []string{"Sam added it."},
		Link:  notice.Link{Path: "https://slipstream.example/works/w1"},
		Body:  notice.Body{Work: heat},
	}},
	{Kind: "works_added_by_scan", Message: notice.Message{
		Title: "A Library scan of Movies on Den added 12 Works",
		Lines: []string{"Alien (1979)", "Arrival (2016)", "Blade Runner (1982)", "9 more were added."},
	}},
	{Kind: "works_deleted", Message: notice.Message{
		Title: "Deleted 12 Works from Movies",
		Lines: []string{"Alien (1979)", "Arrival (2016)", "Blade Runner (1982)", "9 more were deleted."},
	}},
	{Kind: "new_health_item", Message: notice.Message{
		Title: "Den's disk is almost full",
		Lines: []string{"Movies has 4 GB free of 2 TB.", "It has been on Health since 13:57."},
		Link:  notice.Link{Path: "https://slipstream.example/health#health-storage-den"},
		Tone:  notice.Warn,
		Body:  notice.Body{Health: []notice.BodyItem{{Category: "storage", Key: "den", Level: "warning", Subject: "Den's disk is almost full", Text: "Movies has 4 GB free of 2 TB."}}},
	}},
	{Kind: "health_item_cleared", Message: notice.Message{
		Title: "Cleared: Den's disk is almost full",
		Lines: []string{"It was on Health from 13:57 to 14:02."},
		Tone:  notice.Good,
		Body:  notice.Body{Health: []notice.BodyItem{{Category: "storage", Key: "den", Level: "warning", Subject: "Den's disk is almost full", Text: "Movies has 4 GB free of 2 TB."}}},
	}},
	{Kind: "test", Message: notice.Message{
		Title: "Test from SlipStream",
		Lines: []string{"This connection works.", "Den sent this message at 14:02."},
	}},
}

// FixtureOf is the fixture of kind k.
func FixtureOf(k notice.Kind) Fixture {
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
