package imports

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ImportedFacts is what an Imported or Upgraded message tells: the files
// an Import placed since it last came to rest, and how many of its files
// failed or wait for the Admin.
type ImportedFacts struct {
	Work    notice.Work  `json:"work"`
	Root    string       `json:"root"`
	Files   []PlacedFile `json:"files"`
	Failed  int          `json:"failed"`
	Waiting int          `json:"waiting"`
}

// PlacedFile is one placed file: the Items it holds, its quality and
// size, how it got into the library and the qualities it replaced.
type PlacedFile struct {
	Items    []ItemAt          `json:"items"`
	Quality  release.Quality   `json:"quality"`
	Size     int64             `json:"size"`
	PlacedBy PlacedBy          `json:"placedBy"`
	Replaced []release.Quality `json:"replaced,omitempty"`
}

// ItemAt is an Item by its place: its level's label, empty for a type
// with no levels, and its number in that level.
type ItemAt struct {
	Level  string `json:"level"`
	Number int    `json:"number"`
}

// ScanAddedFacts is what a Works added by a Library scan message tells.
type ScanAddedFacts struct {
	Root  string       `json:"root"`
	Nouns notice.Nouns `json:"nouns"`
	Count int          `json:"count"`
	First []string     `json:"first"`
}

var (
	importedRow = notice.Row[ImportedFacts]{
		Spec:   notice.Spec{Kind: "imported", Label: "Imported", Group: notice.Imports, OnByDefault: true, BellByDefault: true},
		Render: renderImported,
	}
	upgradedRow = notice.Row[ImportedFacts]{
		Spec:   notice.Spec{Kind: "upgraded", Label: "Upgraded", Group: notice.Imports, OnByDefault: true, BellByDefault: true},
		Render: renderUpgraded,
	}
	scanAddedRow = notice.Row[ScanAddedFacts]{
		Spec:   notice.Spec{Kind: "works_added_by_scan", Label: "Works added by a Library scan", Group: notice.Library, BellByDefault: true},
		Render: renderScanAdded,
	}
)

// Notices is the events imports records.
func Notices() []notice.Entry { return []notice.Entry{importedRow, upgradedRow, scanAddedRow} }

// announce records Imported and Upgraded for the files imp placed since it
// last came to rest. A root Import never sends: Pending imports' Add and
// in-place adoption both come through one. Neither does an Import that
// failed, or that its Work's delete ended.
func (im *Imports) announce(ctx context.Context, tx *storage.Tx[file], imp Import) error {
	if imp.Source == FromRoot || !imp.atRest() || imp.endedByDelete() {
		return nil
	}
	q := db.New(tx)
	untold, err := q.ListUntoldPlaced(ctx, imp.ID)
	if err != nil || len(untold) == 0 {
		return err
	}
	replaced, err := files.ImportedIn(ctx, tx, imp.Work, imp.ID.UUID)
	if err != nil {
		return err
	}
	tree, err := im.cfg.Works.TreeIn(ctx, tx, imp.Work)
	if err != nil {
		return err
	}
	w, err := notice.WorkIn(ctx, tx, im.cfg.Works, imp.Work)
	if err != nil {
		return err
	}
	fresh := make([]File, 0, len(untold))
	for _, pos := range untold {
		fresh = append(fresh, imp.Files[slices.IndexFunc(imp.Files, func(f File) bool { return int64(f.Position) == pos })])
	}
	words, err := im.cfg.Roots.WordsIn(ctx, tx)
	if err != nil {
		return err
	}
	in, up := splitPlaced(ImportedFacts{Work: w, Root: words[fresh[0].Dest.Root]}, fresh, replaced, numbersOf(tree), imp.Files)
	if in != nil {
		if err := importedRow.Record(ctx, tx, im.cfg.Notices, *in); err != nil {
			return err
		}
	}
	if up != nil {
		if err := upgradedRow.Record(ctx, tx, im.cfg.Notices, *up); err != nil {
			return err
		}
	}
	return q.MarkTold(ctx, db.MarkToldParams{ToldAt: storage.Now(), ImportID: imp.ID})
}

func (im Import) atRest() bool {
	s := im.Stage()
	return s == StageWaiting || s == StageDone
}

// endedByDelete is whether im's Work's delete ended it.
func (im Import) endedByDelete() bool {
	return slices.ContainsFunc(im.Files, func(f File) bool {
		return f.Outcome == Declined && f.Reason != nil && f.Reason.Code == WorkDeleted
	})
}

// numbersOf is each of tree's Items by its place.
func numbersOf(tree works.Tree) map[works.ItemID]ItemAt {
	labels := make(map[works.LevelID]string, len(tree.Levels))
	for _, l := range tree.Levels {
		labels[l.ID] = l.Label
	}
	place := map[works.LevelID]int{}
	out := make(map[works.ItemID]ItemAt, len(tree.Items))
	for _, it := range tree.Items {
		place[it.LevelID]++
		out[it.ID] = ItemAt{Level: labels[it.LevelID], Number: place[it.LevelID]}
	}
	return out
}

// splitPlaced is pure: fresh files that replaced nothing go to Imported,
// the rest to Upgraded. The counts of all's failed and waiting files go to
// the first of the two that has files.
func splitPlaced(base ImportedFacts, fresh []File, replaced map[files.ID][]files.Replaced, at map[works.ItemID]ItemAt, all []File) (imported, upgraded *ImportedFacts) {
	in, up := base, base
	for _, f := range fresh {
		p := PlacedFile{Quality: f.Quality, Size: f.Size, PlacedBy: f.PlacedBy}
		for _, it := range f.Items {
			p.Items = append(p.Items, at[it])
		}
		for _, r := range replaced[f.FileID] {
			p.Replaced = append(p.Replaced, r.Quality)
		}
		if len(p.Replaced) == 0 {
			in.Files = append(in.Files, p)
		} else {
			up.Files = append(up.Files, p)
		}
	}
	var failed, waiting int
	for _, f := range all {
		if f.Outcome == FailedOut {
			failed += len(f.Items)
		}
		if f.Outcome == Waiting {
			waiting += len(f.Items)
		}
	}
	if len(in.Files) > 0 {
		in.Failed, in.Waiting = failed, waiting
		imported = &in
	} else {
		up.Failed, up.Waiting = failed, waiting
	}
	if len(up.Files) > 0 {
		upgraded = &up
	}
	return imported, upgraded
}

// announceScan records Works added by a Library scan for root: the Works a
// scan adopted after since that live in root. The count is all of them;
// only the titles stop at three.
func (im *Imports) announceScan(ctx context.Context, tx *storage.Tx[file], root roots.Root, since storage.Time) error {
	added, err := works.AddedSince(ctx, tx, since)
	if err != nil || len(added) == 0 {
		return err
	}
	in, err := roots.Only(ctx, tx, root.ID, works.IDsOf(added))
	if err != nil || len(in) == 0 {
		return err
	}
	words, err := im.cfg.Roots.WordsIn(ctx, tx)
	if err != nil {
		return err
	}
	return scanAddedRow.Record(ctx, tx, im.cfg.Notices, ScanAddedFacts{
		Root: words[root.ID], Nouns: notice.NounsOf(im.cfg.Works, root.ContentType), Count: len(in), First: works.LabelsOf(added, in, 3),
	})
}

func renderImported(f ImportedFacts, _ notice.Context) notice.Message {
	m := placedMessage(f, notice.Good)
	if f.Work.Nouns.Single {
		p := f.Files[0]
		m.Title = "Imported " + f.Work.Label
		m.Lines = []string{factsWords(f.Files), "SlipStream " + placedWords[p.PlacedBy] + " it into the " + f.Root + "."}
		return m
	}
	item := lower(f.Work.Nouns.Item)
	m.Title = "Imported " + notice.Count(itemCount(f.Files), item) + " of " + f.Work.Label
	m.Lines = []string{capital(itemsWords(f.Files, item)) + ".", factsWords(f.Files)}
	return withLeft(m, f, item)
}

func renderUpgraded(f ImportedFacts, _ notice.Context) notice.Message {
	m := placedMessage(f, notice.Good)
	if f.Work.Nouns.Single {
		p := f.Files[0]
		m.Title = "Upgraded " + f.Work.Label + " to " + p.Quality.Info().Name
		m.Lines = []string{"It replaced " + qualitiesWords(p.Replaced) + "."}
		return m
	}
	item := lower(f.Work.Nouns.Item)
	n := itemCount(f.Files)
	m.Title = "Upgraded " + notice.Count(n, item) + " of " + f.Work.Label
	shown := f.Files[:min(len(f.Files), 3)]
	for _, p := range shown {
		m.Lines = append(m.Lines, capital(itemsWords([]PlacedFile{p}, item))+" went from "+qualitiesWords(p.Replaced)+" to "+p.Quality.Info().Name+".")
	}
	switch rest := n - itemCount(shown); {
	case rest == 1:
		m.Lines = append(m.Lines, "1 more "+item.One+" was upgraded.")
	case rest > 1:
		m.Lines = append(m.Lines, strconv.Itoa(rest)+" more "+item.Many+" were upgraded.")
	}
	return withLeft(m, f, item)
}

func renderScanAdded(f ScanAddedFacts, _ notice.Context) notice.Message {
	return notice.Message{
		Title: "A Library scan of the " + f.Root + " added " + notice.Count(f.Count, lower(f.Nouns.Work)),
		Lines: notice.Titles(f.First, f.Count, "added"),
		Link:  notice.Link{Path: "/library"},
		Tone:  notice.Info,
	}
}

func placedMessage(f ImportedFacts, tone notice.Tone) notice.Message {
	m := notice.Message{Link: notice.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: tone}
	m.Body.Work = &notice.BodyWork{ID: f.Work.ID.String(), Title: f.Work.Label, ContentType: string(f.Work.Type)}
	for _, p := range f.Files {
		b := notice.BodyFile{Quality: p.Quality.Info().Name, Replaced: qualitiesWords(p.Replaced), PlacedBy: string(p.PlacedBy)}
		for _, it := range p.Items {
			b.Items = append(b.Items, itemAtWords(it, lower(f.Work.Nouns.Item)))
		}
		m.Body.Files = append(m.Body.Files, b)
	}
	return m
}

// withLeft adds the sentences for the files that failed or wait.
func withLeft(m notice.Message, f ImportedFacts, item works.Noun) notice.Message {
	if f.Failed > 0 {
		verb := " weren't imported. "
		if f.Failed == 1 {
			verb = " wasn't imported. "
		}
		m.Lines = append(m.Lines, capital(notice.Count(f.Failed, item))+verb+"Open the Work page to see why.")
	}
	if f.Waiting > 0 {
		verb := " wait for you on the Work page."
		if f.Waiting == 1 {
			verb = " waits for you on the Work page."
		}
		m.Lines = append(m.Lines, capital(notice.Count(f.Waiting, item))+verb)
	}
	return m
}

var placedWords = map[PlacedBy]string{
	ByLink: "hard-linked", ByClone: "cloned", ByCopy: "copied", ByTransfer: "sent", ByMove: "moved", InPlace: "found",
}

// factsWords is the files' qualities in order, then their total size:
// "WEBDL-1080p · 6.2 GB".
func factsWords(fs []PlacedFile) string {
	var qualities []string
	var size int64
	for _, p := range fs {
		if q := p.Quality.Info().Name; !slices.Contains(qualities, q) {
			qualities = append(qualities, q)
		}
		size += p.Size
	}
	return strings.Join(append(qualities, release.Describe(release.Read{Facts: release.Facts{Size: release.Known(size)}})), " · ")
}

func qualitiesWords(qs []release.Quality) string {
	var names []string
	for _, q := range qs {
		if n := q.Info().Name; !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	return joinWords(names)
}

func itemCount(fs []PlacedFile) int {
	n := 0
	for _, p := range fs {
		n += len(p.Items)
	}
	return n
}

// itemsWords is the files' Items in words: "season 2, episodes 1 to 10",
// or with more than one level "episodes 9 and 10 of season 1 and episode 1
// of season 2".
func itemsWords(fs []PlacedFile, item works.Noun) string {
	var levels []string
	numbers := map[string][]int{}
	for _, p := range fs {
		for _, it := range p.Items {
			if _, ok := numbers[it.Level]; !ok {
				levels = append(levels, it.Level)
			}
			numbers[it.Level] = append(numbers[it.Level], it.Number)
		}
	}
	if len(levels) == 1 {
		ns := numbersWords(item, slices.Sorted(slices.Values(numbers[levels[0]])))
		if levels[0] == "" {
			return ns
		}
		return lowerFirst(levels[0]) + ", " + ns
	}
	parts := make([]string, len(levels))
	for i, l := range levels {
		parts[i] = numbersWords(item, slices.Sorted(slices.Values(numbers[l]))) + " of " + lowerFirst(l)
	}
	return joinWords(parts)
}

func itemAtWords(it ItemAt, item works.Noun) string {
	if it.Level == "" {
		return item.One + " " + strconv.Itoa(it.Number)
	}
	return lowerFirst(it.Level) + ", " + item.One + " " + strconv.Itoa(it.Number)
}

func lower(n works.Noun) works.Noun {
	return works.Noun{One: strings.ToLower(n.One), Many: strings.ToLower(n.Many)}
}

func capital(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}
