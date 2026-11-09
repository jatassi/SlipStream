package grabs

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
)

func TestAnAssessedReleaseReadsItsFlags(t *testing.T) {
	flags := []release.Flag{release.Halfleech, release.Internal}
	r := indexers.Release{Release: searching.Release{Title: "The.Quiet.Orchard.2021.1080p.WEB-DL", Flags: flags}}
	if got := assess(orchard, listed{Release: r}).read.Facts.Flags; !slices.Equal(got, flags) {
		t.Fatalf("a Release flagged halfleech and internal reads as %v, want %v", got, flags)
	}
}
