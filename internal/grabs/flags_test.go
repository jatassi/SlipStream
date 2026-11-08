package grabs

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
)

func flagged(flags ...searching.Flag) []release.Flag {
	r := indexers.Release{Release: searching.Release{Title: "The.Quiet.Orchard.2021.1080p.WEB-DL", Flags: flags}}
	return candidateOf(r, episodic{}, release.BuiltIn(), orchard.Covers, release.Fact[time.Duration]{}).Facts.Flags
}

func TestEveryIndexerFlagIsAReleaseFlag(t *testing.T) {
	indexer := []searching.Flag{
		searching.Freeleech, searching.Halfleech, searching.Freeleech75, searching.Freeleech25, searching.DoubleUpload,
		searching.Internal, searching.Scene, searching.Exclusive, searching.Nuked, searching.Golden, searching.Approved,
	}
	if got := flagged(indexer...); !slices.Equal(got, release.Flags()) {
		t.Fatalf("a Release flagged with every indexer flag reads as %v, want every release.Flag %v", got, release.Flags())
	}
}

func TestACandidateDropsAFlagNoReleaseFlagNames(t *testing.T) {
	if got := flagged("g_freeleech", searching.Halfleech); !slices.Equal(got, []release.Flag{release.Halfleech}) {
		t.Fatalf("a Release flagged g_freeleech and halfleech reads as %v, want [halfleech]", got)
	}
}
