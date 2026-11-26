package mediaserversapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

type playWork struct{ WorkID, Title, Label string }

type listed struct {
	Plays []struct {
		ID        string
		PlayerKey string
		Work      *playWork
	}
	Next *string
}

func (w watched) getPlays(t *testing.T, query url.Values) (status int, body string) {
	t.Helper()
	return w.o.Call(t, http.MethodGet, servers+"/"+w.id+"/plays?"+query.Encode(), "")
}

func (w watched) listed(t *testing.T, query url.Values) listed {
	t.Helper()
	status, body := w.getPlays(t, query)
	var got listed
	if err := json.Unmarshal([]byte(body), &got); status != http.StatusOK || err != nil {
		t.Fatalf("GET plays?%s = %d %s (%v)", query.Encode(), status, body, err)
	}
	return got
}

// walk follows every page of the plays of player, empty for every
// player, and answers each play's id and each page's size.
func (w watched) walk(t *testing.T, player string) (ids []string, sizes []int) {
	t.Helper()
	query := url.Values{}
	if player != "" {
		query.Set("player", player)
	}
	for {
		page := w.listed(t, query)
		if len(page.Plays) > 0 {
			sizes = append(sizes, len(page.Plays))
		}
		for _, p := range page.Plays {
			ids = append(ids, p.ID)
		}
		if page.Next == nil {
			return ids, sizes
		}
		query.Set("cursor", *page.Next)
	}
}

type startedPlay struct {
	id     link.PlayID
	at     time.Time
	player string
}

// startPlays starts n plays of Clip on Den, two to each second ending a
// minute ago, alternating between lrtv and bdtv, and answers them newest
// first, ties by id.
func (w watched) startPlays(t *testing.T, n int) []startedPlay {
	t.Helper()
	server := serverOf(t, w.id)
	base := time.Now().Add(-time.Minute - time.Duration(n/2)*time.Second)
	var all []startedPlay
	for i := range n {
		s := startedPlay{at: base.Add(time.Duration(i/2) * time.Second), player: []string{"lrtv", "bdtv"}[i%2]}
		s.id = w.w.Play(t, w.w.Seedbox, mstest.Play{
			Server: server, Session: "9", Path: clip, PlayerKey: s.player, Player: s.player, User: "ada",
			Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone},
			Started: s.at, Seen: s.at,
		})
		all = append(all, s)
	}
	slices.SortFunc(all, func(a, b startedPlay) int {
		if c := b.at.Compare(a.at); c != 0 {
			return c
		}
		return bytes.Compare(b.id.Bytes(), a.id.Bytes())
	})
	return all
}

// TestPlaysPageNewestFirst starts 55 plays on Den across two players and
// walks every page for each player key: newest first, ties by id, 50 a
// page, and none for a key no play has.
func TestPlaysPageNewestFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		wait(time.Hour)
		all := w.startPlays(t, 55)
		for _, tc := range []struct {
			player string
			sizes  []int
		}{{"", []int{50, 5}}, {"lrtv", []int{28}}, {"nobody", nil}} {
			var want []string
			for _, s := range all {
				if tc.player == "" || s.player == tc.player {
					want = append(want, s.id.String())
				}
			}
			got, sizes := w.walk(t, tc.player)
			if !slices.Equal(got, want) {
				t.Errorf("player %q lists %v, want %v", tc.player, got, want)
			}
			if !slices.Equal(sizes, tc.sizes) {
				t.Errorf("player %q pages hold %v plays, want %v", tc.player, sizes, tc.sizes)
			}
		}
	})
}

// TestACursorThatNamesNoPlayIsPlayNotFound pages Den's plays after a
// play id Den never had.
func TestACursorThatNamesNoPlayIsPlayNotFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		status, body := w.getPlays(t, url.Values{"cursor": {storage.NewUUID().String()}})
		expectProblem(t, status, body, http.StatusNotFound, `"code":"play_not_found"`)
	})
}

// shownWorks is each play's and each session's work as GET plays and Now
// playing name it, "<workId> <title> <label>".
func (s snapped) shownWorks(t *testing.T) (plays, sessions map[string]string) {
	t.Helper()
	named := func(w *playWork) string {
		if w == nil {
			return ""
		}
		return w.WorkID + " " + w.Title + " " + w.Label
	}
	plays, sessions = map[string]string{}, map[string]string{}
	for _, p := range s.listed(t, url.Values{}).Plays {
		plays[p.ID] = named(p.Work)
	}
	_, body := s.o.Call(t, http.MethodGet, servers+"/"+s.id+"/sessions", "")
	var np nowPlaying
	if err := json.Unmarshal([]byte(body), &np); err != nil {
		t.Fatal(err)
	}
	for _, ses := range np.Sessions {
		sessions[ses.PlayID] = named(ses.Work)
	}
	return plays, sessions
}

// TestAPlayNamesItsWorkOnceItHasASnapshot plays Clip, which Den indexes
// as Heat's file, and Other, which no index names: the plays list and
// Now playing name Heat for Clip's play and no Work for Other's.
func TestAPlayNamesItsWorkOnceItHasASnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSnapped(t)
		err := s.w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'movie', 'heat', 'Heat', 'Heat', 0, 0)", s.work)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		heat := s.w.Play(t, s.w.Seedbox, s.play(nil, nil))
		other := s.play(nil, nil)
		other.Session, other.Path = "10", shelves+"/Other.mkv"
		unsnapped := s.w.Play(t, s.w.Seedbox, other)
		now := time.Now().UnixMilli()
		session := func(id link.PlayID, key, path string) *msv1.Session {
			return &msv1.Session{
				PlayId: id.Bytes(), SessionKey: key, ServerPath: path, User: "ada", Player: "Living room",
				StartedUnixMs: now, SeenUnixMs: now, Streams: &msv1.Streams{Video: "transcode", Audio: "direct", Subtitles: "none"},
			}
		}
		s.w.SetFacts(s.w.Seedbox, factsOf(s.server, session(heat, "9", clip), session(unsnapped, "10", other.Path)))
		synctest.Wait()
		plays, sessions := s.shownWorks(t)
		for _, tc := range []struct {
			id                link.PlayID
			play, sessionWork string
		}{
			{heat, s.work.String() + " Heat 1080p", s.work.String() + " Heat 1080p"},
			{unsnapped, "", ""},
		} {
			if got := plays[tc.id.String()]; got != tc.play {
				t.Errorf("GET plays names play %s's work %q, want %q", tc.id, got, tc.play)
			}
			if got, ok := sessions[tc.id.String()]; !ok || got != tc.sessionWork {
				t.Errorf("Now playing names play %s's work %q (listed %t), want %q", tc.id, got, ok, tc.sessionWork)
			}
		}
	})
}

type nowPlaying struct {
	Sessions []struct {
		PlayID string
		Work   *playWork
	}
}

// TestPlaybackPlaysAndSessionsRefuseAMember has a Member read Den's
// Playback report, plays and Now playing.
func TestPlaybackPlaysAndSessionsRefuseAMember(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		sam := w.w.Member(t, "sam")
		for _, route := range []string{"/playback", "/plays", "/sessions"} {
			if status, body := w.o.CallAs(t, http.MethodGet, servers+"/"+w.id+route, "", sam.Session); status != http.StatusForbidden {
				t.Errorf("a Member's GET %s = %d %s, want 403", route, status, body)
			}
		}
	})
}
