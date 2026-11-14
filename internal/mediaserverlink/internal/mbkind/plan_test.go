package mbkind_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/mbkind"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

var t0 = time.Date(2026, 11, 14, 14, 2, 0, 0, time.UTC)

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// row is a due row in "kind to", "kind from>to" or "library" form, at
// step, reached at stepAt minutes before t0.
type row struct {
	name   string
	change string
	root   int
	step   link.Step
	stepAt int
}

func TestPlan(t *testing.T) {
	movies, fresh := roots.ID{UUID: storage.NewUUID()}, roots.ID{UUID: storage.NewUUID()}
	rootsOf := []roots.ID{movies, fresh}
	maps := []link.PlanMapping{
		{Root: movies, Library: "lib-movies", Folder: path(t, "/media/movies"), Shown: true, Confirmed: true, Scanned: true},
		{Root: fresh, Library: "lib-fresh", Folder: path(t, "/media/fresh"), Shown: true, Confirmed: true},
	}
	cases := []struct {
		name string
		rows []row
		// want is each Job as "work step [rows]".
		want []string
	}{
		{
			name: "several paths in one folder are one Refresh",
			rows: []row{{"a", "added Heat/Heat.mkv", 0, link.StepNone, 0}, {"b", "removed Heat/Heat.srt", 0, link.StepNone, 0}, {"c", "replaced Alien/Alien.mkv", 0, link.StepNone, 0}},
			want: []string{"refresh none [a b c]"},
		},
		{
			name: "rows into a fresh mapping wait behind one Fresh",
			rows: []row{{"a", "added Heat/Heat.mkv", 1, link.StepNone, 0}, {"b", "added Alien/Alien.mkv", 1, link.StepNone, 0}, {"c", "added Up/Up.mkv", 0, link.StepNone, 0}},
			want: []string{"fresh lib-fresh", "refresh none [c]"},
		},
		{
			name: "library rows are one Rescan ahead of the refreshes",
			rows: []row{{"a", "added Heat/Heat.mkv", 0, link.StepNone, 0}, {"b", "library", 0, link.StepNone, 0}},
			want: []string{"rescan [lib-movies] [b]", "refresh none [a]"},
		},
		{
			name: "carries go after the refreshes, as one group",
			rows: []row{{"a", "moved Old/S01E01.mkv>New/S01E01.mkv", 0, link.StepNone, 0}, {"b", "added Heat/Heat.mkv", 0, link.StepNone, 0}, {"c", "moved Old/S01E02.mkv>New/S01E02.mkv", 0, link.StepNone, 0}},
			want: []string{"refresh none [b]", "carry none [a c]"},
		},
		{
			name: "rows that reached a step at different times are different Jobs",
			rows: []row{{"a", "moved A/A.mkv>B/B.mkv", 0, link.StepOldRemoved, 3}, {"b", "moved C/C.mkv>D/D.mkv", 0, link.StepOldRemoved, 3}, {"c", "moved E/E.mkv>F/F.mkv", 0, link.StepSnapshotted, 2}},
			want: []string{"carry old_removed [a b]", "carry snapshotted [c]"},
		},
		{
			name: "a refresh at its settle step goes again as its own Job",
			rows: []row{{"a", "added Heat/Heat.mkv", 0, link.StepReported, 2}, {"b", "added Up/Up.mkv", 0, link.StepNone, 0}},
			want: []string{"refresh reported [a]", "refresh none [b]"},
		},
		{
			name: "a change on a carry's new path waits for the carry",
			rows: []row{{"a", "moved A/A.mkv>B/B.mkv", 0, link.StepOldRemoved, 1}, {"b", "removed B/B.mkv", 0, link.StepNone, 0}},
			want: []string{"carry old_removed [a]"},
		},
	}
	for _, kind := range []link.Kind{mbkind.Jellyfin, mbkind.Emby} {
		for _, tc := range cases {
			t.Run(string(kind.About().Name)+"/"+tc.name, func(t *testing.T) {
				d, names := due(t, maps, rootsOf, tc.rows)
				var got []string
				for _, j := range kind.Plan(d) {
					got = append(got, describe(j, names))
				}
				if fmt.Sprint(got) != fmt.Sprint(tc.want) {
					t.Errorf("Plan = %q, want %q", got, tc.want)
				}
			})
		}
	}
}

func due(t *testing.T, maps []link.PlanMapping, rootsOf []roots.ID, rows []row) (d link.Due, names map[link.ChangeID]string) {
	t.Helper()
	d = link.Due{Now: t0, Mappings: maps, Wait: 10 * time.Second, Timings: link.DefaultTimings()}
	names = map[link.ChangeID]string{}
	for _, r := range rows {
		m := maps[r.root]
		lr := link.Row{
			JobChange: link.JobChange{ID: link.ChangeID{UUID: storage.NewUUID()}, Library: m.Library},
			Root:      rootsOf[r.root], Created: t0.Add(-time.Hour), Step: r.step,
		}
		if r.step != link.StepNone {
			lr.StepAt = t0.Add(-time.Duration(r.stepAt) * time.Minute)
		}
		kind, paths, _ := strings.Cut(r.change, " ")
		lr.Kind = link.ChangeKind(kind)
		if lr.Kind != link.ChangeLibrary {
			from, to, carry := strings.Cut(paths, ">")
			if !carry {
				from, to = "", from
			}
			lr.Rel, lr.Path = rel(t, to), m.Folder.Join(rel(t, to))
			if carry {
				lr.FromRel, lr.From = rel(t, from), m.Folder.Join(rel(t, from))
			}
		}
		names[lr.ID] = r.name
		d.Rows = append(d.Rows, lr)
	}
	return d, names
}

func describe(j link.Job, names map[link.ChangeID]string) string {
	of := func(cs []link.JobChange) []string {
		out := make([]string, len(cs))
		for i, c := range cs {
			out[i] = names[c.ID]
		}
		return out
	}
	switch w := j.Work.(type) {
	case link.Refresh:
		return fmt.Sprintf("refresh %s %v", j.Step, of(w.Changes))
	case link.Carry:
		return fmt.Sprintf("carry %s %v", j.Step, of(w.Changes))
	case link.Fresh:
		return "fresh " + w.Library
	case link.Rescan:
		ids := make([]string, len(w.Changes))
		for i, id := range w.Changes {
			ids[i] = names[id]
		}
		return fmt.Sprintf("rescan %v %v", w.Libraries, ids)
	}
	return fmt.Sprintf("%T", j.Work)
}
