package subtitles_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

var admin = identity.Admin{}

// film is a World whose one movie wants English, its audio scanned and
// SubDL keyed.
func film(t *testing.T) (*subtitlestest.World, subtitles.WantKey) {
	t.Helper()
	w := world(t, nil)
	en := english(t)
	f := w.Place(w.Work("film", w.Profile("Films", en)), "Film/Film.2024.1080p.WEB-DL-GRP.mkv", audio("fr"))
	w.Sweep()
	keyOn(t, w)
	return w, subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}}
}

func keyOn(t *testing.T, w *subtitlestest.World) {
	t.Helper()
	w.Holder = subtitlestest.KeyHolder{Account: "film-fan", Plan: "Free"}
	if _, err := w.Subs.SetKey(t.Context(), admin, "subdl", "k1"); err != nil {
		t.Fatal(err)
	}
	w.Settle()
}

// offer lists a SubDL English offer of key made for the file's release,
// serving body.
func offer(t *testing.T, w *subtitlestest.World, key, body string, downloads int) {
	t.Helper()
	w.Catalog.List(subtitlestest.Listing{
		Offer: subtitling.Offer{Provider: "subdl", Key: key, Language: english(t).Language, Releases: []string{"Film.2024.1080p.WEB-DL-GRP"}, Downloads: downloads},
		Name:  key + ".srt", Bytes: []byte(body),
	})
}

func searchNow(t *testing.T, w *subtitlestest.World, ks ...subtitles.WantKey) {
	t.Helper()
	for _, k := range ks {
		if err := w.Subs.SearchNowFor(t.Context(), admin, k); err != nil {
			t.Fatal(err)
		}
	}
	w.Settle()
}

// tried is each Tried before row as "key outcome".
func tried(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT candidate_key || ' ' || outcome FROM subtitles_tried ORDER BY candidate_key`)
}

// want is the want row's last result and gap.
func want(t *testing.T, w *subtitlestest.World, k subtitles.WantKey) string {
	t.Helper()
	got := query(t, w, `SELECT coalesce(last_result, '-') || ' ' || gap_seconds FROM subtitles_want WHERE file_id = ? AND language = ?`, k.File, k.Language.String())
	if len(got) != 1 {
		t.Fatalf("the want has rows %v", got)
	}
	return got[0]
}

func search(t *testing.T, w *subtitlestest.World, k subtitles.WantKey) {
	t.Helper()
	if err := subtitles.Search(t.Context(), w.Subs, k); err != nil {
		t.Fatal(err)
	}
	w.Settle()
}

func TestSubtitleSearchFetchesDownTheRankedListUntilOnePasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		offer(t, w, "first", againstSpeech, 30)
		offer(t, w, "second", againstSpeech, 20)
		offer(t, w, "third", onSpeech, 10)
		searchNow(t, w, k)
		if got, want := w.Catalog.Asked(), []string{"search subdl", "fetch subdl first", "fetch subdl second", "fetch subdl third"}; !slices.Equal(got, want) {
			t.Fatalf("the search asked %q, want %q", got, want)
		}
		if got := rows(t, w); !slices.Equal(got, []string{"Film/Film.2024.1080p.WEB-DL-GRP.mkv en provider"}) {
			t.Fatalf("the Sidecars are %v", got)
		}
		if got := tried(t, w); !slices.Equal(got, []string{"first failed_timing", "second failed_timing", "third passed"}) {
			t.Fatalf("Tried before holds %v", got)
		}
		if got := want(t, w, k); !strings.HasPrefix(got, "passed ") {
			t.Fatalf("the want reads %q, want passed", got)
		}
	})
}

// show is a World whose files are episodes of a Show of n wanting
// English, each episode its own file, with their audio scanned.
func show(t *testing.T, n int) (*subtitlestest.World, []subtitles.WantKey) {
	t.Helper()
	w := world(t, nil)
	w.Episodes()
	en := english(t)
	tree := w.Show("show", w.Profile("Shows", en), n)
	var ks []subtitles.WantKey
	for i, it := range tree.Items {
		f := w.PlaceItems(tree, []works.ItemID{it.ID}, fmt.Sprintf("Show/Show.S01E%02d.mkv", i+1), audio("fr"))
		ks = append(ks, subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}})
	}
	w.Sweep()
	return w, ks
}

func episodeOffer(t *testing.T, w *subtitlestest.World, key string, episode int, body string) {
	t.Helper()
	w.Catalog.List(subtitlestest.Listing{
		Offer: subtitling.Offer{Provider: "gestdown", Key: key, Language: english(t).Language, Season: 1, Episodes: []int{episode}},
		Name:  fmt.Sprintf("Show.S01E%02d.srt", episode), Bytes: []byte(body),
	})
}

func TestGestdownRefreshingAShowKeepsTheGapAndTheNextSearchPasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, ks := show(t, 1)
		episodeOffer(t, w, "e1", 1, onSpeech)
		before := want(t, w, ks[0])
		w.Catalog.Answer("gestdown", calls.Outcome{Kind: calls.Unreachable, Words: "is refreshing the show"})
		search(t, w, ks[0])
		if got, gap := want(t, w, ks[0]), strings.Fields(before)[1]; got != "unreachable "+gap {
			t.Fatalf("after a refresh the want reads %q, want unreachable with its gap %s kept", got, gap)
		}
		if got := query(t, w, `SELECT last_error FROM subtitles_provider WHERE id = 'gestdown' AND failing_since IS NOT NULL`); !slices.Equal(got, []string{"is refreshing the show"}) {
			t.Fatalf("Gestdown's row reads %v", got)
		}
		search(t, w, ks[0])
		if got := rows(t, w); !slices.Equal(got, []string{"Show/Show.S01E01.mkv en provider"}) {
			t.Fatalf("the next search added %v", got)
		}
		if got := query(t, w, `SELECT id FROM subtitles_provider WHERE failing_since IS NOT NULL`); len(got) != 0 {
			t.Fatalf("after a response %v still read failing", got)
		}
	})
}

func pack(t *testing.T, n int) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for i := range n {
		f, err := z.Create(fmt.Sprintf("Show.S01E%02d.srt", i+1))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(onSpeech)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestASeasonPackServesTenEpisodesWithOneDownload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, ks := show(t, 10)
		w.Catalog.List(subtitlestest.Listing{
			Offer: subtitling.Offer{Provider: "gestdown", Key: "season", Language: english(t).Language, Season: 1, Pack: true},
			Name:  "Show.S01.zip", Bytes: pack(t, 10),
		})
		searchNow(t, w, ks[0])
		for _, k := range ks[1:] {
			if err := w.Subs.SearchNowFor(t.Context(), admin, k); err != nil {
				t.Fatal(err)
			}
		}
		w.Settle()
		if got, want := w.Catalog.Asked(), []string{"search gestdown", "fetch gestdown season"}; !slices.Equal(got, want) {
			t.Fatalf("ten episodes asked %q, want %q", got, want)
		}
		if got := rows(t, w); len(got) != 10 {
			t.Fatalf("the pack gave Sidecars %v, want all ten", got)
		}
	})
}

func TestAnEpisodeWithNoAiredNumberIsNeverSearched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		w.Episodes()
		en := english(t)
		tree := w.Show("show", w.Profile("Shows", en), 1)
		w.Unair(tree.Items[0].ID)
		f := w.Place(tree, "Show/Show.S01E01.mkv", audio("fr"))
		w.Sweep()
		k := subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}}
		if st := outcome(t, w, f, k.Want); st.Missing != subtitles.NoAiredNumber {
			t.Fatalf("the want reads %+v, want NoAiredNumber", st)
		}
		searchNow(t, w, k)
		if got := w.Catalog.Asked(); len(got) != 0 {
			t.Fatalf("an episode with no aired number asked %q", got)
		}
	})
}

// paused is each provider's pause as "id reason until".
func paused(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	ps, err := w.Subs.Providers(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range ps {
		if p.Paused != nil {
			out = append(out, fmt.Sprintf("%s %s %s", p.Spec.ID, p.Paused.Reason, p.Paused.Until.Format(time.DateTime)))
		}
	}
	return out
}

func at(d time.Duration) string { return time.Now().Add(d).Format(time.DateTime) }

func TestAKeyThatRunsOutPausesSubDLUntilItsResetAndSurvivesARestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		en := english(t)
		p := w.Profile("Films", en)
		var ks []subtitles.WantKey
		for _, name := range []string{"a", "b", "c"} {
			f := w.Place(w.Work(works.Key(name), p), name+"/"+name+".mkv", audio("fr"))
			ks = append(ks, subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}})
		}
		w.Sweep()
		keyOn(t, w)
		offer(t, w, "x", onSpeech, 1)
		reset := at(6 * time.Hour)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.OK}, calls.Outcome{Kind: calls.Capped, Words: "has no downloads left today", Until: time.Now().Add(6 * time.Hour)})
		searchNow(t, w, ks...)
		if got := paused(t, w); !slices.Equal(got, []string{"subdl quota " + reset}) {
			t.Fatalf("after the spent allowance the pauses are %v, want SubDL's until %s", got, reset)
		}
		if got := w.Catalog.Asked(); len(got) != 2 || len(rows(t, w)) != 0 {
			t.Fatalf("the drain asked %q, want nothing after the call that spent the allowance", got)
		}
		if got := query(t, w, `SELECT count(*) FROM subtitles_want WHERE queued_at IS NOT NULL AND attempt_id IS NULL`); !slices.Equal(got, []string{"3"}) {
			t.Fatalf("%v wants wait in the queue, want all three", got)
		}
		if got := entrytest.Of(w.History, subtitles.ProviderOutRow); len(got) != 1 || got[0].Provider != "SubDL" {
			t.Fatalf("the pause recorded %+v, want one Subtitle Provider out of downloads", got)
		}
		w.Restart()
		w.Sweep()
		if got := paused(t, w); !slices.Equal(got, []string{"subdl quota " + reset}) {
			t.Fatalf("after a restart the pauses are %v", got)
		}
		if len(w.Catalog.Asked()) != 2 {
			t.Fatalf("a paused SubDL was asked again after the restart: %q", w.Catalog.Asked())
		}
		time.Sleep(6 * time.Hour)
		w.Sweep()
		if got := rows(t, w); len(got) != 3 {
			t.Fatalf("after the reset the drain added %v, want all three", got)
		}
	})
}

func TestARefusedKeyPausesAndIsCheckedAgainWithinTheLongestPause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Refused, Words: "refused the key: Not Authorized"})
		bound := at(w.Timings.MaxPause)
		search(t, w, k)
		if got := paused(t, w); !slices.Equal(got, []string{"subdl key " + bound}) {
			t.Fatalf("a refused key paused %v", got)
		}
		if got := items(t, w); !slices.ContainsFunc(got, func(s string) bool {
			return strings.Contains(s, `SubDL rejected your key and responded "Not Authorized".`)
		}) {
			t.Fatalf("Health reads %v", got)
		}
		time.Sleep(w.Timings.MaxPause)
		w.Sweep()
		if got := paused(t, w); len(got) != 0 {
			t.Fatalf("a key the provider takes again stays paused %v", got)
		}
	})
}

func TestALyingHoldersPauseIsClamped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Capped, Until: time.Now().Add(30 * 24 * time.Hour)})
		bound := at(w.Timings.MaxPause)
		search(t, w, k)
		if got := paused(t, w); !slices.Equal(got, []string{"subdl quota " + bound}) {
			t.Fatalf("a 30 day pause reads %v, want it clamped to a day", got)
		}
	})
}

func TestAPickWaitsForTheRunningSearchThenGoesFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		offer(t, w, "early", againstSpeech, 20)
		w.Catalog.Hold()
		if err := w.Subs.SearchNowFor(t.Context(), admin, k); err != nil {
			t.Fatal(err)
		}
		w.Settle()
		offer(t, w, "late", onSpeech, 10)
		id, err := w.Subs.StartCandidates(t.Context(), admin, k)
		if err != nil {
			t.Fatal(err)
		}
		w.Settle()
		attempt, err := w.Subs.Pick(t.Context(), admin, k, id, "subdl:late")
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Subs.SearchNowFor(t.Context(), admin, k); err != nil {
			t.Fatal(err)
		}
		w.Settle()
		if a, err := w.Subs.AttemptOf(t.Context(), admin, k.File, attempt); err != nil || a.Step != subtitles.Downloading {
			t.Fatalf("the waiting pick reads %+v, %v", a, err)
		}
		w.Catalog.Release()
		w.Settle()
		fetches := slices.DeleteFunc(w.Catalog.Asked(), func(s string) bool { return !strings.HasPrefix(s, "fetch") })
		if len(fetches) < 2 || fetches[0] != "fetch subdl early" || fetches[1] != "fetch subdl late" {
			t.Fatalf("the fetches ran %q, want the search's, then the pick's", fetches)
		}
		if a, err := w.Subs.AttemptOf(t.Context(), admin, k.File, attempt); err != nil || a.Ended != subtitles.Added {
			t.Fatalf("the pick reads %+v, %v, want added", a, err)
		}
	})
}

// reads is what a screen reads of k each time subtitles.changed names
// k's file, as it reads it again on the event.
func reads(t *testing.T, w *subtitlestest.World, k subtitles.WantKey) func() []subtitles.StateKind {
	t.Helper()
	var mu sync.Mutex
	var kinds []subtitles.StateKind
	w.Changed.Seen(func(c subtitles.Changed) {
		if !slices.Contains(c.FileIDs, k.File.String()) {
			return
		}
		err := w.Lib.Read(context.Background(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			st, err := w.Subs.Outcome(ctx, r, k.File, k.Want)
			mu.Lock()
			defer mu.Unlock()
			kinds = append(kinds, st.Kind)
			return err
		})
		if err != nil {
			t.Error(err)
		}
	})
	return func() []subtitles.StateKind {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(kinds)
	}
}

func TestAScreenReadingOnEachChangeSeesTheSubtitleAPickAdded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		seen := reads(t, w, k)
		offer(t, w, "x", onSpeech, 1)
		id, _ := candidates(t, w, k)
		if _, err := w.Subs.Pick(t.Context(), admin, k, id, "subdl:x"); err != nil {
			t.Fatal(err)
		}
		w.Settle()
		if w.On("Film/Film.2024.1080p.WEB-DL-GRP.en.srt") == "" {
			t.Fatal("the pick added no subtitle")
		}
		if got := seen(); len(got) == 0 || got[len(got)-1] != subtitles.Met {
			t.Fatalf("on each subtitles.changed the want read %v, want it to end met", got)
		}
	})
}

func TestAStaleAttemptEndsWithoutWriting(t *testing.T) {
	for name, take := range map[string]func(w *subtitlestest.World, tree works.Tree, p targets.ProfileID){
		"doomed":  func(w *subtitlestest.World, tree works.Tree, _ targets.ProfileID) { w.Doom(tree) },
		"dropped": func(w *subtitlestest.World, _ works.Tree, p targets.ProfileID) { w.Languages(p, "Films") },
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				en := english(t)
				p := w.Profile("Films", en)
				tree := w.Work("film", p)
				f := w.Place(tree, "Film/Film.mkv", audio("fr"))
				w.Sweep()
				keyOn(t, w)
				offer(t, w, "x", onSpeech, 1)
				w.Catalog.Hold()
				done := make(chan error, 1)
				go func() {
					done <- subtitles.Search(t.Context(), w.Subs, subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}})
				}()
				synctest.Wait()
				take(w, tree, p)
				w.Catalog.Release()
				if err := <-done; !errors.Is(err, subtitles.ErrStale) {
					t.Fatalf("the attempt = %v, want ErrStale", err)
				}
				w.Settle()
				if got := query(t, w, `SELECT hex(file_id) FROM subtitles_write UNION ALL SELECT candidate_key FROM subtitles_tried`); len(got) != 0 {
					t.Fatalf("the stale attempt wrote %v", got)
				}
				if got := w.On("Film/Film.en.srt"); got != "" {
					t.Fatalf("the stale attempt wrote %q", got)
				}
			})
		})
	}
}

func TestAHungFetchEndsAtTheAttemptBoundAndReleasesTheLease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		offer(t, w, "x", onSpeech, 1)
		w.Catalog.Hold()
		start := time.Now()
		if err := subtitles.Search(t.Context(), w.Subs, k); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("the hung attempt = %v, want its bound's end", err)
		}
		if took := time.Since(start); took != w.Timings.AttemptBound {
			t.Fatalf("the hung attempt took %v, want %v", took, w.Timings.AttemptBound)
		}
		if got := query(t, w, `SELECT hex(attempt_id) FROM subtitles_want WHERE attempt_id IS NOT NULL`); len(got) != 0 {
			t.Fatalf("the ended attempt left its lease %v", got)
		}
		w.Catalog.Release()
	})
}

func TestTheGapDoublesOnlyWhenAProviderAnswered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		gap := func() string { return strings.Fields(want(t, w, k))[1] }
		before := gap()
		search(t, w, k)
		if got := want(t, w, k); got != "nothing_found 57600" {
			t.Fatalf("after an answered search from a %s s gap the want reads %q, want 16 hours", before, got)
		}
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Unreachable, Words: "responded with 502 Bad Gateway"})
		search(t, w, k)
		if got := want(t, w, k); got != "unreachable 57600" {
			t.Fatalf("after an unanswered search the want reads %q, want the gap kept", got)
		}
		search(t, w, k)
		if got := want(t, w, k); got != "nothing_found 115200" {
			t.Fatalf("after another answered search the want reads %q, want 32 hours", got)
		}
	})
}

func runTask(t *testing.T, w *subtitlestest.World) error {
	t.Helper()
	return w.Subs.Task().Run(t.Context(), tasks.Unit{Key: "due"}, &tasks.Progress{})
}

func TestARunEndsSucceededWhileItsDrainWorks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, _ := film(t)
		offer(t, w, "x", onSpeech, 1)
		exec(t, w, `UPDATE subtitles_want SET due_at = 0`)
		w.Catalog.Hold()
		if err := runTask(t, w); err != nil {
			t.Fatalf("the run = %v, want it Succeeded while the drain works", err)
		}
		w.Settle()
		if got := query(t, w, `SELECT hex(attempt_id) FROM subtitles_want WHERE attempt_id IS NOT NULL`); len(got) != 1 {
			t.Fatalf("after the run ended the drain holds %v, want the want it works", got)
		}
		w.Catalog.Release()
		w.Settle()
		if got := rows(t, w); len(got) != 1 {
			t.Fatalf("the drain added %v", got)
		}
	})
}

func setSettings(t *testing.T, w *subtitlestest.World, values string) {
	t.Helper()
	var patch setting.Patch
	if err := json.Unmarshal([]byte(values), &patch); err != nil {
		t.Fatal(err)
	}
	if err := w.Subs.Settings().Apply(t.Context(), patch); err != nil {
		t.Fatal(err)
	}
}

func TestSearchOffSkipsTheRunWhileMapsStillBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		setSettings(t, w, `{"subtitles.searchOn":false}`)
		w.Place(w.Work("film", w.Profile("Films", english(t))), "Film/Film.mkv", audio("fr"))
		var skip *tasks.SkipError
		if err := runTask(t, w); !errors.As(err, &skip) || !skip.Owned || skip.Cause != "Subtitle search is off in Settings." {
			t.Fatalf("a run with search off = %v, want its owned skip", err)
		}
		w.Sweep()
		if got := maps(t, w); !slices.Equal(got, []string{"Film/Film.mkv built"}) {
			t.Fatalf("with search off the maps read %v", got)
		}
	})
}

func TestUpgradesOffQueuesOnlyMissingWants(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		met := w.Place(w.Work("met", p), "Met/Met.mkv", audio("fr"))
		w.Place(w.Work("missing", p), "Missing/Missing.mkv", audio("fr"))
		w.Sweep()
		intend(t, w, met, subtitles.FromProvider, onSpeech)
		w.Settle()
		w.Holders(false)
		exec(t, w, `UPDATE subtitles_want SET due_at = 0`)
		setSettings(t, w, `{"subtitles.searchUpgrades":false}`)
		queued := func() []string {
			return query(t, w, `SELECT f.path FROM subtitles_want w JOIN files_file f ON f.id = w.file_id WHERE w.queued_at IS NOT NULL ORDER BY f.path`)
		}
		if err := runTask(t, w); err != nil {
			t.Fatal(err)
		}
		if got := queued(); !slices.Equal(got, []string{"Missing/Missing.mkv"}) {
			t.Fatalf("with upgrades off the run queued %v", got)
		}
		setSettings(t, w, `{"subtitles.searchUpgrades":true}`)
		if err := runTask(t, w); err != nil {
			t.Fatal(err)
		}
		if got := queued(); !slices.Equal(got, []string{"Met/Met.mkv", "Missing/Missing.mkv"}) {
			t.Fatalf("with upgrades on the run queued %v", got)
		}
	})
}

func TestMarkBadRemovesTheSubtitleAndPickingItAgainUndoesIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		offer(t, w, "x", onSpeech, 1)
		searchNow(t, w, k)
		sidecar := "Film/Film.2024.1080p.WEB-DL-GRP.en.srt"
		if w.On(sidecar) == "" {
			t.Fatal("the search added no subtitle")
		}
		markBad(t, w, k)
		if got := w.On(sidecar); got != "" {
			t.Fatalf("Mark bad left %q", got)
		}
		if got := tried(t, w); !slices.Equal(got, []string{"x marked_bad"}) {
			t.Fatalf("Tried before holds %v", got)
		}
		if got := changes(w); !slices.Equal(got, []string{"added " + sidecar, "removed " + sidecar}) {
			t.Fatalf("the Media Servers heard %v", got)
		}
		if fetches := slices.DeleteFunc(w.Catalog.Asked(), func(s string) bool { return !strings.HasPrefix(s, "fetch") }); len(fetches) != 1 {
			t.Fatalf("the search after Mark bad fetched the marked one again: %q", fetches)
		}
		id, c := candidates(t, w, k)
		if len(c.Candidates) != 1 || c.Candidates[0].Disabled || c.Candidates[0].Tried == nil || c.Candidates[0].Tried.Outcome != subtitles.TriedMarkedBad {
			t.Fatalf("the candidates read %+v, want the marked one pickable with its Mark bad", c.Candidates)
		}
		if _, err := w.Subs.Pick(t.Context(), admin, k, id, c.Candidates[0].Key); err != nil {
			t.Fatal(err)
		}
		w.Settle()
		if w.On(sidecar) == "" {
			t.Fatal("picking the marked one again added nothing")
		}
		if got := tried(t, w); !slices.Equal(got, []string{"x passed"}) {
			t.Fatalf("after the undo Tried before holds %v", got)
		}
		bestIsTheSidecars(t, w)
	})
}

func TestALanguageDroppedAndAddedAgainFetchesTheSubtitleThatPassed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		en := english(t)
		p := w.Profile("Films", en)
		f := w.Place(w.Work("film", p), "Film/Film.2024.1080p.WEB-DL-GRP.mkv", audio("fr"))
		w.Sweep()
		keyOn(t, w)
		k := subtitles.WantKey{File: f, Want: subtitles.Want{Language: en.Language}}
		offer(t, w, "x", onSpeech, 1)
		searchNow(t, w, k)
		sidecar := "Film/Film.2024.1080p.WEB-DL-GRP.en.srt"
		w.Languages(p, "Films")
		if got := w.On(sidecar); got != "" {
			t.Fatalf("dropping English left %q", got)
		}
		w.Languages(p, "Films", en)
		searchNow(t, w, k)
		if w.On(sidecar) == "" {
			t.Fatalf("adding English again added no subtitle: the want reads %q after the calls %q", want(t, w, k), w.Catalog.Asked())
		}
		if got := tried(t, w); !slices.Equal(got, []string{"x passed"}) {
			t.Fatalf("Tried before holds %v", got)
		}
	})
}

// bestIsTheSidecars checks that the want's best match is its Sidecar's.
func bestIsTheSidecars(t *testing.T, w *subtitlestest.World) {
	t.Helper()
	got := query(t, w, `SELECT coalesce(w.best_match, '-') || ' ' || s.match FROM subtitles_want w JOIN subtitles_sidecar s USING (file_id, language, forced)`)
	if len(got) != 1 || strings.Fields(got[0])[0] != strings.Fields(got[0])[1] {
		t.Fatalf("the best match and the Sidecar's read %v, want the Sidecar's match as the best", got)
	}
}

func markBad(t *testing.T, w *subtitlestest.World, k subtitles.WantKey) {
	t.Helper()
	if err := w.Subs.MarkBadNow(t.Context(), admin, k); err != nil {
		t.Fatal(err)
	}
	w.Settle()
}

// candidates runs a Manual subtitle search on k and reads its candidates.
func candidates(t *testing.T, w *subtitlestest.World, k subtitles.WantKey) (storage.UUID, subtitles.Candidates) {
	t.Helper()
	id, err := w.Subs.StartCandidates(t.Context(), admin, k)
	if err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c, err := w.Subs.Candidates(t.Context(), admin, k, id)
	if err != nil {
		t.Fatal(err)
	}
	return id, c
}

func TestAManualSearchTellsHowEachProviderEndedOnceTheOneOutIncluded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, ks := show(t, 1)
		keyOn(t, w)
		episodeOffer(t, w, "e1", 1, onSpeech)
		reset := time.Now().Add(6 * time.Hour)
		w.Catalog.Answer("subdl", calls.Outcome{Kind: calls.Capped, Words: "has no downloads left today", Until: reset})
		candidates(t, w, ks[0])
		w.Catalog.Lookups("gestdown")
		w.Catalog.Answer("gestdown", calls.Outcome{Kind: calls.NotFound}, calls.Outcome{Kind: calls.OK})
		_, c := candidates(t, w, ks[0])
		var got []string
		for _, o := range c.Outcomes {
			got = append(got, fmt.Sprintf("%s %d %s", o.Provider, o.Kind, o.Until.Format(time.DateTime)))
		}
		if want := []string{fmt.Sprintf("subdl %d %s", calls.Capped, reset.Format(time.DateTime)), fmt.Sprintf("gestdown %d %s", calls.OK, time.Time{}.Format(time.DateTime))}; !slices.Equal(got, want) {
			t.Fatalf("the search's outcomes read %q, want %q", got, want)
		}
	})
}

type keepDefault struct{}

func (keepDefault) Fields(context.Context) ([]setting.Field, error) { return nil, nil }
func (keepDefault) Apply(context.Context, setting.Patch) error      { return nil }
func (keepDefault) Keep(context.Context) (time.Duration, error) {
	return history.KeepFor.Default, nil
}

func TestThePruneOfTriedBeforeKeepsAMarkBad(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, k := film(t)
		old, recent := time.Now().Add(-history.KeepFor.Default-24*time.Hour).UnixMicro(), time.Now().UnixMicro()
		for _, row := range []struct {
			key, outcome string
			at           int64
		}{{"marked", "marked_bad", old}, {"old", "failed_timing", old}, {"recent", "failed_timing", recent}} {
			exec(t, w, `INSERT INTO subtitles_tried (file_id, language, forced, candidate_key, outcome, tried_at) VALUES (?, ?, 0, ?, ?, ?)`,
				k.File, k.Language.String(), row.key, row.outcome, row.at)
		}
		task := housekeeping.Task(housekeeping.Config{Keep: keepDefault{}, Batch: 1000, Steps: []housekeeping.Step{w.Subs.Retention()}})
		if err := task.Run(t.Context(), tasks.Unit{Key: "subtitle-tries"}, &tasks.Progress{}); err != nil {
			t.Fatal(err)
		}
		if got := tried(t, w); !slices.Equal(got, []string{"marked marked_bad", "recent failed_timing"}) {
			t.Fatalf("after the prune Tried before holds %v", got)
		}
	})
}

func TestAFreshImportSearchesOnceItsAudioIsScannedAndToldHistory(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		keyOn(t, w)
		en := english(t)
		f := w.Place(w.Work("film", w.Profile("Films", en)), "Film/Film.mkv", audio("fr"))
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return w.Subs.Landed(ctx, tx, files.File{ID: f})
		})
		if got := w.Catalog.Asked(); len(got) != 0 {
			t.Fatalf("before its audio was scanned the import asked %q", got)
		}
		w.Sweep()
		if got := w.Catalog.Asked(); !slices.Equal(got, []string{"search subdl"}) {
			t.Fatalf("once scanned the import asked %q", got)
		}
		if got := entrytest.Of(w.History, subtitles.MissingRow); len(got) != 1 || got[0].Language != "English" || got[0].File != "Film/Film.mkv" {
			t.Fatalf("the first search recorded %+v, want one Subtitles missing", got)
		}
	})
}
