package indexers_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

func sourced(id, name, privacy, origin string) searching.Release {
	return searching.Release{
		Title: "Orchard.1080p", Link: searching.LinkOf("https://site.test/a.torrent"),
		Source: &searching.Source{ID: id, Name: name, Privacy: privacy, Origin: origin},
	}
}

// reached takes r from Indexer name and returns the Tracker it reached.
func (p *planned) reached(name string, r searching.Release) indexers.Tracker {
	p.t.Helper()
	tk, err := p.take(name, r)
	if err != nil {
		p.t.Fatal(err)
	}
	return p.tracker(tk.Tracker)
}

func (p *planned) tracker(id indexers.TrackerID) indexers.Tracker {
	p.t.Helper()
	tr, err := p.x.Tracker(p.t.Context(), identity.Admin{}, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return tr
}

func (p *planned) trackerNames() []string {
	p.t.Helper()
	all, err := p.x.Trackers(p.t.Context(), identity.Admin{})
	if err != nil {
		p.t.Fatal(err)
	}
	var out []string
	for _, tr := range all {
		out = append(out, tr.Name)
	}
	return out
}

func TestASourceReachesOneTrackerByItsKeyNameOrDefinitionID(t *testing.T) {
	p := plannedWith(t, "Prowlarr", "Jackett")
	prowlarr := p.ids["Prowlarr"]

	first := p.reached("Prowlarr", sourced("3", "Torrent Leech", "private", "prowlarr"))
	if first.Name != "Torrent Leech" || first.Privacy != indexers.Private ||
		!slices.Equal(first.Keys, []string{"prowlarr:" + prowlarr.String() + ":3"}) || !slices.Equal(first.Indexers, []indexers.ID{prowlarr}) {
		t.Fatalf("a Prowlarr source's first Take made %+v, want Torrent Leech, private, keyed by the feed and source id", first)
	}
	if again := p.reached("Prowlarr", sourced("3", "Renamed upstream", "public", "prowlarr")); again.ID != first.ID {
		t.Fatalf("the same source key reached %+v, want %s", again, first.ID)
	}

	byName := p.reached("Jackett", sourced("torrentleech", "TORRENT-LEECH", "semiPrivate", "jackett"))
	if byName.ID != first.ID || !slices.Contains(byName.Keys, "definition:torrentleech") || len(byName.Indexers) != 2 || byName.Privacy != indexers.Private {
		t.Fatalf("a Jackett source of the same name reached %+v, want Tracker %s with its definition key and both Indexers, still private", byName, first.ID)
	}

	if err := p.x.RenameTracker(t.Context(), identity.Admin{}, first.ID, "TL"); err != nil {
		t.Fatal(err)
	}
	byDefinition := p.reached("Prowlarr", sourced("9", "torrent_leech!", "private", "prowlarr"))
	if byDefinition.ID != first.ID || byDefinition.Name != "TL" {
		t.Fatalf("after the rename, a new Prowlarr source named for the definition reached %+v, want TL (%s)", byDefinition, first.ID)
	}
	if other := p.reached("Prowlarr", sourced("4", "Aither", "", "prowlarr")); other.ID == first.ID || other.Privacy != indexers.Unknown {
		t.Fatalf("a source of another name reached %+v, want a new Tracker of unknown privacy", other)
	}
	if got := p.trackerNames(); !slices.Equal(got, []string{"Aither", "TL"}) {
		t.Fatalf("the Trackers are %q, want Aither and TL", got)
	}
}

func TestARenameAndAMergeSurviveTheNextTake(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch")
	plain := searching.Release{Title: "Orchard.1080p", Link: searching.LinkOf("https://site.test/a.torrent")}
	alder, birch := p.reached("Alder", plain), p.reached("Birch", plain)
	if alder.Name != "Alder" || !slices.Equal(alder.Keys, []string{"named:" + p.ids["Alder"].String()}) || alder.ID == birch.ID {
		t.Fatalf("Alder's sourceless Release reached %+v and Birch's %+v, want a Tracker named for each feed", alder, birch)
	}

	if err := p.x.RenameTracker(t.Context(), identity.Admin{}, alder.ID, "Birch"); !errors.Is(err, indexers.ErrTrackerNameTaken) {
		t.Fatalf("renaming Alder onto Birch = %v, want ErrTrackerNameTaken", err)
	}
	if err := p.x.RenameTracker(t.Context(), identity.Admin{}, alder.ID, "Alder Site"); err != nil {
		t.Fatal(err)
	}
	if again := p.reached("Alder", plain); again.ID != alder.ID || again.Name != "Alder Site" {
		t.Fatalf("after the rename Alder's next Take reached %+v, want Alder Site (%s)", again, alder.ID)
	}

	err := p.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := indexers.AddHosts(ctx, tx, birch.ID, true, []string{"tracker.birch.test", "", "tracker.birch.test"}); err != nil {
			return err
		}
		return indexers.AddHosts(ctx, tx, alder.ID, false, []string{"open.tracker.test"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if b, a := p.tracker(birch.ID), p.tracker(alder.ID); b.Privacy != indexers.Private || !slices.Equal(b.Hosts, []string{"tracker.birch.test"}) ||
		a.Privacy != indexers.Public || len(a.Hosts) != 0 {
		t.Fatalf("after one private and one public torrent, Birch is %+v and Alder %+v; want Birch private with its host, Alder public with none", b, a)
	}

	if err := p.x.MergeTracker(t.Context(), identity.Admin{}, birch.ID, birch.ID); !errors.Is(err, indexers.ErrMergeSelf) {
		t.Fatalf("merging Birch into itself = %v, want ErrMergeSelf", err)
	}
	if err := p.x.MergeTracker(t.Context(), identity.Admin{}, birch.ID, indexers.TrackerID{UUID: storage.NewUUID()}); !errors.Is(err, indexers.ErrNoTracker) {
		t.Fatalf("merging Birch into no Tracker = %v, want ErrNoTracker", err)
	}
	if err := p.x.MergeTracker(t.Context(), identity.Admin{}, birch.ID, alder.ID); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(p.repointed, [][2]indexers.TrackerID{{birch.ID, alder.ID}}) {
		t.Fatalf("the Repoints saw %v, want Birch into Alder once", p.repointed)
	}
	kept := p.tracker(alder.ID)
	if kept.Name != "Alder Site" || len(kept.Keys) != 2 || len(kept.Indexers) != 2 || !slices.Equal(kept.Hosts, []string{"tracker.birch.test"}) {
		t.Fatalf("the kept Tracker is %+v, want Alder Site with both keys, both Indexers and Birch's host", kept)
	}
	if again := p.reached("Birch", plain); again.ID != alder.ID {
		t.Fatalf("after the merge Birch's next Take reached %+v, want Alder Site (%s)", again, alder.ID)
	}
	if got := p.trackerNames(); !slices.Equal(got, []string{"Alder Site"}) {
		t.Fatalf("the Trackers are %q, want only Alder Site", got)
	}
}
