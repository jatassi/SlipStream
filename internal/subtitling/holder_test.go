package subtitling_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
	"github.com/jatassi/slipstream/internal/subtitling"
)

type roster struct {
	keys    map[role.NodeID]link.Key
	holders []role.NodeID
}

func (r roster) Key(node role.NodeID) (link.Key, bool) {
	k, ok := r.keys[node]
	return k, ok
}

func (r roster) Serves(n role.Name, node role.NodeID) bool {
	return n == role.Subtitles && slices.Contains(r.holders, node)
}

// trio is a Controller and two Workers over linktest. The first Worker
// holds the Subtitles Role and serves the stub provider; the Controller
// serves it too, as both halves do.
type trio struct {
	holder  subtitling.Remote
	stub    *stub
	other   *rpc.Dispatch
	roster  roster
	control role.NodeID
}

func newTrio(t *testing.T, tm subtitling.Timings) *trio {
	t.Helper()
	cm, am, bm := http.NewServeMux(), http.NewServeMux(), http.NewServeMux()
	p := linktest.NewTrio(t, linktest.Side{Lockstep: cm}, [2]linktest.Side{{Lockstep: am}, {Lockstep: bm}})
	ids := []role.NodeID{{UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}}
	holds := func(role.Name) bool { return true }
	controller := rpc.NewDispatch(p.Controller.Link, p.Controller.Key, p.Controller.Key, holds)
	worker := rpc.NewDispatch(p.Workers[0].Link, p.Workers[0].Key, p.Controller.Key, holds)
	other := rpc.NewDispatch(p.Workers[1].Link, p.Workers[1].Key, p.Controller.Key, holds)
	for mux, d := range map[*http.ServeMux]*rpc.Dispatch{cm: controller, am: worker, bm: other} {
		mux.Handle("/", d)
	}
	s := &stub{files: map[string]subtitling.Download{}}
	for _, d := range []*rpc.Dispatch{controller, worker} {
		subtitlesv1connect.SubtitlesService.Mount(d, subtitling.Serve([]subtitling.Provider{s}, tm))
	}
	r := roster{keys: map[role.NodeID]link.Key{ids[0]: p.Controller.Key, ids[1]: p.Workers[0].Key, ids[2]: p.Workers[1].Key}, holders: ids[:2]}
	p.Connect(t)
	return &trio{
		holder: subtitling.NewRemote(subtitlesv1connect.SubtitlesService.On(rpc.NewCluster(controller, r), ids[1])),
		stub:   s, other: other, roster: r, control: ids[0],
	}
}

const stubID calls.ID = "stub"

type stub struct {
	files     map[string]subtitling.Download
	downloads atomic.Int32
}

func (s *stub) Spec() subtitling.Spec {
	return subtitling.Spec{ID: stubID, Label: "Stub", Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}}
}

func (s *stub) CheckKey(context.Context, calls.Credentials) (subtitling.Account, error) {
	return subtitling.Account{Name: "someone", Plan: "Free"}, nil
}

func (s *stub) Search(context.Context, calls.Credentials, subtitling.Query) ([]subtitling.Offer, error) {
	return nil, nil
}

func (s *stub) Download(_ context.Context, _ calls.Credentials, o subtitling.Offer) (subtitling.Download, error) {
	s.downloads.Add(1)
	return s.files[o.Key], nil
}

// file is a two-minute stretch of speech, the map a Node built of it,
// and the cues a person timed to it.
type file struct {
	sched timingtest.Schedule
	cues  []timing.Span
	m     []byte
}

func newFile(t *testing.T, seed uint64) file {
	t.Helper()
	s := timingtest.NewSchedule(seed, 2*time.Minute, 0.4)
	var runs []speechtest.Run
	for _, r := range s.Heard(seed+1, timingtest.Typical).Runs {
		runs = append(runs, speechtest.Run{Start: r.Start, End: r.End})
	}
	m, err := speechtest.Map(runs, s.Length, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file{sched: s, cues: s.Cues(seed + 2), m: m.Encode()}
}

func gate() timing.Gate {
	return timing.Gate{Match: timing.DefaultMatchPercent / 100.0, MaxOffset: 10 * time.Second, Framerates: timing.StandardRatios}
}

func en(t *testing.T) language.Code {
	t.Helper()
	c, err := language.Parse("en")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f file) ask(t *testing.T, key string, subject subtitling.Subject) subtitling.FetchAsk {
	t.Helper()
	return subtitling.FetchAsk{
		Offer: subtitling.Offer{Provider: stubID, Key: key, Language: en(t)}, Key: "k",
		Want: subtitling.Query{Subject: subject, Language: en(t)}, Map: f.m, Gate: gate(), Constants: timing.DefaultConstants(),
	}
}

var film = subtitling.Subject{Kind: subtitling.Feature, IMDb: "tt0113277"}

func srt(spans []timing.Span) []byte {
	var b strings.Builder
	for i, s := range spans {
		ms := func(d time.Duration) string {
			v := max(d, 0).Milliseconds()
			return fmt.Sprintf("%02d:%02d:%02d,%03d", v/3600000, v/60000%60, v/1000%60, v%1000)
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\nLine %d\n\n", i+1, ms(s.Start), ms(s.End), i+1)
	}
	return []byte(b.String())
}

func ass(spans []timing.Span, ssa bool) []byte {
	cs := func(d time.Duration) string {
		v := max(d, 0).Milliseconds() / 10
		return fmt.Sprintf("%d:%02d:%02d.%02d", v/360000, v/6000%60, v/100%60, v%100)
	}
	var b strings.Builder
	if ssa {
		b.WriteString("[Script Info]\nScriptType: v4.00\n\n[V4 Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding\nStyle: Default,Arial,20,16777215,65535,65535,-2147483640,-1,0,1,1,2,2,30,30,10,0,0\n\n[Events]\nFormat: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
		for i, s := range spans {
			fmt.Fprintf(&b, "Dialogue: Marked=0,%s,%s,Default,,0000,0000,0000,,Line %d\n", cs(s.Start), cs(s.End), i+1)
		}
		return []byte(b.String())
	}
	b.WriteString("[Script Info]\nScriptType: v4.00+\n\n[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\nStyle: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1\n\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	for i, s := range spans {
		fmt.Fprintf(&b, "Dialogue: 0,%s,%s,Default,,0,0,0,,Line %d\n", cs(s.Start), cs(s.End), i+1)
	}
	b.WriteString("\n[Fonts]\nfontname: CustomFont_0.ttf\nM(\\'!!!!!!!!@!%!!!@!!!Q`!!$!-1T5\"0!!!!!X!!!!!!!!!!!!!!!!!!!!!!!!\n")
	return []byte(b.String())
}

func zipOf(t *testing.T, members ...cues.Entry) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, m := range members {
		w, err := z.Create(m.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(m.Bytes); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newTrio(t, subtitling.DefaultTimings())
		f := newFile(t, 11)
		late := timingtest.Shift(f.cues, 2500*time.Millisecond)
		c.stub.files["late"] = subtitling.Download{Name: "late.zip", Bytes: zipOf(t, cues.Entry{Name: "Heat.1995.srt", Bytes: srt(late)})}
		c.stub.files["image"] = subtitling.Download{Name: "image.sup", Bytes: []byte("PG\x00\x00\x00\x00\x00\x00\x00\x00\x16\x00\x00")}
		c.stub.files["pack"] = subtitling.Download{Name: "pack.zip", Bytes: zipOf(t, cues.Entry{Name: "01 Show.S01.srt", Bytes: srt(f.cues)}, cues.Entry{Name: "03 Show.S01.srt", Bytes: srt(f.cues)})}

		got, _, err := c.holder.Fetch(t.Context(), f.ask(t, "late", film))
		if err != nil || got.Verdict != subtitling.Passed || got.Key != "late" {
			t.Fatalf("Fetch of a subtitle 2.5 s late = %+v, %v, want passed", got.Verdict, err)
		}
		if o := got.Claimed.Offset(); (o + 2500*time.Millisecond).Abs() > 150*time.Millisecond {
			t.Errorf("claimed offset = %v, want about -2.5 s", o)
		}
		followsClaim(t, got, srt(late))

		if got, _, err := c.holder.Fetch(t.Context(), f.ask(t, "image", film)); err != nil || got.Verdict != subtitling.Image {
			t.Errorf("Fetch of an image subtitle = %v, %v, want image", got.Verdict, err)
		}
		second := subtitling.Subject{Kind: subtitling.Episode, IMDb: "tt0903747", Season: 1, Episodes: []int{2}}
		if got, _, err := c.holder.Fetch(t.Context(), f.ask(t, "pack", second)); err != nil || got.Verdict != subtitling.WrongEpisode {
			t.Errorf("Fetch of episode 2 from a pack of 1 and 3 = %v, %v, want wrong episode", got.Verdict, err)
		}
	})
}

// followsClaim checks a passed answer keeps the fetched bytes as its
// original and moves every cue of its retimed copy by the claimed timing,
// text untouched.
func followsClaim(t *testing.T, got subtitling.Fetched, fetched []byte) {
	t.Helper()
	original, err := cues.Parse(got.Original, en(t), 0, 1<<20)
	if err != nil || !bytes.Equal(got.Original, fetched) {
		t.Errorf("original = %v, want the bytes as fetched", err)
	}
	retimed, err := cues.Parse(got.Retimed, en(t), 0, 1<<20)
	if err != nil || !cues.SameText(original, retimed) {
		t.Fatalf("retimed = %v, want the same text", err)
	}
	seg := got.Claimed.Segments[0]
	for i, o := range original.Times() {
		want := time.Duration(seg.Ratio*float64(o.Start)) + seg.Offset
		if d := (retimed.Times()[i].Start - want).Abs(); d > 2*time.Millisecond {
			t.Fatalf("cue %d starts %v, want %v by the claimed timing %+v", i, retimed.Times()[i].Start, want, seg)
		}
	}
}

func TestPackMembersBounded(t *testing.T) {
	f := newFile(t, 12)
	one := srt(f.cues)
	members := []cues.Entry{{Name: "00 Show.S01 SEASON 1.srt", Bytes: srt(f.cues[:1])}}
	for e := 1; e <= 6; e++ {
		members = append(members, cues.Entry{Name: fmt.Sprintf("%02d Show.S01.srt", e), Bytes: one})
	}
	first := subtitling.Subject{Kind: subtitling.Episode, IMDb: "tt0903747", Season: 1, Episodes: []int{1}}
	for _, row := range []struct {
		name          string
		members, size int
		held, left    int
	}{
		{"by count", 3, 1 << 30, 3, 2},
		{"by bytes", 100, 2*len(one) + 1, 2, 3},
	} {
		t.Run(row.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tm := subtitling.DefaultTimings()
				tm.PackMembers, tm.PackBytes = row.members, row.size
				c := newTrio(t, tm)
				c.stub.files["pack"] = subtitling.Download{Name: "pack.zip", Bytes: zipOf(t, members...)}
				got, _, err := c.holder.Fetch(t.Context(), f.ask(t, "pack", first))
				if err != nil || got.Verdict != subtitling.Passed {
					t.Fatalf("Fetch = %v, %v", got.Verdict, err)
				}
				if len(got.Members) != row.held || got.MembersLeft != row.left {
					t.Errorf("held %d members with %d left, want %d with %d", len(got.Members), got.MembersLeft, row.held, row.left)
				}
				heldInOrder(t, got.Members, one)
			})
		})
	}
}

// heldInOrder checks members are episodes 2 onwards of season 1, each
// the original bytes.
func heldInOrder(t *testing.T, members []subtitling.Member, original []byte) {
	t.Helper()
	for i, m := range members {
		if m.Season != 1 || !slices.Equal(m.Episodes, []int{i + 2}) || !bytes.Equal(m.Bytes, original) {
			t.Errorf("member %d = %s season %d episodes %v, want episode %d of season 1 as its original", i, m.Name, m.Season, m.Episodes, i+2)
		}
	}
}

func TestSyncTimesBytes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newTrio(t, subtitling.DefaultTimings())
		f := newFile(t, 13)
		got, err := c.holder.Sync(t.Context(), subtitling.SyncAsk{Name: "upload.srt", Bytes: srt(timingtest.Shift(f.cues, -time.Second)), Language: en(t), Map: f.m, Gate: gate(), Constants: timing.DefaultConstants()})
		if err != nil || got.Verdict != subtitling.Passed || len(got.Retimed) == 0 {
			t.Errorf("Sync of a subtitle 1 s early = %v, %v, want passed with a retimed copy", got.Verdict, err)
		}
		if c.stub.downloads.Load() != 0 {
			t.Errorf("Sync downloaded %d times, want none", c.stub.downloads.Load())
		}
	})
}

func TestWorkerRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newTrio(t, subtitling.DefaultTimings())
		forged := subtitlesv1connect.SubtitlesService.On(rpc.NewCluster(c.other, c.roster), c.control)
		_, err := forged.CheckKey(t.Context(), &subtitlesv1.CheckKeyRequest{Provider: string(stubID), Key: "k"})
		if rpc.CodeOf(err) != rpc.CodePermissionDenied {
			t.Errorf("a Worker's CheckKey on the Controller = %v (code %v), want permission denied", err, rpc.CodeOf(err))
		}
		if _, _, err := c.holder.CheckKey(t.Context(), stubID, "k"); err != nil {
			t.Errorf("the Controller's own CheckKey = %v, want it to run", err)
		}
	})
}

func TestFetchAnswerPassesJudge(t *testing.T) {
	f := newFile(t, 14)
	for _, row := range []struct {
		name  string
		bytes []byte
	}{
		{"srt", srt(timingtest.Shift(f.cues, 1700*time.Millisecond))},
		{"ass with fonts", ass(timingtest.Shift(f.cues, -1300*time.Millisecond), false)},
		{"ssa", ass(timingtest.Scale(f.cues, 25/23.976), true)},
	} {
		t.Run(row.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := newTrio(t, subtitling.DefaultTimings())
				c.stub.files["s"] = subtitling.Download{Name: "s." + row.name[:3], Bytes: row.bytes}
				got, _, err := c.holder.Fetch(t.Context(), f.ask(t, "s", film))
				if err != nil || got.Verdict != subtitling.Passed {
					t.Fatalf("Fetch = %v, %v, want passed", got.Verdict, err)
				}
				judged(t, f, got)
			})
		})
	}
}

// judged checks the Controller's parse of both byte sets passes Judge with
// the holder's claimed segments and the same constants.
func judged(t *testing.T, f file, got subtitling.Fetched) {
	t.Helper()
	original, err := cues.Parse(got.Original, en(t), 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	retimed, err := cues.Parse(got.Retimed, en(t), 0, 1<<20)
	if err != nil || retimed.Format() != got.Format || original.Format() != got.Format {
		t.Fatalf("retimed parses as %v, %v, want %v", retimed.Format(), err, got.Format)
	}
	var claimed []int
	for _, s := range got.Claimed.Segments {
		claimed = append(claimed, s.First)
	}
	fit, err := timing.Judge(f.sched.Reference(), subtitling.Spans(original), subtitling.Spans(retimed), claimed, got.Claimed.Prominence, gate(), timing.DefaultConstants(), retimed.Unit())
	if err != nil || fit.Verdict != timing.Passed {
		t.Errorf("Judge = %v, %v, want the holder's answer to pass", fit.Verdict, err)
	}
}
