package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/live"
)

// TestNoMemberSecretLeaves runs each Member path that carries a secret
// with a sentinel in place of it, and reads every response, event payload
// and log line, at debug, of the Controller, its Nodes and the API: none
// holds the sentinel.
func TestNoMemberSecretLeaves(t *testing.T) {
	const sentinel = "sentinel-7f3a9c"
	for _, c := range []struct {
		name string
		// run answers the response bodies the path got.
		run func(t *testing.T, d denRig) []string
	}{
		{"a password made through seedbox", func(t *testing.T, d denRig) []string {
			_, body := d.makeAs(t, d.sam, d.row, "sam", sentinel)
			return []string{body}
		}},
		{"a password Den refused with the name", func(t *testing.T, d denRig) []string {
			_, body := d.makeAs(t, d.sam, d.row, "ada", sentinel)
			return []string{body}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				d := newDenRigOn(t, newRig(t), "http://192.168.1.99:8096")
				tap := d.w.Tap(t.Context())
				bodies := c.run(t, d)
				synctest.Wait()
				bodies = append(bodies, d.expectOK(t, servers+"/access"))
				_, mine := d.o.CallAs(t, http.MethodGet, "/api/v1/account/media-servers", "", d.sam.Session)
				bodies = append(bodies, mine)
				expectNoLeak(t, sentinel, d.rig, tap, bodies)
			})
		})
	}
}

// TestNoMemberPlexTokenLeaves signs Sam in with Plex while the Plex
// server waits on the token for its share, and reads the responses, event
// payloads and log lines: none holds the token.
func TestNoMemberPlexTokenLeaves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newMemberRig(t)
		if status, body := r.offer(t, r.sam.ID, r.server, false); status != http.StatusOK {
			t.Fatalf("offer = %d %s", status, body)
		}
		tap := r.w.Tap(t.Context())
		pin := r.startMemberPin(t, r.sam)
		r.plex.ApproveAs(t, pin.AuthURL, "sam")
		time.Sleep(time.Second)
		synctest.Wait()
		_, state := r.o.CallAs(t, http.MethodGet, mine+"/plex/pin/"+jsonInt(pin.ID), "", r.sam.Session)
		_, rows := r.o.CallAs(t, http.MethodGet, mine, "", r.sam.Session)
		token := r.w.MemberPinToken(t, r.sam.ID, pin.ID)
		if _, held := r.w.PlexTVUser(t, token); !held {
			t.Fatal("plex.tv no longer holds the token the share waits on")
		}
		expectNoLeak(t, token.Reveal(), r.rig, tap, []string{state, rows, r.expectOK(t, servers+"/access")})
	})
}

// expectNoLeak reads bodies, every event tap took, and every log line, at
// debug, of the Controller, its Nodes and the API: none holds secret.
func expectNoLeak(t *testing.T, secret string, r rig, tap *live.Tap, bodies []string) {
	t.Helper()
	msgs, _ := tap.Take()
	for _, m := range msgs {
		payload, err := json.Marshal(m.Data)
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, m.Name+" "+string(payload))
	}
	for _, where := range append(bodies, r.w.Logged(), r.o.Log.String()) {
		if strings.Contains(where, secret) {
			t.Errorf("the secret leaked in %s", where)
		}
	}
}
