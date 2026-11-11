package grabs

import (
	"bytes"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Trial is the names the Custom Format tester reads, and how.
type Trial struct {
	Names []string
	// Work is the Work every name is judged against, nil to match each
	// name to the Works of every Content Type.
	Work *works.WorkID
	// Draft is a format matched in place of the saved one of its ID, or
	// beside the saved ones when none has its ID.
	Draft *formats.Format
	// File reads each name as a file path.
	File bool
}

// Tried is one name as the tester reads and judges it.
type Tried struct {
	Name string
	Read release.Read
	// Sentence is what the name reads as, the Works it matches and the
	// Targets it would fill, or why it would fill none.
	Sentence string
	// Work is the one Work it is judged against: the given Work, or the
	// only Work it matches. Nil when it matches several Works or none.
	Work *works.Summary
	// Works are the Works it matches: the given Work when the name names
	// it, or each Work whose title it gives.
	Works []works.Summary
	// Covers are the Items it names of the one Work it is judged against.
	Covers  []works.ItemID
	Formats []formats.Explained
	Targets []TriedTarget
}

// TriedTarget is a name's verdict at one Target. Reasons are the name's
// own refusals, then the reasons of the first cell it would fill, else of
// the first cell it was judged at.
type TriedTarget struct {
	Target    targets.Target
	Score     int
	WouldFill bool
	Reasons   []Worded
}

// Worded is a reason with its words.
type Worded struct {
	Reason
	Words string
}

// Try reads each name and judges it as a search would. With one Work,
// given or the only one a name matches, it judges that Work's Targets at
// the cells of the Items the name covers, against their held files. With
// several or none it judges every Target with no held file. A name with
// no Work given is read with each Content Type's Names in turn, and reads
// as the first type that finds a Work for it, else as the first type.
func (gr *Grabs) Try(ctx context.Context, by identity.Admin, t Trial) ([]Tried, error) {
	if t.Draft != nil {
		if err := formats.Check(*t.Draft, formats.Timings{Match: gr.cfg.Targets.MatchLimit()}); err != nil {
			return nil, err
		}
	}
	types, err := gr.cfg.Works.Types(ctx)
	if err != nil {
		return nil, err
	}
	every, err := gr.cfg.Targets.ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	tr := trier{gr: gr, file: t.File, work: t.Work, every: every}
	for _, typ := range types {
		if typ.Enabled {
			tr.types = append(tr.types, typ.ID)
		}
	}
	out := make([]Tried, len(t.Names))
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if tr.cuts, err = gr.cfg.Targets.CutsIn(ctx, r); err != nil {
			return err
		}
		if tr.catalog, err = gr.cfg.Targets.FormatsIn(ctx, r); err != nil {
			return err
		}
		if t.Draft != nil {
			tr.catalog = tr.catalog.With(*t.Draft)
		}
		for i, name := range t.Names {
			if out[i], err = tr.try(ctx, r, name); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

type trier struct {
	gr      *Grabs
	file    bool
	work    *works.WorkID
	types   []works.TypeID
	every   []targets.Target
	cuts    targets.Cuts
	catalog *formats.Catalog
}

func (tr trier) try(ctx context.Context, r storage.Reader[file], name string) (Tried, error) {
	if tr.work != nil {
		w, err := works.SummaryOf(ctx, r, *tr.work)
		if err != nil {
			return Tried{}, err
		}
		return tr.againstWork(ctx, r, name, tr.read(name, w.ContentType), w)
	}
	var rd release.Read
	var found []works.Summary
	for i, typ := range tr.types {
		read := tr.read(name, typ)
		ws, err := tr.matching(ctx, r, typ, read)
		if err != nil {
			return Tried{}, err
		}
		if i == 0 || len(found) == 0 && len(ws) > 0 {
			rd = read
		}
		found = append(found, ws...)
	}
	if len(found) == 1 {
		return tr.againstWork(ctx, r, name, rd, found[0])
	}
	return tr.againstEvery(name, rd, found), nil
}

func (tr trier) read(name string, typ works.TypeID) release.Read {
	names, cuts := tr.gr.cfg.Names[typ], tr.cuts.For(typ)
	if tr.file {
		return release.File(name, names, cuts)
	}
	return release.Title(name, names, cuts)
}

// matching is the Works of typ whose titles hold each of rd's titles and
// that the title matching of a search matches rd to.
func (tr trier) matching(ctx context.Context, r storage.Reader[file], typ works.TypeID, rd release.Read) ([]works.Summary, error) {
	var seen []works.WorkID
	var out []works.Summary
	for _, title := range rd.Name.Titles {
		ids, err := tr.gr.cfg.Works.MatchIn(ctx, r, title, &typ)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if slices.Contains(seen, id) {
				continue
			}
			seen = append(seen, id)
			expect, err := tr.gr.cfg.Terms[typ].Expect(ctx, r, id)
			if err != nil {
				return nil, err
			}
			if m, _ := release.Match(rd.Name, rd.Facts.IDs, expect.Claim, tr.gr.cfg.Names[typ]); m != release.Matched {
				continue
			}
			w, err := works.SummaryOf(ctx, r, id)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		}
	}
	return out, nil
}

// againstWork judges name, read as rd, against w as a search for any of
// its Items would: its own refusals, then every counted, released cell of
// the Items it covers.
func (tr trier) againstWork(ctx context.Context, r storage.Reader[file], name string, rd release.Read, w works.Summary) (Tried, error) {
	expect, err := tr.gr.cfg.Terms[w.ContentType].Expect(ctx, r, w.ID)
	if err != nil {
		return Tried{}, err
	}
	j, err := tr.gr.judgingOf(ctx, r, w.ContentType, Search{WorkID: w.ID}, expect)
	if err != nil {
		return Tried{}, err
	}
	j.scoring = tr.catalog
	a := assessRead(j, listed{Release: indexers.Release{Release: searching.Release{Title: name}}}, rd)
	goals, err := targets.GoalsIn(ctx, r, w.ID)
	if err != nil {
		return Tried{}, err
	}
	all, err := tr.gr.cfg.Files.Cells(ctx, r, w.ID)
	if err != nil {
		return Tried{}, err
	}
	var c cells
	for _, cell := range all {
		if cell.Item.Counts() && cell.Released && slices.Contains(a.covers, cell.Item.ID) {
			c.open = append(c.open, cell)
		}
	}
	judged := judge(c, &a)
	t := Tried{Name: name, Read: a.read, Work: &w, Covers: a.covers, Formats: tr.catalog.Explain(a.read)}
	if a.namesTheWork() {
		t.Works = []works.Summary{w}
	}
	s := scoredOf(a.read, t.Formats)
	for _, target := range goals.Targets {
		tt := TriedTarget{Target: target, Score: target.Score(&s)}
		reasons := slices.Clone(a.refusals)
		var at []verdict
		for _, v := range judged.verdicts {
			if v.target == target.ID {
				at = append(at, v)
			}
		}
		if i := slices.IndexFunc(at, verdict.grabbable); i >= 0 {
			tt.WouldFill = len(a.refusals) == 0
			reasons = append(reasons, at[i].reasons()...)
		} else if len(at) > 0 {
			reasons = append(reasons, at[0].reasons()...)
		}
		tt.Reasons = worded(reasons, &target, a.read.Facts)
		t.Targets = append(t.Targets, tt)
	}
	t.Sentence = sentence(t, worded(a.refusals, nil, a.read.Facts))
	return t, nil
}

// againstEvery judges rd at every Target with no held file, for a name
// that matches several Works or none.
func (tr trier) againstEvery(name string, rd release.Read, ws []works.Summary) Tried {
	t := Tried{Name: name, Read: rd, Works: ws, Formats: tr.catalog.Explain(rd)}
	s := scoredOf(rd, t.Formats)
	for _, target := range tr.every {
		v := verdict{target: target.ID, wanted: true, v: targets.Judge(target, s, nil)}
		t.Targets = append(t.Targets, TriedTarget{Target: target, Score: v.v.Score(), WouldFill: v.grabbable(), Reasons: worded(v.reasons(), &target, rd.Facts)})
	}
	t.Sentence = acceptance(t)
	return t
}

func scoredOf(rd release.Read, es []formats.Explained) targets.Scored {
	var matched formats.Set
	for _, e := range es {
		if e.Matched {
			matched = append(matched, e.Format)
		}
	}
	slices.SortFunc(matched, func(a, b formats.ID) int { return bytes.Compare(a[:], b[:]) })
	return targets.Scored{Facts: rd.Facts, Formats: matched}
}

// sentence is t, judged against one Work, in words: "Reads as
// WEBDL-1080p from FLUX. It matches Heat (1995) and would fill 1080p." A
// name that would fill no Target ends with the first reason a Target
// gives, in Profile order. A name that doesn't name the Work says why
// with the first of own, its own refusals.
func sentence(t Tried, own []Worded) string {
	if len(t.Works) == 0 {
		return readWords(t) + ". " + own[0].Words + " It would fill no Target."
	}
	read := readWords(t) + ". " + matchWords(t)
	fills := targetNames(t)
	if len(fills) > 0 {
		return read + " and would fill " + joinWords(fills) + "."
	}
	return read + " and would fill no Target. " + whyNot(t)
}

// acceptance is t, judged at every Target with no held file because it
// matches several Works or none, in words: "Reads as WEBDL-1080p from
// FLUX. It matches no Work. 1080p and HD accept it." With no Work there
// is no cell to fill, so it says which Targets accept the name.
func acceptance(t Tried) string {
	read := readWords(t) + ". " + matchWords(t) + "."
	if len(t.Targets) == 0 {
		return read + " No Target is set up. Add a Target in Settings."
	}
	switch accept := targetNames(t); len(accept) {
	case 0:
		return read + " No Target accepts it. " + whyNot(t)
	case 1:
		return read + " " + accept[0] + " accepts it."
	default:
		return read + " " + joinWords(accept) + " accept it."
	}
}

// readWords is what t reads as: "Reads as WEBDL-1080p from FLUX".
func readWords(t Tried) string {
	f := t.Read.Facts
	read := "Reads no Quality"
	if q, ok := f.Quality.Get(); ok {
		read = "Reads as " + q.Info().Name
	}
	if g, ok := f.Group.Get(); ok {
		read += " from " + g
	}
	return read
}

// matchWords is the Works t matches: "It matches Heat (1995)".
func matchWords(t Tried) string {
	if len(t.Works) == 0 {
		return "It matches no Work"
	}
	labels := make([]string, len(t.Works))
	for i, w := range t.Works {
		labels[i] = w.Label
	}
	return "It matches " + joinWords(labels)
}

// targetNames is the names of the Targets t would fill.
func targetNames(t Tried) []string {
	var names []string
	for _, tt := range t.Targets {
		if tt.WouldFill {
			names = append(names, tt.Target.Name)
		}
	}
	return names
}

// whyNot is the first reason a Target of t gives that isn't a note, in
// words.
func whyNot(t Tried) string {
	if len(t.Targets) == 0 {
		return "No Target is set up. Add a Target in Settings."
	}
	for _, tt := range t.Targets {
		for _, r := range tt.Reasons {
			if r.Code != targets.SizeNotChecked {
				return r.Words
			}
		}
	}
	return "No Item it names is monitored and released."
}

var attributeWords = map[targets.Attribute]string{
	targets.HDR: "HDR", targets.VideoCodec: "video codec", targets.AudioCodec: "audio codec",
	targets.Channels: "channels", targets.Language: "language",
}

// worded is rs, reasons at t about a name whose facts are f, each with
// its words. t is nil once the Target is deleted.
func worded(rs []Reason, t *targets.Target, f release.Facts) []Worded {
	out := make([]Worded, len(rs))
	for i, r := range rs {
		out[i] = Worded{Reason: r, Words: reasonWords(r, t, f)}
	}
	return out
}

const deletedTarget = "a deleted Target"

// reasonWords is r, a reason at t about a name whose facts are f, as a
// sentence. t is nil once the Target is deleted.
func reasonWords(r Reason, t *targets.Target, f release.Facts) string {
	at := deletedTarget
	if t != nil {
		at = t.Name
	}
	switch r.Code {
	case targets.WrongWork:
		return "An id in the name belongs to another Work."
	case targets.WrongItem:
		if r.Value == "" {
			return "It names none of the Work's Items."
		}
		return "The Work has no Item " + r.Value + "."
	case targets.CoversMore:
		return "It holds more than one Item."
	case targets.WrongTitle:
		if r.Value == "" {
			return "Its name gives no title."
		}
		return "Its name gives " + r.Value + ", which isn't this Work."
	case targets.CutNotWanted:
		return "The Work doesn't want the " + r.Value + " cut at " + at + "."
	case targets.QualityNotInTarget:
		what := "a release with no Quality"
		if q, ok := f.Quality.Get(); ok {
			what = q.Info().Name
		}
		if t == nil {
			return "A deleted Target doesn't accept " + what + "."
		}
		return at + " doesn't accept " + what + "."
	case targets.FilterFailed:
		return "It fails the " + attributeWords[r.Attribute] + " filter of " + at + "."
	case targets.SizeBelow:
		return "It's smaller than " + at + " allows."
	case targets.SizeAbove:
		return "It's larger than " + at + " allows."
	case targets.SizeNotChecked:
		return "Its size isn't checked at " + at + " because the name gives no size or the runtime is unknown."
	case targets.ScoreBelowMinimum:
		if t == nil {
			return "Score " + r.Value + " is below the minimum of " + deletedTarget + "."
		}
		return "Score " + r.Value + " is below the minimum of " + strconv.Itoa(t.Scores.Minimum) + " at " + at + "."
	case targets.NotAnUpgrade:
		return "It isn't an upgrade of the file at " + at + "."
	case targets.RepackOtherGroup:
		from := "It's a repack with no release group"
		if g, ok := f.Group.Get(); ok {
			from = "It's a repack from " + g
		}
		if r.Value == "" {
			return from + ", and the file at " + at + " has no release group."
		}
		return from + ", and the file at " + at + " is from " + r.Value + "."
	case targets.Blocklisted:
		return "It's on the Blocklist."
	}
	panic("grabs: no words for the reason " + string(r.Code))
}

// joinWords is ws as a list in a sentence: "A", "A and B", "A, B and C".
func joinWords(ws []string) string {
	if len(ws) == 1 {
		return ws[0]
	}
	return strings.Join(ws[:len(ws)-1], ", ") + " and " + ws[len(ws)-1]
}
