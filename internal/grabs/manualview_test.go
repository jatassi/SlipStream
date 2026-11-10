package grabs_test

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
)

func (s streaming) view(t *testing.T, g *grabs.Grabs, pins ...grabs.Pin) grabs.ManualView {
	t.Helper()
	v, err := g.ManualSearch(t.Context(), identity.Admin{}, s.work, pins)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAPinSurvivesReleasesStreamingIn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		g := s.open()
		if _, err := g.ManualSearch(t.Context(), identity.Admin{}, s.work, nil); !errors.Is(err, grabs.ErrNoManualSearch) {
			t.Fatalf("the view of a Work never searched by hand = %v, want ErrNoManualSearch", err)
		}
		s.start(t, g, grabs.ManualScope{})
		stop := running(t, g)
		defer stop()
		time.Sleep(10 * time.Second)
		synctest.Wait()
		v := s.view(t, g)
		if !v.Searching || v.Line != "Searching 2 Indexers. 1 has responded." {
			t.Fatalf("with Slow Site silent the sheet reads %v %q, want searching with 1 of 2 responded", v.Searching, v.Line)
		}
		row := v.Plan.Rows[0]
		if row.Pick == nil || row.Pick.Release.Title != "The.Attic.2020.1080p.BluRay.x264-GRP" || len(row.Choices) != 1 {
			t.Fatalf("Quick Site's answer plans %+v, want its BluRay with its WEB-DL under Other choices", row)
		}
		pin := row.Choices[0].Pin
		if got := s.view(t, g, pin).Plan.Rows[0].Pick.Release.Title; got != "The.Attic.2020.1080p.WEB-DL-GRP" {
			t.Fatalf("the pinned plan leads with %q, want the WEB-DL", got)
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		s.answeredWithPin(t, g, pin)
		offers, err := g.ManualReleases(t.Context(), identity.Admin{}, s.work)
		if err != nil {
			t.Fatal(err)
		}
		if len(offers) != 3 || offers[0].Rank != 1 || offers[0].Release.Title != "The.Attic.2020.1080p.BluRay.x264-GRP" || !offers[0].Grab.Grabbable {
			t.Fatalf("All Releases is %+v, want 3 led by Quick Site's BluRay", offers)
		}
	})
}

// answeredWithPin checks the view once both Indexers responded: pin
// still leads, and Other choices notes Slow Site's Release.
func (s streaming) answeredWithPin(t *testing.T, g *grabs.Grabs, pin grabs.Pin) {
	t.Helper()
	v := s.view(t, g, pin)
	if v.Searching || v.Line != "SlipStream searched 2 Indexers." {
		t.Fatalf("once Slow Site responded the sheet reads %v %q, want finished", v.Searching, v.Line)
	}
	row := v.Plan.Rows[0]
	if row.Pick == nil || row.Pick.Release.Title != "The.Attic.2020.1080p.WEB-DL-GRP" || !row.Pinned {
		t.Fatalf("after Slow Site's answer the first row is %+v, want the pinned WEB-DL", row)
	}
	if !slices.Equal(v.Pins, []grabs.Pin{pin}) {
		t.Fatalf("the pins in effect are %+v, want %+v", v.Pins, pin)
	}
	var notes []string
	for _, c := range row.Choices {
		notes = append(notes, c.Release.Title+": "+c.Note)
	}
	want := []string{
		"The.Attic.2020.1080p.BluRay.x264-GRP: ",
		"The.Attic.2020.1080p.BluRay.x264-OTHER: From Slow Site, which automatic search doesn't use.",
	}
	if !slices.Equal(notes, want) {
		t.Fatalf("Other choices are %q, want %q", notes, want)
	}
}
