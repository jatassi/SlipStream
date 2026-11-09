package scanning_test

import (
	"errors"
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

func rel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var (
	monday  = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	tuesday = monday.Add(24 * time.Hour)
	heat    = works.WorkID{UUID: storage.NewUUID()}
	brazil  = works.WorkID{UUID: storage.NewUUID()}
)

func id(file string) storing.FileID { return storing.FileID{Volume: "803", File: file} }

// file is a walked file: path, size, its id ("" for a volume without
// ids) and when it was modified.
type file struct {
	path string
	size int64
	id   string
	at   time.Time
}

func (f file) walked(t *testing.T) storing.Walked {
	t.Helper()
	w := storing.Walked{Path: rel(t, f.path), Size: f.size, Seen: storing.Seen{Modified: f.at}}
	if f.id != "" {
		w.Seen.ID = id(f.id)
	}
	return w
}

// row is a recorded file. seen is false for a row no scan or import has
// stamped yet.
type row struct {
	file
	seen  bool
	sits  string
	next  string
	gone  bool
	stale bool
	work  works.WorkID
}

func (r row) recorded(t *testing.T) scanning.Recorded {
	t.Helper()
	rec := scanning.Recorded{ID: storage.NewUUID(), Work: r.work, Path: rel(t, r.path), Sits: rel(t, r.path), Size: r.size, Gone: r.gone, Stale: r.stale}
	if rec.Work.IsZero() {
		rec.Work = heat
	}
	if r.seen {
		w := r.walked(t)
		rec.Seen = &w.Seen
	}
	if r.sits != "" {
		rec.Sits = rel(t, r.sits)
	}
	if r.next != "" {
		rec.Next = rel(t, r.next)
	}
	return rec
}

func seen(path string, size int64, fileID string) row {
	return row{file: file{path, size, fileID, monday}, seen: true}
}

func listing(t *testing.T, caseFold bool, files []file, unread ...string) scanning.Listing {
	t.Helper()
	var entries []storing.Walked
	for _, f := range files {
		entries = append(entries, f.walked(t))
	}
	var u []storing.Unread
	for _, p := range unread {
		u = append(u, storing.Unread{Path: p, Reason: storing.UnreadDenied})
	}
	half := len(entries) / 2
	l, err := scanning.Finish([]storing.Page{
		{Entries: entries[:half], CaseFold: caseFold},
		{Entries: entries[half:], Unread: u, Done: true, CaseFold: caseFold},
	})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// outcome is a plan as words, each list sorted: a recorded file by its
// path, a move as "from > to", and a gone verdict as its files or
// "stopped N of M".
func outcome(p scanning.Plan) map[string][]string {
	out := map[string][]string{}
	put := func(list, s string) { out[list] = append(out[list], s) }
	reprobes := map[scanning.Reprobe]string{scanning.Keep: "", scanning.Touched: " touched", scanning.Stale: " stale", scanning.Resized: " resized"}
	returned := func(b bool) string {
		if b {
			return " returned"
		}
		return ""
	}
	for _, s := range p.Same {
		put("same", s.Rec.Path.String()+reprobes[s.Reprobe]+returned(s.Returned))
	}
	for _, s := range p.Changed {
		put("changed", s.Rec.Path.String()+reprobes[s.Reprobe]+returned(s.Returned))
	}
	for _, m := range p.Moved {
		put("moved", m.Rec.Path.String()+" > "+m.To.Path.String()+returned(m.Returned))
	}
	switch g := p.Gone.(type) {
	case scanning.Marked:
		for _, r := range g.Files {
			put("gone", r.Path.String())
		}
	case scanning.Stopped:
		put("stopped", strconv.Itoa(g.Gone)+" of "+strconv.Itoa(g.Of))
	}
	for _, r := range p.Held {
		put("held", r.Path.String())
	}
	for _, w := range p.Found {
		put("found", w.Path.String())
	}
	for _, w := range p.Listed {
		put("listed", w.Path.String())
	}
	for _, v := range p.Vanished {
		put("vanished", v.String())
	}
	for _, list := range out {
		slices.Sort(list)
	}
	return out
}

func TestCompare(t *testing.T) {
	const (
		nfc = "Amélie (2001)/Amélie.mkv"
		nfd = "Amélie (2001)/Amélie.mkv"
	)
	quarter := scanning.Guard{Share: 25}
	tests := []struct {
		name     string
		recorded []row
		walked   []file
		caseFold bool
		unread   []string
		known    func(t *testing.T) scanning.Known
		guard    scanning.Guard
		want     map[string][]string
	}{
		{
			name:     "a file at its path with its size and stamps is the same",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1")},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "1", monday}},
			want:     map[string][]string{"same": {"Heat (1995)/Heat.mkv"}},
		},
		{
			name:     "a new modification time reads the file again",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1")},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "1", tuesday}},
			want:     map[string][]string{"same": {"Heat (1995)/Heat.mkv touched"}},
		},
		{
			name:     "a media record older than the parser reads the file again",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "1", monday}, seen: true, stale: true}},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "1", monday}},
			want:     map[string][]string{"same": {"Heat (1995)/Heat.mkv stale"}},
		},
		{
			name:     "a row no scan has stamped is the same at any time",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "", time.Time{}}}},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "1", tuesday}},
			want:     map[string][]string{"same": {"Heat (1995)/Heat.mkv"}},
		},
		{
			name:     "another size at its path is changed",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1")},
			walked:   []file{{"Heat (1995)/Heat.mkv", 12, "2", tuesday}},
			want:     map[string][]string{"changed": {"Heat (1995)/Heat.mkv resized"}},
		},
		{
			name:     "a rename keeps the file's id and size, so it moved",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1")},
			walked:   []file{{"Heat (1995)/Heat (1995) Bluray-1080p.mkv", 10, "1", monday}},
			want:     map[string][]string{"moved": {"Heat (1995)/Heat.mkv > Heat (1995)/Heat (1995) Bluray-1080p.mkv"}},
		},
		{
			name:     "the same id at another size is a different file",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1"), seen("Brazil (1985)/Brazil.mkv", 20, "3")},
			walked:   []file{{"Heat (1995)/Heat 2160p.mkv", 11, "1", monday}, {"Brazil (1985)/Brazil.mkv", 20, "3", monday}},
			guard:    scanning.Guard{Share: 100},
			want: map[string][]string{
				"same": {"Brazil (1985)/Brazil.mkv"}, "gone": {"Heat (1995)/Heat.mkv"}, "found": {"Heat (1995)/Heat 2160p.mkv"},
			},
		},
		{
			name:     "on a volume without ids a move keeps its size and base name",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "")},
			walked:   []file{{"Old/Heat.mkv", 10, "", monday}},
			want:     map[string][]string{"moved": {"Heat (1995)/Heat.mkv > Old/Heat.mkv"}},
		},
		{
			name:     "on a volume without ids two files of one size and name are no move",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, ""), seen("Brazil (1985)/Brazil.mkv", 20, "")},
			walked:   []file{{"A/Heat.mkv", 10, "", monday}, {"B/Heat.mkv", 10, "", monday}, {"Brazil (1985)/Brazil.mkv", 20, "", monday}},
			guard:    scanning.Guard{Share: 100},
			want: map[string][]string{
				"same": {"Brazil (1985)/Brazil.mkv"}, "gone": {"Heat (1995)/Heat.mkv"}, "found": {"A/Heat.mkv", "B/Heat.mkv"},
			},
		},
		{
			name: "on a volume without ids two records of one size and name are no move",
			recorded: []row{
				seen("Heat (1995)/Heat.mkv", 10, ""), seen("Heat (1995) 4K/Heat.mkv", 10, ""), seen("Brazil (1985)/Brazil.mkv", 20, ""),
			},
			walked: []file{{"A/Heat.mkv", 10, "", monday}, {"Brazil (1985)/Brazil.mkv", 20, "", monday}},
			guard:  scanning.Guard{Share: 100},
			want: map[string][]string{
				"same": {"Brazil (1985)/Brazil.mkv"}, "gone": {"Heat (1995) 4K/Heat.mkv", "Heat (1995)/Heat.mkv"}, "found": {"A/Heat.mkv"},
			},
		},
		{
			name:     "two ids that differ are two files, whatever their size and name",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1"), seen("Brazil (1985)/Brazil.mkv", 20, "3")},
			walked:   []file{{"Copy/Heat.mkv", 10, "2", monday}, {"Brazil (1985)/Brazil.mkv", 20, "3", monday}},
			guard:    scanning.Guard{Share: 100},
			want: map[string][]string{
				"same": {"Brazil (1985)/Brazil.mkv"}, "gone": {"Heat (1995)/Heat.mkv"}, "found": {"Copy/Heat.mkv"},
			},
		},
		{
			name:     "a gone file back at its path with its size returns",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "1", monday}, seen: true, gone: true}},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "1", monday}},
			want:     map[string][]string{"same": {"Heat (1995)/Heat.mkv returned"}},
		},
		{
			name:     "a gone file found elsewhere by its id returns as a move",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "1", monday}, seen: true, gone: true}},
			walked:   []file{{"Back/Heat.mkv", 10, "1", monday}},
			want:     map[string][]string{"moved": {"Heat (1995)/Heat.mkv > Back/Heat.mkv returned"}},
		},
		{
			name:     "a gone file still absent is not gone again",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "1", monday}, seen: true, gone: true}, seen("Brazil (1985)/Brazil.mkv", 20, "3")},
			walked:   []file{{"Brazil (1985)/Brazil.mkv", 20, "3", monday}},
			want:     map[string][]string{"same": {"Brazil (1985)/Brazil.mkv"}},
		},
		{
			name:     "an NFD spelling of a recorded NFC path is the file, respelled",
			recorded: []row{seen(nfc, 10, "")},
			walked:   []file{{nfd, 10, "", monday}},
			want:     map[string][]string{"moved": {nfc + " > " + nfd}},
		},
		{
			name:     "on a case-insensitive root a case-only respelling at another size is changed",
			recorded: []row{seen("Heat (1995)/heat.mkv", 10, "")},
			walked:   []file{{"HEAT (1995)/Heat.mkv", 12, "", monday}},
			caseFold: true,
			want:     map[string][]string{"changed": {"Heat (1995)/heat.mkv resized"}},
		},
		{
			name:     "on a case-sensitive root a case-only respelling at another size is gone and found",
			recorded: []row{seen("Heat (1995)/heat.mkv", 10, ""), seen("Brazil (1985)/Brazil.mkv", 20, "")},
			walked:   []file{{"HEAT (1995)/Heat.mkv", 12, "", monday}, {"Brazil (1985)/Brazil.mkv", 20, "", monday}},
			guard:    scanning.Guard{Share: 100},
			want: map[string][]string{
				"same": {"Brazil (1985)/Brazil.mkv"}, "gone": {"Heat (1995)/heat.mkv"}, "found": {"HEAT (1995)/Heat.mkv"},
			},
		},
		{
			name:     "a file mid-move is the same where its bytes sit or go next",
			recorded: []row{{file: file{"Heat (1995)/Heat.mkv", 10, "1", monday}, seen: true, sits: "Old/Heat.mkv"}, {file: file{"Brazil (1985)/Brazil.mkv", 20, "3", monday}, seen: true, sits: "Old/Brazil.mkv", next: ".slipstream-incoming/b"}},
			walked:   []file{{"Old/Heat.mkv", 10, "1", monday}, {".slipstream-incoming/b", 20, "3", monday}},
			want:     map[string][]string{"same": {"Brazil (1985)/Brazil.mkv", "Heat (1995)/Heat.mkv"}},
		},
		{
			name:     "a held path is neither found nor a move target",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "")},
			walked:   []file{{"Heat (1995)/Heat.mkv", 10, "", monday}, {"Doomed/Old.mkv", 30, "", monday}},
			known: func(t *testing.T) scanning.Known {
				return scanning.Known{Held: []storing.Rel{rel(t, "Doomed/Old.mkv")}}
			},
			want: map[string][]string{"same": {"Heat (1995)/Heat.mkv"}},
		},
		{
			name:     "a file under a folder the walk couldn't read is held, not gone",
			recorded: []row{seen("Heat (1995)/Heat.mkv", 10, "1"), seen("Brazil (1985)/Brazil.mkv", 20, "3")},
			walked:   []file{{"Brazil (1985)/Brazil.mkv", 20, "3", monday}},
			unread:   []string{"Heat (1995)"},
			want:     map[string][]string{"same": {"Brazil (1985)/Brazil.mkv"}, "held": {"Heat (1995)/Heat.mkv"}},
		},
		{
			name: "a listed file unchanged is listed, changed is found, and missing has vanished",
			walked: []file{
				{"Halloween/Halloween.mkv", 10, "", monday}, {"Old Listing/Old.mkv", 20, "", tuesday},
				{"Grown/Grown.mkv", 31, "", monday},
			},
			known: func(t *testing.T) scanning.Known {
				return scanning.Known{Listed: []storing.Walked{
					file{"Halloween/Halloween.mkv", 10, "", monday}.walked(t), file{"Old Listing/Old.mkv", 20, "", time.Time{}}.walked(t),
					file{"Grown/Grown.mkv", 30, "", monday}.walked(t), file{"Away/Away.mkv", 5, "", monday}.walked(t),
				}}
			},
			want: map[string][]string{
				"listed": {"Halloween/Halloween.mkv", "Old Listing/Old.mkv"}, "found": {"Grown/Grown.mkv"}, "vanished": {"Away/Away.mkv"},
			},
		},
		{
			name:     "on a case-insensitive root a listing compares ignoring case",
			walked:   []file{{"HALLOWEEN/Halloween.mkv", 10, "", monday}},
			caseFold: true,
			known: func(t *testing.T) scanning.Known {
				return scanning.Known{Listed: []storing.Walked{file{"Halloween/halloween.mkv", 10, "", monday}.walked(t)}}
			},
			want: map[string][]string{"listed": {"HALLOWEEN/Halloween.mkv"}},
		},
		{
			name:     "a quarter of the root gone is marked at a stop share of 25%",
			recorded: fourFolders(),
			walked:   []file{{"B/b.mkv", 2, "", monday}, {"C/c.mkv", 3, "", monday}, {"D/d.mkv", 4, "", monday}},
			guard:    quarter,
			want:     map[string][]string{"same": {"B/b.mkv", "C/c.mkv", "D/d.mkv"}, "gone": {"A/a.mkv"}},
		},
		{
			name:     "more than a quarter gone stops at 25%, keeping every other transition",
			recorded: fourFolders(),
			walked:   []file{{"C/c.mkv", 3, "", monday}, {"D/d.mkv", 4, "", monday}, {"New/new.mkv", 9, "", monday}},
			guard:    quarter,
			want:     map[string][]string{"same": {"C/c.mkv", "D/d.mkv"}, "found": {"New/new.mkv"}, "stopped": {"2 of 4"}},
		},
		{
			name:     "a stop share of 100% never stops while a Work folder is there",
			recorded: fourFolders(),
			walked:   []file{{"D/d.mkv", 4, "", monday}},
			guard:    scanning.Guard{Share: 100},
			want:     map[string][]string{"same": {"D/d.mkv"}, "gone": {"A/a.mkv", "B/b.mkv", "C/c.mkv"}},
		},
		{
			name:     "a confirmed count marks up to that many gone",
			recorded: fourFolders(),
			walked:   []file{{"C/c.mkv", 3, "", monday}, {"D/d.mkv", 4, "", monday}},
			guard:    scanning.Guard{Share: 25, Confirmed: 2},
			want:     map[string][]string{"same": {"C/c.mkv", "D/d.mkv"}, "gone": {"A/a.mkv", "B/b.mkv"}},
		},
		{
			name:     "more gone than confirmed stops again with the new count",
			recorded: fourFolders(),
			walked:   []file{{"D/d.mkv", 4, "", monday}},
			guard:    scanning.Guard{Share: 25, Confirmed: 2},
			want:     map[string][]string{"same": {"D/d.mkv"}, "stopped": {"3 of 4"}},
		},
		{
			name:     "every Work folder absent stops at any stop share",
			recorded: fourFolders(),
			walked:   []file{{"Loose.mkv", 9, "", monday}},
			guard:    scanning.Guard{Share: 100},
			want:     map[string][]string{"found": {"Loose.mkv"}, "stopped": {"4 of 4"}},
		},
		{
			name:     "files loose at the top name no Work folder",
			recorded: []row{seen("Heat.mkv", 1, ""), seen("Brazil.mkv", 2, "")},
			walked:   []file{{"Brazil.mkv", 2, "", monday}},
			guard:    scanning.Guard{Share: 100},
			want:     map[string][]string{"same": {"Brazil.mkv"}, "gone": {"Heat.mkv"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recorded []scanning.Recorded
			for _, r := range tt.recorded {
				recorded = append(recorded, r.recorded(t))
			}
			var k scanning.Known
			if tt.known != nil {
				k = tt.known(t)
			}
			guard := tt.guard
			if guard == (scanning.Guard{}) {
				guard = quarter
			}
			got := outcome(scanning.Compare(listing(t, tt.caseFold, tt.walked, tt.unread...), recorded, k, guard))
			if !maps.EqualFunc(got, tt.want, slices.Equal) {
				t.Fatalf("plan = %v, want %v", got, tt.want)
			}
		})
	}
}

// fourFolders is a root of four Works, one file each.
func fourFolders() []row {
	return []row{seen("A/a.mkv", 1, ""), seen("B/b.mkv", 2, ""), seen("C/c.mkv", 3, ""), seen("D/d.mkv", 4, "")}
}

func TestFinishRefusesAnUnfinishedWalk(t *testing.T) {
	pages := []storing.Page{{Entries: []storing.Walked{file{"Heat (1995)/Heat.mkv", 10, "", monday}.walked(t)}}}
	if _, err := scanning.Finish(pages); !errors.Is(err, scanning.ErrUnfinished) {
		t.Fatalf("Finish of a walk with no Done page = %v, want ErrUnfinished", err)
	}
	if _, err := scanning.Finish(nil); !errors.Is(err, scanning.ErrUnfinished) {
		t.Fatalf("Finish of no pages = %v, want ErrUnfinished", err)
	}
	l, err := scanning.Finish(append(pages, storing.Page{Done: true}))
	if err != nil || l.Len() != 1 {
		t.Fatalf("Finish of a done walk = %d files, %v; want 1 file", l.Len(), err)
	}
}

func TestByWorkWritesOnlyWhatChangedAndWhatVerifyConfirmed(t *testing.T) {
	recorded := []scanning.Recorded{
		seen("Heat (1995)/Heat.mkv", 10, "1").recorded(t),
		{ID: storage.NewUUID(), Work: heat, Path: rel(t, "Heat (1995)/Heat 4K.mkv"), Sits: rel(t, "Heat (1995)/Heat 4K.mkv"), Size: 40},
		row{file: file{"Brazil (1985)/Brazil.mkv", 20, "3", monday}, seen: true, work: brazil}.recorded(t),
		row{file: file{"Brazil (1985)/Brazil Extended.mkv", 30, "4", monday}, seen: true, work: brazil}.recorded(t),
		row{file: file{"Brazil (1985)/Brazil Final.mkv", 35, "5", monday}, seen: true, work: brazil}.recorded(t),
	}
	l := listing(t, false, []file{
		{"Heat (1995)/Heat.mkv", 10, "1", monday}, {"Heat (1995)/Heat 4K.mkv", 40, "2", monday},
		{"Brazil (1985)/Brazil (1985).mkv", 20, "3", tuesday},
	})
	p := scanning.Compare(l, recorded, scanning.Known{}, scanning.Guard{Share: 100})
	marked, ok := p.Gone.(scanning.Marked)
	if !ok || len(marked.Files) != 2 {
		t.Fatalf("gone = %+v, want two marked files", p.Gone)
	}
	shares := p.ByWork(marked.Files[:1])

	h := shares[heat]
	if len(h.Same) != 1 || h.Same[0].Rec.Path.String() != "Heat (1995)/Heat 4K.mkv" {
		t.Fatalf("Heat's share = %+v, want only the file no scan had stamped", h)
	}
	if got := h.Reprobes(); len(got) != 0 {
		t.Fatalf("Heat reprobes %v, want none", got)
	}
	b := shares[brazil]
	if len(b.Moved) != 1 || len(b.Gone) != 1 || b.Gone[0].ID != marked.Files[0].ID {
		t.Fatalf("Brazil's share = %+v, want its move and only the confirmed gone file", b)
	}
	if len(shares) != 2 {
		t.Fatalf("shares for %d Works, want 2", len(shares))
	}

	stopped := scanning.Compare(l, recorded, scanning.Known{}, scanning.Guard{Share: 25})
	if _, ok := stopped.Gone.(scanning.Stopped); !ok {
		t.Fatalf("gone at 25%% = %+v, want stopped", stopped.Gone)
	}
	if got := stopped.ByWork(marked.Files); len(got[brazil].Gone) != 0 {
		t.Fatalf("a stopped plan marks %v gone, want none", got[brazil].Gone)
	}
}

func TestReprobesAreChangedTouchedAndStaleFiles(t *testing.T) {
	recorded := []scanning.Recorded{
		seen("A/a.mkv", 1, "").recorded(t), seen("B/b.mkv", 2, "").recorded(t),
		row{file: file{"C/c.mkv", 3, "", monday}, seen: true, stale: true}.recorded(t), seen("D/d.mkv", 4, "").recorded(t),
	}
	l := listing(t, false, []file{{"A/a.mkv", 9, "", monday}, {"B/b.mkv", 2, "", tuesday}, {"C/c.mkv", 3, "", monday}, {"D/d.mkv", 4, "", monday}})
	var got []string
	for _, w := range scanning.Compare(l, recorded, scanning.Known{}, scanning.Guard{Share: 25}).ByWork(nil)[heat].Reprobes() {
		got = append(got, w.Path.String())
	}
	slices.Sort(got)
	if want := []string{"A/a.mkv", "B/b.mkv", "C/c.mkv"}; !slices.Equal(got, want) {
		t.Fatalf("reprobes = %v, want %v", got, want)
	}
}
