package plexkind_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

const every = 10 * time.Second

type watched struct {
	mu    sync.Mutex
	looks []link.Seen
}

func (d *den) watch() *watched {
	ctx, cancel := context.WithCancel(d.t.Context())
	out := &watched{}
	done := make(chan error)
	go func() {
		done <- d.conn.Watch(ctx, every, func(s link.Seen) {
			out.mu.Lock()
			defer out.mu.Unlock()
			out.looks = append(out.looks, s)
		})
	}()
	d.t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			d.t.Errorf("Watch = %v, want nil once its context ends", err)
		}
	})
	synctest.Wait()
	return out
}

// polls is the sessions of each poll so far, as sentences.
func (w *watched) polls() [][]string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out [][]string
	for _, s := range w.looks {
		if !s.Polled {
			continue
		}
		var ss []string
		for _, x := range s.Sessions {
			ss = append(ss, fmt.Sprintf("%s for %s on %s, %s %q hardware %v", x.Path, x.User, x.Player, x.Decision, x.Reasons, x.Hardware))
		}
		out = append(out, ss)
	}
	return out
}

func (w *watched) transcoding() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.looks[len(w.looks)-1].Transcoding
}

func (d *den) play(path link.ServerPath, decision link.Decision, unreported bool) string {
	d.t.Helper()
	raw, err := json.Marshal(d.f.Cue(d.t, "plex-play", map[string]any{
		"path": path.String(), "user": "jackson", "player": "Living room", "decision": decision,
		"hardware": decision == link.DecisionTranscode, "unreported": unreported,
	}))
	var played struct{ Session string }
	if err == nil {
		err = json.Unmarshal(raw, &played)
	}
	if err != nil {
		d.t.Fatal(err)
	}
	synctest.Wait()
	return played.Session
}

// TestWatchSeesEachDecision plays Metropolis once per decision: the
// playing notification polls at once, the poll lists the session with its
// decision and hardware and no reasons, which Plex never gives, and the
// poll after a stop lists nothing.
func TestWatchSeesEachDecision(t *testing.T) {
	for _, dec := range link.Decisions() {
		t.Run(string(dec), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const film = "Films/Metropolis (1927)/Metropolis (1927).mkv"
				d := newDen(t, film)
				w := d.watch()
				session := d.play(d.at(film), dec, false)
				want := fmt.Sprintf(`%s for jackson on Living room, %s [] hardware %v`, d.at(film), dec, dec == link.DecisionTranscode)
				if got := w.polls(); len(got) != 2 || !slices.Equal(got[1], []string{want}) {
					t.Errorf("before the next tick the polls saw %q, want a second poll with %q", got, want)
				}
				if w.transcoding() != (dec == link.DecisionTranscode) {
					t.Errorf("transcoding = %v, want %v", w.transcoding(), dec == link.DecisionTranscode)
				}
				d.f.Cue(t, "plex-stop", map[string]string{"session": session})
				time.Sleep(every)
				synctest.Wait()
				if got := w.polls(); len(got[len(got)-1]) != 0 {
					t.Errorf("the poll after the stop saw %q, want no session", got[len(got)-1])
				}
			})
		})
	}
}

// TestWatchCountsATranscodeNoSessionLists plays Metropolis on a client
// that never reports its timeline: no poll lists the session, and the
// server still transcodes.
func TestWatchCountsATranscodeNoSessionLists(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const film = "Films/Metropolis (1927)/Metropolis (1927).mkv"
		d := newDen(t, film)
		w := d.watch()
		d.play(d.at(film), link.DecisionTranscode, true)
		time.Sleep(every)
		synctest.Wait()
		if got := w.polls(); len(got[len(got)-1]) != 0 || !w.transcoding() {
			t.Errorf("the poll saw %q transcoding %v, want no session and transcoding", got[len(got)-1], w.transcoding())
		}
	})
}
