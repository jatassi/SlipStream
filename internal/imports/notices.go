package imports

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ImportedFacts is what an Imported or Upgraded message tells: the files
// an Import placed since it last came to rest, and how many of its files
// failed or wait for the Admin.
type ImportedFacts struct {
	Import ImportID `json:"import"`
	// Grab is zero for a folder Import.
	Grab    grabs.GrabID `json:"grab"`
	Work    entry.Work   `json:"work"`
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
	// ReplacedLabels are the replaced files' labels, for History's detail.
	ReplacedLabels []string `json:"replacedLabels,omitempty"`
}

// ItemAt is an Item by its place: its level's label, empty for a type
// with no levels, and its number in that level.
type ItemAt struct {
	Level  string `json:"level"`
	Number int    `json:"number"`
}

// ScanAddedFacts is what a Works added by a Library scan message tells.
// Works is every Work it added, for History, and isn't stored.
type ScanAddedFacts struct {
	Type  works.TypeID   `json:"type"`
	Works []works.WorkID `json:"-"`
	Root  string         `json:"root"`
	Nouns entry.Nouns    `json:"nouns"`
	Count int            `json:"count"`
	First []string       `json:"first"`
}

func (f ImportedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}, Grab: f.Grab.UUID, Import: f.Import.UUID}
}

func (f ScanAddedFacts) About() entry.About { return entry.About{Type: f.Type, Works: f.Works} }

// importNoun is Imported's and Upgraded's noun in History's 7-day
// sentence: each counts its Imports.
var importNoun = works.Noun{One: "Import", Many: "Imports"}

var (
	importedRow = entry.Row[ImportedFacts]{
		Spec:   entry.Spec{Kind: "imported", Label: "Imported", Group: entry.Imports, OnByDefault: true, BellByDefault: true, Tally: importNoun},
		Render: renderImported,
	}
	upgradedRow = entry.Row[ImportedFacts]{
		Spec:   entry.Spec{Kind: "upgraded", Label: "Upgraded", Group: entry.Imports, OnByDefault: true, BellByDefault: true, Tally: importNoun},
		Render: renderUpgraded,
	}
	scanAddedRow = entry.Row[ScanAddedFacts]{
		Spec:   entry.Spec{Kind: "works_added_by_scan", Label: "Works added by a Library scan", Group: entry.Library, BellByDefault: true},
		Render: renderScanAdded,
	}
)

// Entries is the kinds imports records.
func Entries() []entry.Entry { return []entry.Entry{importedRow, upgradedRow, scanAddedRow} }

// announce records Imported and Upgraded for the files imp placed since it
// last came to rest. A root Import never sends: Pending imports' Add and
// in-place adoption both come through one. Neither does an Import that
// failed, or that its Work's delete ended.
func (im *Imports) announce(ctx context.Context, tx *storage.Tx[file], imp Import) error {
	if sourceRules[imp.Source].announce == none || !imp.atRest() || imp.endedByDelete() {
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
	w, err := entry.WorkIn(ctx, tx, im.cfg.Works, imp.Work)
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
	in, up := splitPlaced(ImportedFacts{Import: imp.ID, Grab: imp.Grab, Work: w, Root: words[fresh[0].Dest.Root]}, fresh, replaced, numbersOf(tree), imp.Files)
	if in != nil {
		if err := importedRow.Record(ctx, tx, im.cfg.History, *in); err != nil {
			return err
		}
	}
	if up != nil {
		if err := upgradedRow.Record(ctx, tx, im.cfg.History, *up); err != nil {
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
	places := works.Places(tree.Items)
	out := make(map[works.ItemID]ItemAt, len(tree.Items))
	for _, it := range tree.Items {
		out[it.ID] = ItemAt{Level: labels[it.LevelID], Number: places[it.ID]}
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
			p.ReplacedLabels = append(p.ReplacedLabels, r.Label)
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
	return scanAddedRow.Record(ctx, tx, im.cfg.History, ScanAddedFacts{
		Type: root.ContentType, Works: in, Root: words[root.ID], Nouns: entry.NounsOf(im.cfg.Works, root.ContentType), Count: len(in), First: works.LabelsOf(added, in, 3),
	})
}

func renderImported(f ImportedFacts, _ entry.Context) entry.Message {
	m := placedMessage(f, entry.Good)
	if f.Work.Nouns.Single {
		p := f.Files[0]
		m.Title = "Imported " + f.Work.Label
		m.Lines = []string{factsWords(f.Files), fmt.Sprintf(placedWords[p.PlacedBy], f.Root)}
		return m
	}
	item := lower(f.Work.Nouns.Item)
	m.Title = "Imported " + entry.Count(itemCount(f.Files), item) + " of " + f.Work.Label
	m.Lines = []string{capital(itemsWords(f.Files, item)) + ".", factsWords(f.Files)}
	return withLeft(m, f, item)
}

func renderUpgraded(f ImportedFacts, _ entry.Context) entry.Message {
	m := placedMessage(f, entry.Good)
	if f.Work.Nouns.Single {
		p := f.Files[0]
		m.Title = "Upgraded " + f.Work.Label + " to " + p.Quality.Info().Name
		m.Lines = []string{"It replaced " + qualitiesWords(p.Replaced) + "."}
		return m
	}
	item := lower(f.Work.Nouns.Item)
	n := itemCount(f.Files)
	m.Title = "Upgraded " + entry.Count(n, item) + " of " + f.Work.Label
	shown := f.Files[:min(len(f.Files), 3)]
	for _, p := range shown {
		m.Lines = append(m.Lines, capital(itemsWords([]PlacedFile{p}, item))+" went from "+qualitiesWords(p.Replaced)+" to "+p.Quality.Info().Name+".")
	}
	switch rest := n - itemCount(shown); {
	case rest == 1:
		m.Lines = append(m.Lines, "1 more "+item.One+" was upgraded.")
	case rest > 1:
		m.Lines = append(m.Lines, setting.Grouped(rest)+" more "+item.Many+" were upgraded.")
	}
	return withLeft(m, f, item)
}

func renderScanAdded(f ScanAddedFacts, _ entry.Context) entry.Message {
	return entry.Message{
		Title: "A Library scan of the " + f.Root + " added " + entry.Count(f.Count, lower(f.Nouns.Work)),
		Lines: entry.Titles(f.First, f.Count, "added"),
		Link:  entry.Link{Path: "/library"},
		Tone:  entry.Info,
	}
}

// placedMessage is f's Work and files. A Work of one Item lists its files
// with no Item, and leaves what each replaced to the entry's own line.
func placedMessage(f ImportedFacts, tone entry.Tone) entry.Message {
	m := entry.Message{Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: tone}
	m.Body.Work = &entry.BodyWork{ID: f.Work.ID.String(), Title: f.Work.Label, ContentType: string(f.Work.Type)}
	for _, p := range f.Files {
		b := entry.BodyFile{Quality: p.Quality.Info().Name, Replaced: qualitiesWords(p.Replaced), PlacedBy: string(p.PlacedBy)}
		for _, it := range p.Items {
			b.Items = append(b.Items, itemAtWords(it, lower(f.Work.Nouns.Item)))
		}
		m.Body.Files = append(m.Body.Files, b)
		d := entry.DetailFile{Items: []string{}, Quality: b.Quality, PlacedBy: b.PlacedBy}
		if !f.Work.Nouns.Single {
			d.Items, d.Replaced, d.ReplacedLabel = b.Items, b.Replaced, works.JoinWords(p.ReplacedLabels)
		}
		m.Detail.Files = append(m.Detail.Files, d)
	}
	return m
}

// withLeft adds the sentences for the files that failed or wait.
func withLeft(m entry.Message, f ImportedFacts, item works.Noun) entry.Message {
	page := " the " + lower(f.Work.Nouns.Work).One + " page"
	if f.Failed > 0 {
		verb := " weren't imported. "
		if f.Failed == 1 {
			verb = " wasn't imported. "
		}
		m.Lines = append(m.Lines, capital(entry.Count(f.Failed, item))+verb+"Open"+page+" to see why.")
	}
	if f.Waiting > 0 {
		verb := " wait for you on"
		if f.Waiting == 1 {
			verb = " waits for you on"
		}
		m.Lines = append(m.Lines, capital(entry.Count(f.Waiting, item))+verb+page+".")
	}
	return m
}

// placedWords takes the library root's words.
var placedWords = map[PlacedBy]string{
	ByLink:     "SlipStream hard-linked it into the %s.",
	ByClone:    "SlipStream cloned it into the %s.",
	ByCopy:     "SlipStream copied it into the %s.",
	ByTransfer: "SlipStream transferred it from another Node into the %s.",
	ByMove:     "SlipStream moved it into the %s.",
	InPlace:    "SlipStream found it in the %s.",
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
	return strings.Join(append(qualities, release.Summary{Size: release.Known(size)}.Words()), " · ")
}

func qualitiesWords(qs []release.Quality) string {
	var names []string
	for _, q := range qs {
		if n := q.Info().Name; !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	return works.JoinWords(names)
}

func itemCount(fs []PlacedFile) int {
	n := 0
	for _, p := range fs {
		n += len(p.Items)
	}
	return n
}

// itemsWords is the files' Items in words: "season 2, episodes 1 to 10",
// or with more than one level "episodes 9 and 10 of season 1, plus episode
// 1 of season 2". The levels join with "plus" so a level's own "and"
// isn't read as theirs.
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
		ns := works.NumbersWords(item, slices.Sorted(slices.Values(numbers[levels[0]])))
		if levels[0] == "" {
			return ns
		}
		return lowerFirst(levels[0]) + ", " + ns
	}
	parts := make([]string, len(levels))
	for i, l := range levels {
		parts[i] = works.NumbersWords(item, slices.Sorted(slices.Values(numbers[l]))) + " of " + lowerFirst(l)
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", plus " + parts[len(parts)-1]
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
