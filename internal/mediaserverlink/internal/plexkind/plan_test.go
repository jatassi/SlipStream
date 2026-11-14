package plexkind_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/plexkind"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

var t0 = time.Date(2026, 11, 14, 14, 2, 0, 0, time.UTC)

type plan struct {
	t      *testing.T
	shows  roots.ID
	movies roots.ID
	due    link.Due
	names  map[link.ChangeID]string
}

func newPlan(t *testing.T) *plan {
	p := &plan{t: t, shows: roots.ID{UUID: storage.NewUUID()}, movies: roots.ID{UUID: storage.NewUUID()}, names: map[link.ChangeID]string{}}
	p.due = link.Due{
		Now: t0, Timings: link.DefaultTimings(),
		Mappings: []link.PlanMapping{
			{Root: p.shows, Library: "2", Folder: mustPath(t, `D:\Plex\Shows`), Shown: true, Confirmed: true, Scanned: true},
			{Root: p.movies, Library: "1", Folder: mustPath(t, `D:\Plex\Movies`), Shown: true, Confirmed: true, Scanned: true},
		},
	}
	return p
}

func mustPath(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustRel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (p *plan) mapping(root roots.ID) link.PlanMapping {
	for _, m := range p.due.Mappings {
		if m.Root == root {
			return m
		}
	}
	p.t.Fatalf("no mapping for root %s", root)
	return link.PlanMapping{}
}

// row adds a due row named name: kind at to, or a carry from -> to when
// from isn't empty.
func (p *plan) row(name string, root roots.ID, kind link.ChangeKind, from, to string) *link.Row {
	m := p.mapping(root)
	r := link.Row{
		JobChange: link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Kind: kind, Library: m.Library},
		Root:      root, Created: t0.Add(-time.Minute), NextAt: t0.Add(-time.Minute), Step: link.StepNone,
	}
	if kind != link.ChangeLibrary {
		r.Rel = mustRel(p.t, to)
		r.Path = m.Folder.Join(r.Rel)
	}
	if from != "" {
		r.FromRel = mustRel(p.t, from)
		r.From = m.Folder.Join(r.FromRel)
	}
	p.names[r.ID] = name
	p.due.Rows = append(p.due.Rows, r)
	return &p.due.Rows[len(p.due.Rows)-1]
}

// jobs is each Job of Plan as a line: its work, step and changes.
func (p *plan) jobs() []string {
	var out []string
	for _, j := range plexkind.Kind.Plan(p.due) {
		switch w := j.Work.(type) {
		case link.Fresh:
			out = append(out, "fresh "+w.Folder.String())
		case link.Rescan:
			out = append(out, fmt.Sprintf("rescan %v %v", w.Libraries, p.named(w.Changes)))
		case link.Refresh:
			out = append(out, fmt.Sprintf("refresh %v", p.named(changeIDs(w.Changes))))
		case link.Carry:
			out = append(out, fmt.Sprintf("carry at %s since %s %v", j.Step, j.At.Format(time.TimeOnly), p.named(changeIDs(w.Changes))))
		default:
			p.t.Fatalf("Plan answered %T", j.Work)
		}
	}
	return out
}

func (p *plan) named(ids []link.ChangeID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = p.names[id]
	}
	return out
}

func changeIDs(cs []link.JobChange) []link.ChangeID {
	out := make([]link.ChangeID, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

func TestPlan(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows func(p *plan)
		want []string
	}{
		{
			name: "forty episodes of one season are one refresh",
			rows: func(p *plan) {
				for e := 1; e <= 40; e++ {
					p.row("e", p.shows, link.ChangeAdded, "", fmt.Sprintf("Show/Season 01/Show - S01E%02d.mkv", e))
				}
			},
			want: []string{"refresh [" + repeat("e", 40) + "]"},
		},
		{
			name: "ten folders of one section are ten folder refreshes in one job",
			rows: func(p *plan) {
				for s := 1; s <= 10; s++ {
					p.row(fmt.Sprintf("s%d", s), p.shows, link.ChangeAdded, "", fmt.Sprintf("Show/Season %02d/E01.mkv", s))
				}
			},
			want: []string{"refresh [s1 s2 s3 s4 s5 s6 s7 s8 s9 s10]"},
		},
		{
			name: "eleven folders of one section are one section refresh",
			rows: func(p *plan) {
				for s := 1; s <= 11; s++ {
					p.row(fmt.Sprintf("s%d", s), p.shows, link.ChangeAdded, "", fmt.Sprintf("Show/Season %02d/E01.mkv", s))
				}
				p.row("film", p.movies, link.ChangeAdded, "", "Heat (1995)/Heat (1995).mkv")
				p.row("subs", p.shows, link.ChangeSidecarAdded, "", "Show/Season 01/E01.en.srt")
			},
			want: []string{"rescan [2] [s1 s2 s3 s4 s5 s6 s7 s8 s9 s10 s11]", "refresh [film subs]"},
		},
		{
			name: "a section with an open carry is never refreshed whole",
			rows: func(p *plan) {
				p.row("moved", p.shows, link.ChangeMoved, "Other/S01E01.mkv", "Other/Season 01/S01E01.mkv")
				for s := 1; s <= 11; s++ {
					p.row(fmt.Sprintf("s%d", s), p.shows, link.ChangeAdded, "", fmt.Sprintf("Show/Season %02d/E01.mkv", s))
				}
				p.row("library", p.shows, link.ChangeLibrary, "", "")
			},
			want: []string{"refresh [s1 s2 s3 s4 s5 s6 s7 s8 s9 s10 s11]", "carry at none since 00:00:00 [moved]"},
		},
		{
			name: "a refresh of a carry's source folder waits until the carry is confirmed",
			rows: func(p *plan) {
				p.row("moved", p.movies, link.ChangeMoved, "Heat (1995)/Heat.mkv", "Heat (1995) {edition-Test}/Heat.mkv")
				p.row("beside", p.movies, link.ChangeRemoved, "", "Heat (1995)/Heat.nfo.mkv")
				p.row("elsewhere", p.movies, link.ChangeAdded, "", "Alien (1979)/Alien.mkv")
			},
			want: []string{"refresh [elsewhere]", "carry at none since 00:00:00 [moved]"},
		},
		{
			name: "a confirmed carry no longer holds its source folder",
			rows: func(p *plan) {
				r := p.row("moved", p.movies, link.ChangeMoved, "Heat (1995)/Heat.mkv", "Heat (1995) {edition-Test}/Heat.mkv")
				r.Step, r.StepAt = link.StepConfirmed, t0.Add(-time.Second)
				p.row("beside", p.movies, link.ChangeRemoved, "", "Heat (1995)/Heat.nfo.mkv")
			},
			want: []string{"refresh [beside]", "carry at confirmed since 14:01:59 [moved]"},
		},
		{
			name: "library changes are one rescan and carries go one job per step",
			rows: func(p *plan) {
				p.row("library", p.movies, link.ChangeLibrary, "", "")
				a := p.row("a", p.shows, link.ChangeMoved, "A/E01.mkv", "A/E01 renamed.mkv")
				a.Step, a.StepAt, a.Snapshot = link.StepSnapshotted, t0.Add(-30*time.Second), &link.Snapshot{}
				b := p.row("b", p.shows, link.ChangeMoved, "B/E01.mkv", "B/E01 renamed.mkv")
				b.Step, b.StepAt = link.StepScanned, t0.Add(-20*time.Second)
				c := p.row("c", p.shows, link.ChangeMoved, "C/E01.mkv", "C/E01 renamed.mkv")
				c.Step, c.StepAt, c.Snapshot = link.StepSnapshotted, t0.Add(-40*time.Second), &link.Snapshot{}
			},
			want: []string{
				"rescan [1] [library]",
				"carry at snapshotted since 14:01:20 [a c]",
				"carry at scanned since 14:01:40 [b]",
			},
		},
		{
			name: "a fresh mapping's Job goes first and holds its rows",
			rows: func(p *plan) {
				p.due.Mappings[1].Scanned = false
				p.row("film", p.movies, link.ChangeAdded, "", "Heat (1995)/Heat (1995).mkv")
				p.row("episode", p.shows, link.ChangeAdded, "", "Show/Season 01/E01.mkv")
			},
			want: []string{`fresh D:\Plex\Movies`, "refresh [episode]"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPlan(t)
			tc.rows(p)
			if got := p.jobs(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Plan =\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

func TestPlanWaitsOutWaitBeforeSending(t *testing.T) {
	p := newPlan(t)
	p.due.Wait = 2 * time.Minute
	p.row("film", p.movies, link.ChangeAdded, "", "Heat (1995)/Heat (1995).mkv")
	if got := p.jobs(); len(got) != 0 {
		t.Errorf("Plan before the wait passed = %q, want nothing", got)
	}
	p.due.Now = t0.Add(time.Minute)
	if got, want := p.jobs(), []string{"refresh [film]"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Plan after the wait passed = %q, want %q", got, want)
	}
}

func repeat(s string, n int) string { return strings.TrimSpace(strings.Repeat(s+" ", n)) }
