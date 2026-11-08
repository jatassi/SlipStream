package grabs

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
)

func TestACandidateReadsItsReleasesFlags(t *testing.T) {
	flags := []release.Flag{release.Halfleech, release.Internal}
	r := indexers.Release{Release: searching.Release{Title: "The.Quiet.Orchard.2021.1080p.WEB-DL", Flags: flags}}
	if got := candidateOf(r, episodic{}, release.BuiltIn(), orchard.Covers, release.Fact[time.Duration]{}).Facts.Flags; !slices.Equal(got, flags) {
		t.Fatalf("a Release flagged halfleech and internal reads as %v, want %v", got, flags)
	}
}
