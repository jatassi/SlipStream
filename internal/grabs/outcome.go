package grabs

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	ReleaseGrabbed ReleaseOutcome = "grabbed"
	// ReleaseNotTaken ranked above the grabbed Release, and its Indexer
	// stopped, reached its limit or dropped it.
	ReleaseNotTaken  ReleaseOutcome = "not_taken"
	ReleaseOutranked ReleaseOutcome = "outranked"
	// ReleaseKept ranked, and the run took no Grab.
	ReleaseKept    ReleaseOutcome = "kept"
	ReleaseRefused ReleaseOutcome = "refused"
	// ReleaseListed is a Manual run's Release: stored as heard, judged on
	// read.
	ReleaseListed ReleaseOutcome = "listed"
)

// TargetScore is a Release's score at one Target.
type TargetScore struct {
	Target targets.TargetID
	Score  int
}

// row is one listed Release as its run keeps it. rank is its place among
// the ranked Releases from 1, 0 when it is grabbable at no cell, and
// copyOf the position of the Release it is a copy of, -1 for none.
type row struct {
	release indexers.Release
	at      int
	copyOf  int
	rank    int
	outcome ReleaseOutcome
	reasons []Reason
	loss    *Loss
	scores  []TargetScore
	formats formats.Set
	covers  []works.ItemID
}

// offers is j, then j as each of its copies, in the order a search
// offers them.
func (j judged) offers() []judged {
	out := []judged{j}
	for _, c := range j.copies {
		o := j
		o.Release, o.at = c.Release, c.at
		out = append(out, o)
	}
	return out
}

// outcomes is every Release d judged, copies included, by position, with
// its outcome. taken is the position of the Release the run grabbed, or
// of its copy, -1 for none. An outranked or kept Release says where it
// lost to the reference: the grabbed Release as its best Indexer listed
// it, else the first a search may take.
func outcomes(d Decision, c cells, taken int) []row {
	var ref *judged
	for i, j := range d.Ranked {
		grabbed := slices.ContainsFunc(j.offers(), func(o judged) bool { return o.at == taken })
		if len(j.refusals) == 0 && (grabbed || taken < 0 && ref == nil) {
			ref = &d.Ranked[i]
		}
	}
	var rows []row
	passed := false
	for i, j := range d.Ranked {
		for _, o := range j.offers() {
			r := rowOf(j, o)
			r.rank = i + 1
			switch {
			case len(j.refusals) > 0:
				r.outcome = ReleaseRefused
			case taken < 0:
				r.outcome = ReleaseKept
			case o.at == taken:
				r.outcome, passed = ReleaseGrabbed, true
			case !passed:
				r.outcome = ReleaseNotTaken
			default:
				r.outcome = ReleaseOutranked
			}
			if r.outcome == ReleaseKept || r.outcome == ReleaseOutranked {
				r.loss = lostAt(c, ref, &o)
			}
			rows = append(rows, r)
		}
	}
	for _, j := range d.Refused {
		for _, o := range j.offers() {
			r := rowOf(j, o)
			r.outcome = ReleaseRefused
			rows = append(rows, r)
		}
	}
	slices.SortFunc(rows, func(a, b row) int { return cmp.Compare(a.at, b.at) })
	return rows
}

// rowOf is o, j or one of its copies, with j's reasons, scores, formats
// and covered Items.
func rowOf(j, o judged) row {
	r := row{release: o.Release, at: o.at, copyOf: -1, reasons: j.reasons(), formats: j.formats.Matched, covers: j.covers}
	if o.at != j.at {
		r.copyOf = j.at
	}
	for _, v := range j.verdicts {
		if !slices.ContainsFunc(r.scores, func(s TargetScore) bool { return s.Target == v.target }) {
			r.scores = append(r.scores, TargetScore{Target: v.target, Score: v.v.Score()})
		}
	}
	return r
}

// storeReleases inserts rows as run's Releases, with names[at] the name
// of the Indexer that listed the Release at. While a rerun may take them,
// rerunnable, each keeps its Release, and its link unless it is the
// grabbed one or gone, taken by an earlier Grab of the run.
func (gr *Grabs) storeReleases(ctx context.Context, q *db.Queries, run SearchID, rows []row, names map[int]string, rerunnable bool, gone map[int]bool) error {
	for _, r := range rows {
		p, err := releaseRow(run, r)
		if err != nil {
			return err
		}
		p.IndexerName = names[r.at]
		if rerunnable {
			k, err := encodeKept(r.release)
			if err != nil {
				return err
			}
			p.Release = &k
			if r.outcome != ReleaseGrabbed && !gone[r.at] {
				p.LinkSealed = gr.cfg.Sealer.Seal([]byte(r.release.Offered().Reveal()), keptLinkAD(run, r.at))
			}
		}
		if err := q.InsertRelease(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// storedScore is a TargetScore as grabs_release.scores keeps it.
type storedScore struct {
	TargetID string `json:"targetId"`
	Score    int    `json:"score"`
}

func releaseRow(run SearchID, r row) (db.InsertReleaseParams, error) {
	rel := r.release
	p := db.InsertReleaseParams{
		SearchID: run, Position: int64(r.at), Title: rel.Title, IndexerID: rel.Indexer, Size: rel.Size, Seeders: int64(rel.Seeders),
		Outcome: r.outcome,
	}
	if r.rank > 0 {
		p.Rank = new(int64(r.rank))
	}
	if r.copyOf >= 0 {
		p.CopyOf = new(int64(r.copyOf))
	}
	if l := r.loss; l != nil {
		p.LostStep, p.LostTargetID, p.LostOwn, p.LostRef = &l.Step, l.Target, &l.Own, &l.Ref
	}
	var err error
	if p.Reasons, err = encodeReasons(r.reasons); err != nil {
		return p, err
	}
	scores := make([]storedScore, len(r.scores))
	for i, s := range r.scores {
		scores[i] = storedScore{TargetID: s.Target.String(), Score: s.Score}
	}
	ids := func(n int, at func(int) string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = at(i)
		}
		return out
	}
	for _, f := range []struct {
		to *string
		v  any
	}{
		{&p.Scores, scores},
		{&p.Formats, ids(len(r.formats), func(i int) string { return r.formats[i].String() })},
		{&p.Covers, ids(len(r.covers), func(i int) string { return r.covers[i].String() })},
	} {
		b, err := json.Marshal(f.v)
		if err != nil {
			return p, fmt.Errorf("encode a Release row: %w", err)
		}
		*f.to = string(b)
	}
	return p, nil
}

// Listed is one Release a run listed, as its row keeps it. Rank is 0 for
// a Release grabbable at no cell and CopyOf -1 for one that is no copy.
// Words is its facts in words, read again from its title, and its
// reasons' words are written from that read.
type Listed struct {
	Position    int
	Title       string
	Indexer     indexers.ID
	IndexerName string
	Size        int64
	Seeders     int
	Outcome     ReleaseOutcome
	Rank        int
	CopyOf      int
	Words       string
	Covers      []works.ItemID
	Scores      []TargetScore
	Formats     []formats.ID
	Reasons     []Worded
	LostAt      *Loss
}

var ErrNoSearch = errors.New("no search run has that id")

// Releases is run's Releases, ranked first in rank order, then the rest
// by position. A run keeps them while it is its Item's latest finished
// run, or its Grab still fetches, and lists none after.
func (gr *Grabs) Releases(ctx context.Context, by identity.Admin, run SearchID) ([]Listed, error) {
	all, err := gr.cfg.Targets.ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	byID := make(map[targets.TargetID]*targets.Target, len(all))
	for i := range all {
		byID[all[i].ID] = &all[i]
	}
	var out []Listed
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		s, err := q.GetSearch(ctx, run)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoSearch
		}
		if err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, s.WorkID)
		if err != nil {
			return err
		}
		known, err := gr.cfg.Targets.CutsIn(ctx, r)
		if err != nil {
			return err
		}
		names, cuts := gr.cfg.Names[tree.ContentType], known.For(tree.ContentType)
		rows, err := q.ListReleasesOfSearch(ctx, run)
		if err != nil {
			return err
		}
		for _, row := range rows {
			l, reasons, err := releaseOf(row)
			if err != nil {
				return fmt.Errorf("search %s: Release %d: %w", run, row.Position, err)
			}
			out = append(out, l.inWords(reasons, release.Title(l.Title, names, cuts), byID))
		}
		return nil
	})
	return out, err
}

// inWords is l with its facts, read as rd with its size, and its reasons,
// at the Targets of byID, in words.
func (l Listed) inWords(reasons []Reason, rd release.Read, byID map[targets.TargetID]*targets.Target) Listed {
	if l.Size > 0 {
		rd.Facts.Size = release.Known(l.Size)
	}
	l.Words = release.Describe(rd)
	for _, r := range reasons {
		l.Reasons = append(l.Reasons, Worded{Reason: r, Words: reasonWords(r, byID[r.Target], rd.Facts)})
	}
	return l
}

// releaseOf is row as a Listed with no words, and its reasons.
func releaseOf(row db.GrabsRelease) (Listed, []Reason, error) {
	l := Listed{
		Position: int(row.Position), Title: row.Title, Indexer: row.IndexerID, IndexerName: row.IndexerName, Size: row.Size,
		Seeders: int(row.Seeders), Outcome: row.Outcome, CopyOf: -1,
	}
	if row.Rank != nil {
		l.Rank = int(*row.Rank)
	}
	if row.CopyOf != nil {
		l.CopyOf = int(*row.CopyOf)
	}
	if row.LostStep != nil {
		l.LostAt = &Loss{Step: *row.LostStep, Target: row.LostTargetID, Own: *row.LostOwn, Ref: *row.LostRef}
	}
	reasons, err := decodeReasons(row.Reasons)
	if err != nil {
		return l, nil, err
	}
	var scores []storedScore
	var fs, covers []string
	for _, f := range []struct {
		from string
		to   any
	}{{row.Scores, &scores}, {row.Formats, &fs}, {row.Covers, &covers}} {
		if err := json.Unmarshal([]byte(f.from), f.to); err != nil {
			return l, nil, fmt.Errorf("decode: %w", err)
		}
	}
	for _, s := range scores {
		u, err := storage.ParseUUID(s.TargetID)
		if err != nil {
			return l, nil, err
		}
		l.Scores = append(l.Scores, TargetScore{Target: targets.TargetID{UUID: u}, Score: s.Score})
	}
	for _, s := range fs {
		id, err := targets.ParseFormatID(s)
		if err != nil {
			return l, nil, err
		}
		l.Formats = append(l.Formats, id)
	}
	for _, s := range covers {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return l, nil, err
		}
		l.Covers = append(l.Covers, works.ItemID{UUID: u})
	}
	return l, reasons, nil
}
