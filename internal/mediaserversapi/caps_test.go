package mediaserversapi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestTheControllerCapsWhatAPlayReports starts one play on Den per row,
// each report bent one way, and reads the history back: long values are
// cut, impossible ones refused or stored as unreported, times are held to
// now plus a minute and to the window, and a report from a Node that
// doesn't reach Den is dropped.
func TestTheControllerCapsWhatAPlayReports(t *testing.T) {
	reasons := make([]string, 70)
	for i := range reasons {
		reasons[i] = fmt.Sprintf("Reason%d", i)
	}
	kept, err := json.Marshal(reasons[:64])
	if err != nil {
		t.Fatal(err)
	}
	const by = ` by ada on Living room`
	for _, tc := range []struct {
		name string
		bend func(s *msv1.Session, now time.Time)
		// from is the reporting Node, seedbox when nil.
		from    func(w watched) role.NodeID
		end     time.Duration
		refused bool
		// says is the refusal's words, where the design gives them.
		says string
		want []string
	}{
		{
			name: "a long player name is cut at a rune boundary",
			bend: func(s *msv1.Session, _ time.Time) { s.Player = strings.Repeat("é", 300) },
			want: []string{clip + ` direct_play (direct direct none) [] by ada on ` + strings.Repeat("é", 200) + `, open`},
		},
		{
			name:    "a long path is refused",
			bend:    func(s *msv1.Session, _ time.Time) { s.ServerPath = "/" + strings.Repeat("a", 4096) },
			refused: true, says: "the play's server path is longer than 4096 bytes",
		},
		{
			name:    "a negative height is refused",
			bend:    func(s *msv1.Session, _ time.Time) { s.Received = &msv1.Received{Container: "TS", Height: -1} },
			refused: true,
		},
		{
			name: "a height of 9,000 is unreported",
			bend: func(s *msv1.Session, _ time.Time) { s.Received = &msv1.Received{Container: "TS", Height: 9000} },
			want: []string{clip + ` direct_play (direct direct none) []` + by + ` received {Container:TS VideoCodec: Height:0 Bitrate:0 AudioCodec: Channels:0}, open`},
		},
		{
			name: "70 reasons keep the first 64",
			bend: func(s *msv1.Session, _ time.Time) { s.Reasons = reasons },
			want: []string{clip + ` direct_play (direct direct none) ` + string(kept) + by + `, open`},
		},
		{
			name:    "an empty stream decision is refused",
			bend:    func(s *msv1.Session, _ time.Time) { s.Streams.Audio = "" },
			refused: true,
		},
		{
			name: "a start in the future reads as now",
			bend: func(s *msv1.Session, now time.Time) {
				s.StartedUnixMs, s.SeenUnixMs = now.Add(time.Hour).UnixMilli(), now.Add(time.Hour).UnixMilli()
			},
			end:  30 * time.Second,
			want: []string{clip + ` direct_play (direct direct none) []` + by + `, ended 30s in`},
		},
		{
			name: "a start before the window reads as its start",
			bend: func(s *msv1.Session, now time.Time) {
				s.StartedUnixMs, s.SeenUnixMs = now.Add(-time.Hour).UnixMilli(), now.Add(-time.Hour).UnixMilli()
			},
			end:  20 * time.Second,
			want: []string{clip + ` direct_play (direct direct none) []` + by + `, ended 20s in`},
		},
		{
			name: "a report from another Node is dropped",
			bend: func(*msv1.Session, time.Time) {},
			from: func(w watched) role.NodeID { return w.w.Home },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newWatched(t)
				server := serverOf(t, w.id)
				now := time.Now()
				s := session(now)
				tc.bend(s, now)
				from := w.w.Seedbox
				if tc.from != nil {
					from = tc.from(w)
				}
				err := w.w.Apply(t.Context(), from, &msv1.Report{ServerId: server.Bytes(), Report: &msv1.Report_PlayStarted{PlayStarted: &msv1.PlayStarted{Session: s}}})
				switch {
				case tc.refused && (err == nil || tc.says != "" && err.Error() != tc.says):
					t.Fatalf("the start answered %v, want a refusal %q", err, tc.says)
				case !tc.refused && err != nil:
					t.Fatal(err)
				}
				if tc.end != 0 {
					w.w.EndPlay(t, w.w.Seedbox, server, playID(t, s), now.Add(tc.end))
				}
				expectPlays(t, w.plays(t), tc.want...)
			})
		})
	}
}

func serverOf(t *testing.T, id string) link.ServerID {
	t.Helper()
	u, err := storage.ParseUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	return link.ServerID{UUID: u}
}

func session(now time.Time) *msv1.Session {
	return &msv1.Session{
		PlayId: storage.NewUUID().Bytes(), SessionKey: "9", ServerPath: clip, User: "ada", Player: "Living room",
		StartedUnixMs: now.UnixMilli(), SeenUnixMs: now.UnixMilli(),
		Streams: &msv1.Streams{Video: "direct", Audio: "direct", Subtitles: "none"},
	}
}

func playID(t *testing.T, s *msv1.Session) link.PlayID {
	t.Helper()
	u, err := storage.UUIDFromBytes(s.GetPlayId())
	if err != nil {
		t.Fatal(err)
	}
	return link.PlayID{UUID: u}
}
