package trackers_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

const gib = 1 << 30

// hashOf is a stand-in info hash for name.
func hashOf(name string) bencode.InfoHash {
	var h bencode.InfoHash
	copy(h[:], name)
	return h
}

// torrent adds a 4 GiB Torrent of name on the library's Node, added at,
// seeding complete, with up and down bytes, private or not, naming
// tracker, nil for none.
func (l *library) torrent(name string, tracker *indexers.TrackerID, private bool, at time.Time, up, down int64) downloadclients.ID {
	l.t.Helper()
	id := downloadclients.ID{UUID: storage.NewUUID()}
	report, err := json.Marshal(downloadclients.Report{At: at, Engine: downloadclients.EngineSeeding, Complete: true})
	if err != nil {
		l.t.Fatal(err)
	}
	h := hashOf(name)
	l.exec(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
		wanted, added_order, added_at, uploaded, downloaded, report, reported_at)
		VALUES (?, ?, ?, x'00', ?, ?, ?, '["tracker.test"]', 'fixture', ?, 'started',
		(SELECT coalesce(max(added_order), 0) + 1 FROM downloadclients_torrent), ?, ?, ?, ?, ?)`,
		id, l.node, h[:], name, int64(4*gib), private, tracker, storage.FromTime(at), up, down, string(report), storage.FromTime(at))
	return id
}

// grabOf plants tracker's Grab downloading into torrent, whose Release
// stated ratio and seed, each nil for none.
func (l *library) grabOf(tracker indexers.TrackerID, torrent downloadclients.ID, ratio *float64, seed *time.Duration) {
	l.t.Helper()
	var seconds *int64
	if seed != nil {
		seconds = ptr(int64(*seed / time.Second))
	}
	at := storage.Now()
	l.exec(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, tracker_id, node_id, release_title, release_size, release_magnet,
		min_ratio, min_seed_seconds, stage, torrent_id, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', ?, ?, 'downloading', ?, ?, ?)`,
		storage.NewUUID(), l.work, l.item, tracker, l.node, ratio, seconds, torrent, at, at)
}

// remove removes torrent at and parts it from its Node.
func (l *library) remove(torrent downloadclients.ID, at time.Time) {
	l.t.Helper()
	l.exec(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'admin', parted_at = ?, freed = 0 WHERE id = ?`,
		storage.FromTime(at), storage.FromTime(at), torrent)
}

// owing stores name's hash as owing tracker a ratio of 1.0 since opened,
// as a goals run would.
func (l *library) owing(name string, tracker indexers.TrackerID, opened time.Time) {
	l.t.Helper()
	h := hashOf(name)
	l.exec(`INSERT INTO trackers_window (info_hash, tracker_id, state, rule_clears, rule_ratio_centi, opened_at) VALUES (?, ?, 'owing', 'ratio', 100, ?)`,
		h[:], tracker, storage.FromTime(opened))
}

// rows is every live Torrent's row by name, as the Torrents view reads it.
func (l *library) rows() map[string]downloadclients.Row {
	l.t.Helper()
	listing, err := l.clients.List(l.t.Context(), downloadclients.Filter{})
	if err != nil {
		l.t.Fatal(err)
	}
	out := map[string]downloadclients.Row{}
	for _, row := range listing.Rows {
		out[row.Name] = row
	}
	return out
}

func (l *library) view(name string) seeding.View {
	l.t.Helper()
	var v seeding.View
	err := l.lib.Read(l.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		v, err = l.x.ViewOf(ctx, r, hashOf(name))
		return err
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return v
}

func TestAWindowOpensOwingAtAPrivateTorrentsFirstByte(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	nyaa := l.tracker("Nyaa", indexers.Public)
	start := time.Now().Add(-time.Hour)
	orchard := l.torrent("Orchard", &seedpool, true, start, 0, 0)
	l.grabOf(seedpool, orchard, nil, ptr(3*day))
	l.torrent("Harbor", &nyaa, false, start, 0, 4*gib)
	if w := l.rows()["Orchard"].Seeding.Window; w != nil {
		t.Fatalf("before its first byte Orchard reads window %+v, want none", w)
	}

	l.exec(`UPDATE downloadclients_torrent SET downloaded = 1 WHERE id = ?`, orchard)
	before := time.Now()
	rows := l.rows()
	s := rows["Orchard"].Seeding
	if w := s.Window; w == nil || w.State != seeding.Owing || w.Rule != (seeding.TimeRule{Seed: 3 * day}) || w.OpenedAt.Before(before) {
		t.Errorf("after its first byte Orchard reads window %+v, want one owing its Grab's 3 days, opened now", w)
	}
	if s.Risk != seeding.Safe || s.Owes == nil || s.Owes.Time != 3*day+trackers.Margin.Default {
		t.Errorf("Orchard reads risk %s owing %+v, want safe owing 3 days and the margin", s.Risk, s.Owes)
	}
	if h := rows["Harbor"].Seeding; h.Window != nil || h.Risk != seeding.Safe {
		t.Errorf("public Harbor reads window %+v and risk %s, want none and safe", h.Window, h.Risk)
	}
}

func TestARemovedHashReadsMissedAndARegrabReopensItKeepingTheMiss(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	now := time.Now().Truncate(time.Second)
	opened, removed, regrabbed := now.Add(-3*day), now.Add(-2*day), now.Add(-day)
	orchard := l.torrent("Orchard", &seedpool, true, opened, gib, 4*gib)
	l.owing("Orchard", seedpool, opened)

	l.remove(orchard, removed)
	if _, live := l.rows()["Orchard"]; live {
		t.Fatal("a removed Orchard still lists")
	}
	if w := l.view("Orchard").Window; w == nil || w.State != seeding.Missed || !w.ClosedAt.Equal(removed) || w.Owed.Bytes != 3*gib {
		t.Errorf("once removed Orchard reads window %+v, want missed at its removal owing 3 GiB of upload", w)
	}

	l.torrent("Orchard", &seedpool, true, regrabbed, gib, 4*gib)
	w := l.rows()["Orchard"].Seeding.Window
	if w == nil || w.State != seeding.Owing || !w.RegrabbedAt.Equal(regrabbed) || !w.ClosedAt.Equal(removed) || w.Owed.Bytes != 3*gib {
		t.Errorf("regrabbed Orchard reads window %+v, want owing again since the regrab, keeping the miss", w)
	}
}

func TestARowCarriesItsLanesWhatItOwesAndItsTracker(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	orchard := l.torrent("Orchard", &seedpool, true, time.Now().Add(-time.Hour), gib, 4*gib)
	l.grabOf(seedpool, orchard, ptr(1.0), nil)

	row := l.rows()["Orchard"]
	if row.Tracker == nil || *row.Tracker != (downloadclients.TrackerRef{ID: seedpool, Name: "seedpool"}) {
		t.Errorf("Orchard names Tracker %+v, want seedpool", row.Tracker)
	}
	s := row.Seeding
	wantRatio := seeding.Lane[seeding.Ratio]{Value: 25, Rule: ptr(seeding.Ratio(100)), Goal: ptr(seeding.Ratio(200))}
	wantTime := seeding.Lane[time.Duration]{Goal: ptr(14 * day)}
	if !reflect.DeepEqual(s.Ratio, wantRatio) || !reflect.DeepEqual(s.Time, wantTime) {
		t.Errorf("Orchard's lanes are ratio %s and time %s, want ratio 0.25 of the rule's 1.0 and the goal's 2.0, and time 0 of the goal's 14 days",
			lane(s.Ratio), lane(s.Time))
	}
	if s.Owes == nil || *s.Owes != (seeding.Owes{Bytes: 3 * gib, Clears: seeding.ClearsRatio}) || s.GoalMet {
		t.Errorf("Orchard owes %+v with goal met %v, want 3 GiB of upload and the goal not met", s.Owes, s.GoalMet)
	}
}

// TestATimeRuleIsMetByTheAnnouncedSecondsTheNodeReported stores what
// Orchard's Node reported of seedpool's credited seed time: short of its
// Grab's 3 days and the margin Orchard owes the rest, and past them its
// window is met.
func TestATimeRuleIsMetByTheAnnouncedSecondsTheNodeReported(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	orchard := l.torrent("Orchard", &seedpool, true, time.Now().Add(-4*day), gib, 4*gib)
	l.grabOf(seedpool, orchard, nil, ptr(3*day))
	report := func(announced time.Duration) {
		t.Helper()
		err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return downloadclients.MergeTotals(ctx, tx, role.NodeID{UUID: l.node}, orchard,
				downloadclients.Totals{Uploaded: gib, Downloaded: 4 * gib, Seeding: 4 * day, Announced: announced, At: time.Now()})
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	report(3 * day)
	s := l.rows()["Orchard"].Seeding
	if s.Time.Value != 3*day || s.Window == nil || s.Window.State != seeding.Owing || s.Owes == nil || s.Owes.Time != trackers.Margin.Default {
		t.Errorf("credited 3 days, Orchard reads time %s, window %+v, owing %+v; want 3 days, owing the margin", lane(s.Time), s.Window, s.Owes)
	}

	report(3*day + trackers.Margin.Default)
	s = l.rows()["Orchard"].Seeding
	if s.Time.Value != 3*day+trackers.Margin.Default || s.Window == nil || s.Window.State != seeding.Met || s.Owes != nil {
		t.Errorf("credited 3 days and the margin, Orchard reads time %s, window %+v, owing %+v; want its window met", lane(s.Time), s.Window, s.Owes)
	}
}

func lane[T any](l seeding.Lane[T]) string {
	b, err := json.Marshal(struct {
		Value      T
		Rule, Goal *T
	}{l.Value, l.Rule, l.Goal})
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func TestADrainCountsTheTorrentsWhoseGoalNeverActs(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	nyaa := l.tracker("Nyaa", indexers.Public)
	start := time.Now().Add(-time.Hour)
	l.torrent("Orchard", &seedpool, true, start, 0, 4*gib)
	l.torrent("Attic", nil, true, start, 0, 4*gib)
	l.torrent("Harbor", &nyaa, false, start, 0, 4*gib)

	var held nodesHeld
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		h, err := l.clients.DownloadingOwner().Held(ctx, r, role.NodeID{UUID: l.node})
		held = nodesHeld{h.Units, h.EndlessUnits}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if held != (nodesHeld{units: 3, endless: 2}) {
		t.Errorf("the Node holds %+v, want 3 torrents, 2 of them endless: one owing seedpool with no rule and one with no known Tracker", held)
	}
}

type nodesHeld struct{ units, endless int64 }
