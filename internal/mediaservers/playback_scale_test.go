//go:build scale

package mediaservers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	scaleServers = 10
	scalePlays   = 50000
	scaleWorks   = 5000
	// span is how far back each server's plays start: inside the default
	// 30 days of Keep play history for.
	span = 29 * 24 * time.Hour
	// rarePlays is the plays of the player car on each server, one in
	// 1,000, so its page scans the server's whole range.
	rarePlays = scalePlays / 1000
)

const (
	reportBudget = time.Second
	pageBudget   = 100 * time.Millisecond
)

// shapes are the files the Works hold, by Work index modulo four.
var shapes = []struct {
	rec    media.Record
	target string
}{
	{media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 8_000_000,
		Video: media.Video{Codec: "h264", Profile: "High", Width: 1920, Height: 1080, BitDepth: 8, HDR: []release.HDR{"SDR"}},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
	}, "1080p"},
	{media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 10_000_000,
		Video: media.Video{Codec: "hevc", Profile: "Main 10", Width: 1920, Height: 1080, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
		Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
	}, "1080p"},
	{media.Record{
		Version: 1, Container: "matroska,webm", Bitrate: 40_000_000,
		Video: media.Video{
			Codec: "hevc", Profile: "Main 10", Width: 3840, Height: 2160, BitDepth: 10,
			HDR: []release.HDR{"DV", "HDR10"}, DolbyVision: &media.DolbyVision{Profile: 8, Compatibility: 1},
		},
		Audio:     []media.Audio{{Codec: "truehd", Channels: 8, Language: "en", Default: true}},
		Subtitles: []media.Subtitle{{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"}},
	}, "2160p"},
	{media.Record{
		Version: 1, Container: "mov,mp4,m4a,3gp,3g2,mj2", Bitrate: 4_000_000,
		Video: media.Video{Codec: "h264", Profile: "Main", Width: 1280, Height: 720, BitDepth: 8, HDR: []release.HDR{"SDR"}},
		Audio: []media.Audio{{Codec: "aac", Channels: 2, Language: "en", Default: true}},
	}, "720p"},
}

// scaleLibrary is the 5,000 Works, each with one Item held by its played
// file and an alternate of the next shape, every fifth alternate Kept.
type scaleLibrary struct {
	works   []works.WorkID
	items   []works.ItemID
	played  []files.ID
	targets []targets.Target
	files   map[works.ItemID][]playback.Sibling
	goals   map[works.WorkID]playback.Goal
}

func newScaleLibrary() *scaleLibrary {
	profile := playback.TargetRef{ID: storage.NewUUID().String(), Name: "Any"}
	l := &scaleLibrary{files: map[works.ItemID][]playback.Sibling{}, goals: map[works.WorkID]playback.Goal{}}
	refs := map[string]playback.TargetRef{}
	goal := playback.Goal{Profile: profile}
	for _, tier := range []struct {
		name string
		qs   []release.Quality
	}{
		{"2160p", []release.Quality{release.WEBDL2160p, release.Bluray2160p}},
		{"1080p", []release.Quality{release.WEBDL1080p, release.Bluray1080p}},
		{"720p", []release.Quality{release.WEBDL720p}},
	} {
		t := targets.Target{
			ID: targets.TargetID{UUID: storage.NewUUID()}, Name: tier.name,
			Filters: targets.Filters{
				HDR: targets.Filter[release.HDR]{Mode: targets.Any}, Video: targets.Filter[release.VideoCodec]{Mode: targets.Any},
				Audio: targets.Filter[release.AudioCodec]{Mode: targets.Any}, Channels: targets.Filter[release.Channels]{Mode: targets.Any},
				Language: targets.Filter[release.Language]{Mode: targets.Any},
			},
			Profiles: []targets.ProfileRef{{Name: profile.Name}},
		}
		for _, q := range tier.qs {
			t.Tiers = append(t.Tiers, targets.Tier{{Quality: q}})
		}
		l.targets = append(l.targets, t)
		refs[tier.name] = playback.TargetRef{ID: t.ID.String(), Name: t.Name}
		goal.Targets = append(goal.Targets, t.ID)
	}
	for k := range scaleWorks {
		w, it := works.WorkID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
		f, alt := files.ID{UUID: storage.NewUUID()}, files.ID{UUID: storage.NewUUID()}
		l.works, l.items, l.played = append(l.works, w), append(l.items, it), append(l.played, f)
		own, next := shapes[k%len(shapes)], shapes[(k+1)%len(shapes)]
		l.files[it] = []playback.Sibling{
			{File: f, Work: w, Label: own.target, Met: []playback.TargetRef{refs[own.target]}, Media: own.rec},
			{File: alt, Work: w, Label: next.target, Kept: k%5 == 0, Met: []playback.TargetRef{refs[next.target]}, Media: next.rec},
		}
		l.goals[w] = goal
	}
	return l
}

func (l *scaleLibrary) snapshot(k int) playback.Snapshot {
	sib := l.files[l.items[k]][0]
	return playback.Snapshot{Work: sib.Work, Item: l.items[k], Label: sib.Label, Targets: sib.Met, Facts: playback.FactsOf(sib.Media, nil, nil)}
}

func (l *scaleLibrary) siblings(_ context.Context, _ storage.Reader[storage.Library], items []works.ItemID) (playback.Siblings, error) {
	out := playback.Siblings{Files: make(map[works.ItemID][]playback.Sibling, len(items)), Goals: map[works.WorkID]playback.Goal{}}
	for _, it := range items {
		sibs := l.files[it]
		out.Files[it] = sibs
		out.Goals[sibs[0].Work] = l.goals[sibs[0].Work]
	}
	return out, nil
}

func (l *scaleLibrary) allTargets(context.Context, storage.Reader[storage.Library]) ([]targets.Target, error) {
	return slices.Clone(l.targets), nil
}

// TestScalePlayback seeds 10 Jellyfin servers of 50,000 plays each over 29
// days, of 5,000 Works by eight players and car, one in 1,000. A quarter
// transcode, one in 17 of those with no file, one in 13 seen on one look,
// one in 997 still open. It times each server's report, a page of plays
// with and without a player, and a prune of what's older than 14 days.
func TestScalePlayback(t *testing.T) {
	lib := storagetest.Library(t, mstest.Schemas()...)
	l := newScaleLibrary()
	now := time.Now()
	servers := seedPlays(t, lib, l, now)
	ms := mediaservers.New(mediaservers.Config{
		Library: lib, Timings: mstest.Timings, Siblings: l.siblings, Targets: l.allTargets,
		Facts: func(role.NodeID) (*msv1.Facts, bool) { return nil, false },
		Batch: housekeeping.DefaultTimings().PruneBatch, Log: slog.New(slog.DiscardHandler),
	})
	svc := ms.Service()

	var took []time.Duration
	for _, id := range servers {
		start := time.Now()
		rep, err := svc.Playback(t.Context(), id)
		d := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Plays != scalePlays || len(rep.Players) != 9 {
			t.Fatalf("server %s's report covers %d plays by %d players, want %d by 9", id, rep.Plays, len(rep.Players), scalePlays)
		}
		groups := 0
		for _, p := range rep.Players {
			groups += len(p.Groups)
		}
		t.Logf("a report of %d plays, %d transcoded, %d unmatched, %d groups: %s", rep.Plays, rep.Transcoded, rep.Unmatched, groups, d)
		took = append(took, d)
	}
	logSpread(t, "a report", took, reportBudget)

	first, err := svc.Plays(t.Context(), servers[0], "", link.PlayID{})
	if err != nil {
		t.Fatal(err)
	}
	next, err := storage.ParseUUID(first.Next)
	if err != nil {
		t.Fatal(err)
	}
	after := link.PlayID{UUID: next}
	for _, c := range []struct {
		name, player string
		cursor       link.PlayID
		want         int
	}{
		{"every player", "", link.PlayID{}, 50},
		{"every player after a cursor", "", after, 50},
		{"p3", "p3", link.PlayID{}, 50},
		{"car, one play in 1,000", "car", link.PlayID{}, rarePlays},
		{"a player with no plays", "gone", link.PlayID{}, 0},
	} {
		took = took[:0]
		for range 5 {
			start := time.Now()
			p, err := svc.Plays(t.Context(), servers[0], c.player, c.cursor)
			d := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Plays) != c.want {
				t.Fatalf("a page of %s holds %d plays, want %d", c.name, len(p.Plays), c.want)
			}
			took = append(took, d)
		}
		logSpread(t, "a page of "+c.name, took, pageBudget)
	}

	cutoff := now.Add(-14 * 24 * time.Hour)
	before := scaleCount(t, lib, `SELECT COUNT(*) FROM mediaservers_play`)
	scaleExec(t, lib, `UPDATE mediaservers_settings SET keep_plays_seconds = ?`, int64(14*24*time.Hour/time.Second))
	task := housekeeping.Task(housekeeping.Config{Library: lib, Steps: []housekeeping.Step{ms.Housekeeping()}})
	start := time.Now()
	if err := task.Run(t.Context(), tasks.Unit{Key: "media-server-history"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
	d := time.Since(start)
	if old := scaleCount(t, lib, `SELECT COUNT(*) FROM mediaservers_play WHERE ended_at < ?`, storage.FromTime(cutoff)); old != 0 {
		t.Errorf("the prune left %d plays that ended before its cutoff", old)
	}
	left := scaleCount(t, lib, `SELECT COUNT(*) FROM mediaservers_play`)
	t.Logf("a full prune of %d plays to %d in %d batches: %s", before-left, left, (before-left+999)/1000, d)
}

// seedPlays writes the servers, the Works, each server's index of the
// Works' files and its plays in one Write, then analyzes, and answers
// the servers' ids.
func seedPlays(t *testing.T, lib *storage.DB[storage.Library], l *scaleLibrary, now time.Time) []link.ServerID {
	t.Helper()
	node := role.NodeID{UUID: storage.NewUUID()}
	var servers []link.ServerID
	type file struct {
		Work     string `json:"work"`
		Item     string `json:"item"`
		Played   string `json:"played"`
		Alt      string `json:"alt"`
		Snapshot string `json:"snapshot"`
	}
	fs := make([]file, scaleWorks)
	for k := range scaleWorks {
		snap, err := l.snapshot(k).Value()
		if err != nil {
			t.Fatal(err)
		}
		fs[k] = file{
			Work: hexOf(l.works[k].Bytes()), Item: hexOf(l.items[k].Bytes()), Played: hexOf(l.played[k].Bytes()),
			Alt: hexOf(l.files[l.items[k]][1].File.Bytes()), Snapshot: snap.(string),
		}
	}
	filesJSON, err := json.Marshal(fs)
	if err != nil {
		t.Fatal(err)
	}
	reasons, err := json.Marshal([][]string{
		{"VideoCodecNotSupported"}, {"VideoRangeTypeNotSupported"}, {"AudioCodecNotSupported"}, {"AudioChannelsNotSupported"},
		{"ContainerBitrateExceedsLimit"}, {"VideoResolutionNotSupported", "VideoBitDepthNotSupported"}, {},
	})
	if err != nil {
		t.Fatal(err)
	}
	received, err := link.Received{Container: "mpegts", VideoCodec: "h264", Height: 1080, Bitrate: 8_000_000, AudioCodec: "aac", Channels: 2}.Value()
	if err != nil {
		t.Fatal(err)
	}
	step := int64(span/time.Microsecond) / scalePlays
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, node, "home"); err != nil {
			return err
		}
		for n := range scaleServers {
			id := link.ServerID{UUID: storage.NewUUID()}
			servers = append(servers, id)
			if _, err := tx.ExecContext(ctx, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, monitor_delay_ms, reaching_node_id, device_id, secrets_sealed, created_at, updated_at)
				VALUES (?, 'jellyfin', ?, 'http://192.168.1.20:8096', ?, '10.11.0', 0, ?, ?, x'00', 1, 1)`,
				id, fmt.Sprintf("Den %d", n), fmt.Sprintf("identity-%d", n), node, fmt.Sprintf("device-%d", n)); err != nil {
				return err
			}
		}
		ids := make([]string, len(servers))
		for i, s := range servers {
			ids[i] = hexOf(s.Bytes())
		}
		serversJSON, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`CREATE TEMP TABLE scale_server (n INTEGER PRIMARY KEY, id BLOB NOT NULL)`, nil},
			{`INSERT INTO scale_server SELECT key, unhex(value) FROM json_each(?)`, []any{string(serversJSON)}},
			{`CREATE TEMP TABLE scale_file (k INTEGER PRIMARY KEY, work BLOB NOT NULL, item BLOB NOT NULL, played BLOB NOT NULL, alt BLOB NOT NULL, snapshot TEXT NOT NULL)`, nil},
			{`INSERT INTO scale_file SELECT key, unhex(value->>'work'), unhex(value->>'item'), unhex(value->>'played'), unhex(value->>'alt'), value->>'snapshot' FROM json_each(?)`, []any{string(filesJSON)}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at)
				SELECT work, 'movie', 'w' || k, 'Work ' || k, 'Work ' || k, 1, 1 FROM scale_file`, nil},
			{`INSERT INTO mediaservers_item (server_id, path_key, server_path, library_key, file_id, item_key, top_key, seen_at)
				SELECT s.id, '/shelves/w' || f.k || '.mkv', '/shelves/w' || f.k || '.mkv', 'films', f.played, 'w' || f.k, 'w' || f.k, ?1
				FROM scale_server s, scale_file f
				UNION ALL
				SELECT s.id, '/shelves/w' || f.k || ' alt.mkv', '/shelves/w' || f.k || ' alt.mkv', 'films', f.alt, 'a' || f.k, 'a' || f.k, ?1
				FROM scale_server s, scale_file f WHERE f.k % 2 = 0`, []any{storage.FromTime(now)}},
			{`WITH RECURSIVE n(i) AS (SELECT 0 UNION ALL SELECT i + 1 FROM n WHERE i < ?1 - 1),
				p AS (
					SELECT i % ?2 AS s, i / ?2 AS j, (i / ?2 * 7919) % ?3 AS k,
						?4 - ?5 * ?6 + (i / ?2) * ?6 AS started
					FROM n
				),
				q AS (
					SELECT p.*,
						CASE WHEN j % 4 = 0 THEN 'transcode' WHEN j % 4 = 1 THEN 'direct_stream' ELSE 'direct_play' END AS decision,
						CASE WHEN j % 4 = 0 AND (j / 4) % 3 <> 1 THEN 'transcode' ELSE 'direct' END AS video,
						CASE WHEN j % 4 = 0 AND (j / 4) % 3 <> 0 THEN 'transcode' ELSE 'direct' END AS audio,
						j % 4 = 0 AND j % 17 = 0 AS unmatched,
						CASE WHEN j % 13 = 0 THEN started ELSE started + 300000000 END AS seen
					FROM p
				)
				INSERT INTO mediaservers_play (
					server_id, id, session_key, server_path, path_key, item_keys, player_key, player, platform, product, user_name,
					decision, video, audio, subtitles, reasons, received, file_id, snapshot, started_at, last_seen_at, ended_at
				)
				SELECT ss.id, randomblob(16), 's' || q.j, '/shelves/w' || q.k || '.mkv', '/shelves/w' || q.k || '.mkv', json_array('w' || q.k),
					CASE WHEN q.j % 1000 = 999 THEN 'car' ELSE 'p' || (q.j % 8) END,
					CASE WHEN q.j % 1000 = 999 THEN 'Car' ELSE 'Player ' || (q.j % 8) END,
					'Android TV', 'Jellyfin Android TV', 'ada',
					q.decision, q.video, q.audio, 'none',
					CASE WHEN q.decision = 'transcode' THEN (SELECT value FROM json_each(?7) WHERE key = q.j % 7) ELSE '[]' END,
					CASE WHEN q.decision = 'transcode' THEN ?8 ELSE '{}' END,
					CASE WHEN q.unmatched THEN NULL ELSE f.played END,
					CASE WHEN q.unmatched THEN NULL ELSE f.snapshot END,
					q.started, q.seen, CASE WHEN q.j % 997 = 0 THEN NULL ELSE q.seen END
				FROM q JOIN scale_server ss ON ss.n = q.s JOIN scale_file f ON f.k = q.k`,
				[]any{scaleServers * scalePlays, scaleServers, scaleWorks, storage.FromTime(now), scalePlays, step, string(reasons), received}},
			{`DROP TABLE scale_server`, nil},
			{`DROP TABLE scale_file`, nil},
			{`ANALYZE`, nil},
		} {
			if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
				return fmt.Errorf("%.60s: %w", q.sql, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := scaleCount(t, lib, `SELECT COUNT(*) FROM mediaservers_play`); n != scaleServers*scalePlays {
		t.Fatalf("seeded %d plays, want %d", n, scaleServers*scalePlays)
	}
	return servers
}

func hexOf(b []byte) string { return fmt.Sprintf("%X", b) }

// logSpread logs the median and range of took and fails when the slowest
// is over budget.
func logSpread(t *testing.T, what string, took []time.Duration, budget time.Duration) {
	t.Helper()
	s := slices.Sorted(slices.Values(took))
	t.Logf("%s: median %s, range %s to %s over %d runs", what, s[len(s)/2], s[0], s[len(s)-1], len(s))
	if s[len(s)-1] > budget {
		t.Errorf("%s took up to %s, over %s", what, s[len(s)-1], budget)
	}
}

func scaleCount(t *testing.T, lib *storage.DB[storage.Library], q string, args ...any) int {
	t.Helper()
	var n int
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, q, args...).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func scaleExec(t *testing.T, lib *storage.DB[storage.Library], q string, args ...any) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, q, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
