package imports

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Recent is Home's Recent: the newest finished Imports in words, and how
// many finished since a time, upgrades and imports apart.
type Recent struct {
	Arrivals           []Arrival
	Imported, Upgraded int
}

// Arrival is one finished Import, at its last landing.
type Arrival struct {
	Import ImportID
	Work   works.WorkID
	At     time.Time
	// Words is what it did: "Imported 10 episodes · 2 upgrades",
	// "Upgraded to Remux-2160p from WEBDL-1080p", "Imported in Remux-2160p".
	Words string
}

// recentPage is how many landings one step of Recent's walk reads.
const recentPage = 256

// Recent is the rows newest finished Imports, an Import finished once its
// record says so or is gone, and how many finished Imports landed a file
// at or after since. A conversion Import is never among them. It walks
// landings newest first, so it reads past the landings of unfinished and
// conversion Imports only.
func (im *Imports) Recent(ctx context.Context, r storage.Reader[file], rows int, since time.Time) (Recent, error) {
	kept, err := finishedNewest(ctx, r, rows)
	if err != nil {
		return Recent{}, err
	}
	var out Recent
	if out.Arrivals, err = im.arrivals(ctx, r, kept); err != nil {
		return Recent{}, err
	}
	tallies, err := files.TalliesSince(ctx, r, since)
	if err != nil {
		return Recent{}, err
	}
	ids := make([]storage.UUID, len(tallies))
	for i, t := range tallies {
		ids[i] = t.Import
	}
	skip, err := unrecentAmong(ctx, r, ids)
	if err != nil {
		return Recent{}, err
	}
	for _, t := range tallies {
		switch {
		case skip[t.Import]:
		case t.Upgraded:
			out.Upgraded++
		default:
			out.Imported++
		}
	}
	return out, nil
}

// finishedNewest is the ids of the n finished Imports whose last landing
// is newest, newest first.
func finishedNewest(ctx context.Context, r storage.Reader[file], n int) ([]storage.UUID, error) {
	var kept []storage.UUID
	seen := map[storage.UUID]bool{}
	var key files.RecordKey
	for len(kept) < n {
		page, err := files.RecordsBefore(ctx, r, key, recentPage)
		if err != nil {
			return nil, err
		}
		var fresh []storage.UUID
		for _, l := range page {
			if !seen[l.Import] {
				seen[l.Import] = true
				fresh = append(fresh, l.Import)
			}
		}
		skip, err := unrecentAmong(ctx, r, fresh)
		if err != nil {
			return nil, err
		}
		for _, id := range fresh {
			if !skip[id] && len(kept) < n {
				kept = append(kept, id)
			}
		}
		if len(page) < recentPage {
			break
		}
		key = page[len(page)-1].Key()
	}
	return kept, nil
}

// unrecentAmong is which of ids Recent leaves out: unfinished and
// conversion Imports.
func unrecentAmong(ctx context.Context, r storage.Reader[file], ids []storage.UUID) (map[storage.UUID]bool, error) {
	q, out := db.New(r), map[storage.UUID]bool{}
	for chunk := range slices.Chunk(ids, works.Batch) {
		of := make([]ImportID, len(chunk))
		for i, id := range chunk {
			of[i] = ImportID{UUID: id}
		}
		open, err := q.ListUnrecentAmong(ctx, of)
		if err != nil {
			return nil, err
		}
		for _, id := range open {
			out[id.UUID] = true
		}
	}
	return out, nil
}

// arrivals is each of ids, newest first, in words in its Work's nouns.
func (im *Imports) arrivals(ctx context.Context, r storage.Reader[file], ids []storage.UUID) ([]Arrival, error) {
	records, err := files.RecordsOf(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	of := map[storage.UUID][]files.ImportRecord{}
	var ws []works.WorkID
	for _, l := range records {
		of[l.Import] = append(of[l.Import], l)
		ws = append(ws, l.Work)
	}
	summaries, err := works.SummariesOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := make([]Arrival, 0, len(ids))
	for _, id := range ids {
		ls := of[id]
		last := ls[len(ls)-1]
		d, _ := im.cfg.Works.Describe(summaries[last.Work].ContentType)
		out = append(out, Arrival{Import: ImportID{UUID: id}, Work: last.Work, At: last.At, Words: arrivalWords(ls, d)})
	}
	return out, nil
}

// arrivalWords is what an Import's landings, oldest first, did, in d's
// nouns. An upgrade names the newest landing's quality, which is every
// landing's when they agree.
func arrivalWords(ls []files.ImportRecord, d works.Description) string {
	var replaced []release.Quality
	upgradedAll := true
	for _, l := range ls {
		if len(l.Replaced) == 0 || l.Quality == nil {
			upgradedAll = false
		}
		for _, p := range l.Replaced {
			replaced = append(replaced, p.Quality)
		}
	}
	newest := ls[len(ls)-1].Quality
	switch {
	case upgradedAll:
		return fmt.Sprintf("Upgraded to %s from %s", newest.Info().Name, qualitiesWords(replaced))
	case d.Items == works.ItemsOne && len(d.Levels) == 0 && newest != nil:
		return "Imported in " + newest.Info().Name
	}
	items, upgrades := map[works.ItemID]bool{}, map[works.ItemID]bool{}
	for _, l := range ls {
		for _, it := range l.Items {
			items[it] = true
			if len(l.Replaced) > 0 {
				upgrades[it] = true
			}
		}
	}
	noun := lower(d.Item)
	words := "Imported " + plural(len(items), noun)
	if len(upgrades) > 0 {
		words += " · " + plural(len(upgrades), works.Noun{One: "upgrade", Many: "upgrades"})
	}
	return words
}

func plural(n int, noun works.Noun) string {
	if n == 1 {
		return "1 " + noun.One
	}
	return fmt.Sprintf("%d %s", n, noun.Many)
}
