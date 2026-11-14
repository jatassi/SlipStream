package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	clip  = shelves + "/Clip.mkv"
	every = 15 * time.Second
)

// watched is Den with Clip on its Shelves, saved with seedbox reaching it,
// and seedbox's holder watching it.
type watched struct {
	rig
	den  faketest.Served
	id   string
	stop func()
}

func newWatched(t *testing.T) watched {
	t.Helper()
	r := newRig(t)
	srv := den(floorVer)
	srv.Libraries[0].Scanned = true
	w := watched{rig: r, den: r.w.Jellyfin(t, host, srv, clip)}
	w.id = r.add(t)
	w.stop = r.w.Watch(t, r.w.Seedbox)
	return w
}

// play starts Clip on Living room for ada, transcoding for a reason, and
// answers its session key.
func (w watched) play(t *testing.T) string {
	t.Helper()
	raw, err := json.Marshal(w.den.Cue(t, "jellyfin-play", map[string]any{
		"path": clip, "user": "ada", "player": "Living room", "decision": link.DecisionTranscode, "reasons": []string{"ContainerNotSupported"},
	}))
	var played struct{ Session string }
	if err == nil {
		err = json.Unmarshal(raw, &played)
	}
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return played.Session
}

func (w watched) plays(t *testing.T) []string {
	t.Helper()
	u, err := storage.ParseUUID(w.id)
	if err != nil {
		t.Fatal(err)
	}
	return w.w.Plays(t, link.ServerID{UUID: u})
}

func wait(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

func expectPlays(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("the play history is %q, want %q", got, want)
	}
}

const transcoding = `/media/Shelves/Clip.mkv transcode ["ContainerNotSupported"] by ada on Living room, `

// TestSessionsShowAPlayUntilItVanishesFromTwoPolls plays Clip on Den: the
// Admin sees it playing at once and hears of it, and once it stops it
// stays in the history, ended at the last poll that saw it.
func TestSessionsShowAPlayUntilItVanishesFromTwoPolls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		w.o.Expect(t, http.MethodGet, servers+"/"+w.id+"/sessions", "", http.StatusOK, `{"reachable":true,"sessions":[]}`)
		session := w.play(t)
		w.o.Expect(t, http.MethodGet, servers+"/"+w.id+"/sessions", "", http.StatusOK,
			`{"reachable":true,"sessions":[{"playId":"id1","user":"ada","player":"Living room","product":"Jellyfin Server Web",`+
				`"path":"/media/Shelves/Clip.mkv","decision":"transcode","hardware":false,"reasons":["ContainerNotSupported"],"startedAt":"T"}]}`)
		if got := w.w.SessionsChanged(); len(got) == 0 || !slices.Equal(got[len(got)-1].ServerIDs, []string{w.id}) {
			t.Errorf("media-servers.sessions-changed published %v, want Den last", got)
		}
		wait(every)
		w.den.Cue(t, "jellyfin-stop", map[string]string{"session": session})
		wait(every)
		expectPlays(t, w.plays(t), transcoding+"open")
		wait(every)
		expectPlays(t, w.plays(t), transcoding+"ended 15s in")
		w.o.Expect(t, http.MethodGet, servers+"/"+w.id+"/sessions", "", http.StatusOK, `{"reachable":true,"sessions":[]}`)
	})
}

// TestNamingAnotherReachingNodeEndsThePlays moves Den to home while Clip
// plays: the play seedbox saw ends at its last look.
func TestNamingAnotherReachingNodeEndsThePlays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		w.play(t)
		wait(every)
		if status, body := w.o.Call(t, http.MethodPatch, servers+"/"+w.id, `{"reachingNodeId":"`+w.w.Home.String()+`"}`); status != http.StatusOK {
			t.Fatalf("name home = %d %s", status, body)
		}
		synctest.Wait()
		expectPlays(t, w.plays(t), transcoding+"ended 15s in")
		w.o.Expect(t, http.MethodGet, servers+"/"+w.id+"/sessions", "", http.StatusOK, `{"reachable":false,"sessions":[]}`)
	})
}

// TestALinkDownPastMarkUnhealthyAfterEndsThePlays stops seedbox while Clip
// plays: the play stays open for Mark unhealthy after, 5 minutes, then
// ends at seedbox's last look.
func TestALinkDownPastMarkUnhealthyAfterEndsThePlays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		w.play(t)
		wait(every)
		w.stop()
		w.w.Away(w.w.Seedbox)
		wait(5*time.Minute - time.Second)
		expectPlays(t, w.plays(t), transcoding+"open")
		wait(time.Second)
		expectPlays(t, w.plays(t), transcoding+"ended 15s in")
	})
}

// TestAHolderStartedAgainEndsThePlaysItLeftOpen stops seedbox's holder
// with no last report while Clip plays, then starts it again: its first
// poll's Watching ends the play it left open, Clip plays on as a new play,
// and late reports for the ended play change nothing.
func TestAHolderStartedAgainEndsThePlaysItLeftOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		w.play(t)
		_, body := w.o.Call(t, http.MethodGet, servers+"/"+w.id+"/sessions", "")
		wait(every)
		w.stop()
		wait(every)
		w.w.Watch(t, w.w.Seedbox)
		expectPlays(t, w.plays(t), transcoding+"ended 15s in", transcoding+"open")

		old, err := storage.ParseUUID(ownertest.FirstID(body))
		if err != nil {
			t.Fatal(err)
		}
		server, err := storage.ParseUUID(w.id)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Now().UnixMilli()
		for _, r := range []*msv1.Report{
			{Report: &msv1.Report_PlayChanged{PlayChanged: &msv1.PlayChanged{PlayId: old.Bytes(), Decision: "direct_play", SeenUnixMs: at}}},
			{Report: &msv1.Report_PlayEnded{PlayEnded: &msv1.PlayEnded{PlayId: old.Bytes(), LastSeenUnixMs: at}}},
			{Report: &msv1.Report_PlayStarted{PlayStarted: &msv1.PlayStarted{Session: &msv1.Session{
				PlayId: old.Bytes(), SessionKey: "1", ServerPath: clip, User: "ada", Player: "Living room", Decision: "direct_play",
				StartedUnixMs: at, SeenUnixMs: at,
			}}}},
		} {
			r.ServerId = server.Bytes()
			if err := w.w.Apply(t.Context(), w.w.Seedbox, r); err != nil {
				t.Fatal(err)
			}
		}
		expectPlays(t, w.plays(t), transcoding+"ended 15s in", transcoding+"open")
	})
}
